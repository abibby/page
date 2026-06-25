package bookmeta

import (
	"fmt"
	"os"

	"github.com/dhowden/tag"
)

func extractMP3(path string) (Meta, error) {
	f, err := os.Open(path)
	if err != nil {
		return Meta{}, fmt.Errorf("open mp4: %w", err)
	}
	defer f.Close()
	m, err := tag.ReadFrom(f)
	if err != nil {
		return Meta{}, err
	}

	author := ""
	if m.Album() != m.Artist() {
		author = m.Artist()
	}

	// fmt.Println(
	// 	"\nFormat:", m.Format(),
	// 	"\nFileType:", m.FileType(),
	// 	"\nTitle:", m.Title(),
	// 	"\nAlbum:", m.Album(),
	// 	"\nArtist:", m.Artist(),
	// 	"\nAlbumArtist:", m.AlbumArtist(),
	// 	"\nComposer:", m.Composer(),
	// 	"\nYear:", m.Year(),
	// 	"\nGenre:", m.Genre(),
	// 	// "\nTrack:", m.Track(),
	// 	// "\nDisc:", m.Disc(),
	// 	"\nPicture:", m.Picture(),
	// 	"\nLyrics:", m.Lyrics(),
	// 	"\nComment:", m.Comment(),
	// )
	return Meta{
		ISBN:        "",
		Title:       m.Album(),
		Author:      author,
		IsAudiobook: true,
	}, nil
}
