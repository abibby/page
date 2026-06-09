package bookmeta

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

// extractMP4 walks the MP4/iTunes atom tree (moov > udta > meta > ilst) of an
// M4B/M4A file and collects metadata, then scans it for an ISBN.
//
// The ilst children are metadata items keyed by atom type, e.g. "©nam" (title)
// and "©ART" (artist/author). "----" items are freeform (mean/name/data); their
// human key comes from the "name" sub-atom. Many audiobook taggers stash an
// ISBN in the comment ("©cmt"), description ("desc"), or a freeform atom, so we
// scan every collected value.
func extractMP4(path string) (Meta, error) {
	f, err := os.Open(path)
	if err != nil {
		return Meta{}, fmt.Errorf("open mp4: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return Meta{}, err
	}

	tags := map[string]string{}
	if err := walkContainer(f, 0, info.Size(), false, tags); err != nil {
		return Meta{}, fmt.Errorf("parse mp4 atoms: %w", err)
	}

	// iTunes atom keys begin with the 0xA9 byte (e.g. 0xA9 'n' 'a' 'm'), not the
	// UTF-8 copyright sign, so these are written with explicit \xa9 escapes.
	meta := Meta{
		Title:  firstNonEmpty(tags["\xa9nam"], tags["title"]),
		Author: firstNonEmpty(tags["\xa9ART"], tags["aART"], tags["author"], tags["artist"]),
	}

	// Scan every value plus the freeform keys for an ISBN.
	values := make([]string, 0, len(tags))
	for k, v := range tags {
		values = append(values, v, k)
	}
	meta.ISBN = findISBN(values...)
	return meta, nil
}

const (
	maxAtomDepth = 12
	// metaFullBox: the iTunes-style "meta" atom is a FullBox with a 4-byte
	// version/flags field before its children.
	fullBoxHeader = 4
)

// walkContainer iterates the boxes in [start,end). When inIlst is true each
// child box is treated as a metadata item rather than a container.
func walkContainer(r io.ReaderAt, start, end int64, inIlst bool, tags map[string]string) error {
	return walkDepth(r, start, end, inIlst, tags, 0)
}

func walkDepth(r io.ReaderAt, start, end int64, inIlst bool, tags map[string]string, depth int) error {
	if depth > maxAtomDepth {
		return nil
	}
	pos := start
	for pos+8 <= end {
		size, typ, headerLen, err := readBoxHeader(r, pos)
		if err != nil {
			return err
		}
		if size == 0 { // box extends to end of file/parent
			size = end - pos
		}
		payloadStart := pos + int64(headerLen)
		payloadEnd := pos + size
		if size < int64(headerLen) || payloadEnd > end {
			break // malformed or truncated; stop scanning this level
		}

		if inIlst {
			parseItem(r, payloadStart, payloadEnd, typ, tags)
		} else {
			switch typ {
			case "moov", "udta", "trak", "mdia", "minf", "stbl":
				_ = walkDepth(r, payloadStart, payloadEnd, false, tags, depth+1)
			case "meta":
				// FullBox: skip 4-byte version/flags, unless this file uses the
				// QuickTime variant where children start immediately.
				childStart := payloadStart
				if !looksLikeBoxHeader(r, payloadStart) && looksLikeBoxHeader(r, payloadStart+fullBoxHeader) {
					childStart += fullBoxHeader
				}
				_ = walkDepth(r, childStart, payloadEnd, false, tags, depth+1)
			case "ilst":
				_ = walkDepth(r, payloadStart, payloadEnd, true, tags, depth+1)
			}
		}
		pos = payloadEnd
	}
	return nil
}

// parseItem reads an ilst child: its "data" sub-atom holds the value, and for
// freeform "----" atoms the "name" sub-atom holds the key.
func parseItem(r io.ReaderAt, start, end int64, key string, tags map[string]string) {
	var value, name string
	pos := start
	for pos+8 <= end {
		size, typ, headerLen, err := readBoxHeader(r, pos)
		if err != nil || size < int64(headerLen) || pos+size > end {
			break
		}
		payloadStart := pos + int64(headerLen)
		payloadEnd := pos + size
		switch typ {
		case "data":
			// data payload: 4-byte type indicator, 4-byte locale, then value.
			if payloadEnd-payloadStart > 8 {
				value = readString(r, payloadStart+8, payloadEnd)
			}
		case "name":
			// FullBox: 4-byte version/flags then the freeform key name.
			if payloadEnd-payloadStart > 4 {
				name = readString(r, payloadStart+4, payloadEnd)
			}
		}
		pos = payloadEnd
	}
	if value == "" {
		return
	}
	if key == "----" && name != "" {
		tags[strings.ToLower(name)] = value
	} else {
		tags[key] = value
	}
}

// readBoxHeader reads the 8- or 16-byte box header at off, returning the total
// box size, the 4-char type, and the header length.
func readBoxHeader(r io.ReaderAt, off int64) (size int64, typ string, headerLen int, err error) {
	var hdr [8]byte
	if _, err = r.ReadAt(hdr[:], off); err != nil {
		return 0, "", 0, err
	}
	size = int64(binary.BigEndian.Uint32(hdr[:4]))
	typ = string(hdr[4:8])
	headerLen = 8
	if size == 1 { // 64-bit largesize follows
		var large [8]byte
		if _, err = r.ReadAt(large[:], off+8); err != nil {
			return 0, "", 0, err
		}
		size = int64(binary.BigEndian.Uint64(large[:]))
		headerLen = 16
	}
	return size, typ, headerLen, nil
}

// looksLikeBoxHeader reports whether the 8 bytes at off plausibly start a box
// (printable ASCII type, sane size). Used to detect the meta FullBox variant.
func looksLikeBoxHeader(r io.ReaderAt, off int64) bool {
	size, typ, _, err := readBoxHeader(r, off)
	if err != nil || size < 8 {
		return false
	}
	for i := 0; i < len(typ); i++ {
		c := typ[i]
		if c < 0x20 || c > 0x7e {
			return false
		}
	}
	return true
}

func readString(r io.ReaderAt, start, end int64) string {
	n := end - start
	if n <= 0 || n > 1<<20 { // cap absurd sizes
		return ""
	}
	buf := make([]byte, n)
	if _, err := r.ReadAt(buf, start); err != nil && err != io.EOF {
		return ""
	}
	return strings.TrimRight(string(buf), "\x00")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
