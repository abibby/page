package bookmeta

import (
	"fmt"
	"os"

	"github.com/dhowden/tag"
)

func extractAudio(path string) (Meta, error) {
	f, err := os.Open(path)
	if err != nil {
		return Meta{}, fmt.Errorf("extractAudio: open file: %w", err)
	}
	defer f.Close()
	m, err := tag.ReadFrom(f)
	if err != nil {
		return Meta{}, fmt.Errorf("extractAudio: read tags: %w", err)
	}

	title := m.Title()
	if title == "" {
		title = m.Album()
	}

	author := ""
	if title != m.Artist() {
		author = m.Artist()
	}

	fmt.Println(
		"\nFormat:", m.Format(),
		"\nFileType:", m.FileType(),
		"\nTitle:", m.Title(),
		"\nAlbum:", m.Album(),
		"\nArtist:", m.Artist(),
		"\nAlbumArtist:", m.AlbumArtist(),
		"\nComposer:", m.Composer(),
		"\nYear:", m.Year(),
		"\nGenre:", m.Genre(),
		// "\nTrack:", m.Track(),
		// "\nDisc:", m.Disc(),
		"\nPicture:", m.Picture(),
		"\nLyrics:", m.Lyrics(),
		"\nComment:", m.Comment(),
		"\nRaw:", m.Raw(),
	)
	return Meta{
		ISBN:        "",
		Title:       title,
		Author:      author,
		IsAudiobook: true,
	}, nil
}
