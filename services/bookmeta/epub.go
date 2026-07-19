package bookmeta

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// containerXML maps META-INF/container.xml to the OPF package path.
type containerXML struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

// opfPackage is the subset of the OPF metadata we read.
type opfPackage struct {
	Identifiers []struct {
		Scheme string `xml:"scheme,attr"`
		Value  string `xml:",chardata"`
	} `xml:"metadata>identifier"`
	Title    string   `xml:"metadata>title"`
	Creators []string `xml:"metadata>creator"`
}

// extractEPUB reads the OPF metadata embedded in an EPUB (a ZIP archive).
func extractEPUB(path string) (Meta, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return Meta{}, fmt.Errorf("open epub: %w", err)
	}
	defer r.Close() //nolint:errcheck

	opfPath, err := opfPathFromContainer(r)
	if err != nil {
		return Meta{}, err
	}

	pkg, err := readOPF(r, opfPath)
	if err != nil {
		return Meta{}, err
	}

	meta := Meta{
		Title:       strings.TrimSpace(pkg.Title),
		Author:      strings.TrimSpace(strings.Join(pkg.Creators, " & ")),
		IsAudiobook: false,
	}

	// Prefer an identifier explicitly scheme-tagged as ISBN, then fall back to
	// scanning all identifier values for an ISBN pattern.
	var candidates []string
	for _, id := range pkg.Identifiers {
		if strings.Contains(strings.ToLower(id.Scheme), "isbn") {
			candidates = append([]string{id.Value}, candidates...)
		} else {
			candidates = append(candidates, id.Value)
		}
	}
	meta.ISBN = findISBN(candidates...)
	return meta, nil
}

func opfPathFromContainer(r *zip.ReadCloser) (string, error) {
	for _, f := range r.File {
		if f.Name != "META-INF/container.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close() //nolint:errcheck
		var c containerXML
		if err := xml.NewDecoder(rc).Decode(&c); err != nil {
			return "", fmt.Errorf("parse container.xml: %w", err)
		}
		if len(c.Rootfiles) > 0 && c.Rootfiles[0].FullPath != "" {
			return c.Rootfiles[0].FullPath, nil
		}
	}
	return "", fmt.Errorf("no OPF rootfile found in epub")
}

func readOPF(r *zip.ReadCloser, opfPath string) (opfPackage, error) {
	for _, f := range r.File {
		if f.Name != opfPath {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return opfPackage{}, err
		}
		defer rc.Close() //nolint:errcheck
		data, err := io.ReadAll(rc)
		if err != nil {
			return opfPackage{}, err
		}
		var pkg opfPackage
		if err := xml.Unmarshal(data, &pkg); err != nil {
			return opfPackage{}, fmt.Errorf("parse opf: %w", err)
		}
		return pkg, nil
	}
	return opfPackage{}, fmt.Errorf("opf file %q not found in epub", opfPath)
}
