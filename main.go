// Command page watches qBittorrent for completed downloads tagged "book",
// enriches them with metadata from Hardcover (by ISBN, with a title/author
// search fallback), and imports the EPUB/M4B files into a Calibre library.
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/abibby/page/app"
	"github.com/abibby/page/internal/calibre"
	"github.com/abibby/page/internal/calibredb"
	"github.com/abibby/page/internal/config"
	"github.com/abibby/page/internal/hardcover"
)

func main() {
	envFile := flag.String("env", ".env", "path to the .env file")
	once := flag.Bool("once", false, "run a single pass and exit instead of looping")
	flag.Parse()

	cfg, err := config.Load(*envFile)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	hc := hardcover.New(cfg.HardcoverURL, cfg.HardcoverToken)
	client := calibredb.NewClient(cfg.CalibredbBin, &calibredb.GlobalFlags{
		LibraryPath: cfg.CalibreLibrary,
	})
	importer := calibre.NewClient(cfg, client)

	a := app.New(cfg, hc, importer)

	ctx := context.Background()

	if *once {
		if err := a.RunPass(ctx); err != nil {
			log.Fatalf("run: %v", err)
		}
		return
	}

	log.Printf("watching qBittorrent at %s for tag %q every %s", cfg.QbitURL, cfg.QbitTag, cfg.PollInterval)
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	if err := a.RunPass(ctx); err != nil {
		log.Printf("pass error: %v", err)
	}
	for {
		select {
		case <-ctx.Done():
			log.Println("shutting down")
			return
		case <-ticker.C:
			if err := a.RunPass(ctx); err != nil {
				log.Printf("pass error: %v", err)
			}
		}
	}
}
