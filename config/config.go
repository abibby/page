package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/abibby/salusa/database"
	"github.com/abibby/salusa/database/dialects/sqlite"
	"github.com/abibby/salusa/env"
	"github.com/abibby/salusa/event"
	"github.com/joho/godotenv"
)

type Config struct {
	Port     int
	BasePath string

	Database database.Config
	Queue    event.Config

	// qBittorrent WebUI.
	QbitURL      string
	QbitUsername string
	QbitPassword string
	QbitTag      string
	QbitDoneTag  string
	QbitErrorTag string

	// Calibre.
	CalibreLibrary string
	// CalibreServer   string
	// CalibreUsername string
	// CalibrePassword string
	CalibredbBin  string
	AddDuplicates bool

	// Hardcover GraphQL API.
	HardcoverURL   string
	HardcoverToken string

	// Daemon behaviour.
	PollInterval time.Duration
	StateFile    string
	DryRun       bool

	GeminiAPIKey string

	JackettUrl    string
	JackettApiKey string

	// pathMaps remap container paths (as reported by qBittorrent running in
	// Docker) to paths reachable from this host. Sorted longest-prefix first.
	pathMaps []pathMap
}

type pathMap struct {
	container string
	host      string
}

func Load() *Config {
	err := godotenv.Load("./.env")
	if errors.Is(err, os.ErrNotExist) {
		// fall through
	} else if err != nil {
		panic(err)
	}

	c := &Config{
		Port:     env.Int("PORT", 2303),
		BasePath: env.String("BASE_PATH", ""),
		Database: sqlite.NewConfig(env.String("DATABASE_PATH", "./db.sqlite")),
		Queue:    event.NewChannelQueueConfig(),

		QbitURL:        env.String("QBIT_URL", "http://localhost:8080"),
		QbitUsername:   os.Getenv("QBIT_USERNAME"),
		QbitPassword:   os.Getenv("QBIT_PASSWORD"),
		QbitTag:        env.String("QBIT_TAG", "book"),
		QbitDoneTag:    env.String("QBIT_DONE_TAG", "done"),
		QbitErrorTag:   env.String("QBIT_ERROR_TAG", "error"),
		CalibreLibrary: os.Getenv("CALIBRE_LIBRARY"),
		CalibredbBin:   env.String("CALIBREDB_BIN", "calibredb"),
		AddDuplicates:  env.Bool("ADD_DUPLICATES", false),
		HardcoverURL:   env.String("HARDCOVER_URL", "https://api.hardcover.app/v1/graphql"),
		HardcoverToken: os.Getenv("HARDCOVER_TOKEN"),
		GeminiAPIKey:   os.Getenv("GEMINI_API_KEY"),
		JackettUrl:     os.Getenv("JACKETT_URL"),
		JackettApiKey:  os.Getenv("JACKETT_API_KEY"),
		StateFile:      env.String("STATE_FILE", "state.json"),
		DryRun:         env.Bool("DRY_RUN", false),
	}
	c.PollInterval, err = time.ParseDuration(env.String("POLL_INTERVAL", "5m"))
	if err != nil {
		log.Fatalf("failed to load POLL_INTERVAL: %v", err)
	}

	c.pathMaps, err = parsePathMap(os.Getenv("PATH_MAP"))
	if err != nil {
		log.Fatalf("failed to load PATH_MAP: %v", err)
	}

	return c
}

func (c *Config) GetHTTPPort() int {
	return c.Port
}
func (c *Config) GetBaseURL() string {
	return c.BasePath
}

func (c *Config) DBConfig() database.Config {
	return c.Database
}
func (c *Config) QueueConfig() event.Config {
	return c.Queue
}

// RemapPath rewrites a container path to a host path using the configured
// mappings (longest matching prefix wins). Paths with no matching mapping are
// returned unchanged.
func (c *Config) RemapPath(p string) string {
	for _, m := range c.pathMaps {
		if p == m.container || strings.HasPrefix(p, m.container+"/") {
			return m.host + strings.TrimPrefix(p, m.container)
		}
	}
	return p
}

// parsePathMap parses "container=host,container2=host2" into mappings sorted by
// descending container length so the most specific prefix matches first.
func parsePathMap(raw string) ([]pathMap, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var maps []pathMap
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		container, host, ok := strings.Cut(pair, "=")
		container, host = strings.TrimRight(strings.TrimSpace(container), "/"), strings.TrimRight(strings.TrimSpace(host), "/")
		if !ok || container == "" || host == "" {
			return nil, fmt.Errorf("invalid PATH_MAP entry %q, expected container=host", pair)
		}
		maps = append(maps, pathMap{container: container, host: host})
	}
	sort.Slice(maps, func(i, j int) bool {
		return len(maps[i].container) > len(maps[j].container)
	})
	return maps, nil
}
