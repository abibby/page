package bookmeta

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"google.golang.org/genai"
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

	meta, err := aiExtract(path, m)
	if err != nil {
		return Meta{}, fmt.Errorf("extractAudio: ai: %w", err)
	}
	meta.ISBN = ""
	meta.IsAudiobook = true

	return meta, nil
}
func fileNameWithoutExtension(fileName string) string {
	return fileName[:len(fileName)-len(filepath.Ext(fileName))]
}

var cache = map[string]Meta{}

func aiExtract(path string, m tag.Metadata) (Meta, error) {
	cacheKey := m.Album() + "|" + m.Artist()

	meta, ok := cache[cacheKey]
	if ok {
		return meta, nil
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return Meta{}, err
	}
	parts := []*genai.Part{
		{Text: fmt.Sprintf(
			"return the title and author if this autiobook file in a json object with the keys title and author. The title should not have any subtitles or series information with it. File path: %s, file metadata: title: %s, album: %s, artist: %s",
			path,
			m.Title(),
			m.Album(),
			m.Artist(),
		)},
	}
	result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", []*genai.Content{{Parts: parts}}, nil)
	if err != nil {
		return Meta{}, err
	}

	resultText := result.Text()

	startTag := "```json"
	endTag := "```"

	resultJSON := resultText[strings.Index(resultText, startTag)+len(startTag) : strings.LastIndex(resultText, endTag)]

	meta = Meta{}
	err = json.Unmarshal([]byte(resultJSON), &meta)
	if err != nil {
		return Meta{}, err
	}
	cache[cacheKey] = meta
	go func() {
		time.Sleep(time.Minute)
		delete(cache, cacheKey)
	}()
	return meta, nil
}
