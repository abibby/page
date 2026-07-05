// Package config loads runtime configuration from environment variables,
// optionally seeded from a .env file, and handles Docker path remapping.
package config

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime settings.
type Config struct {
	// qBittorrent WebUI.
	QbitURL      string
	QbitUsername string
	QbitPassword string
	QbitTag      string
	QbitDoneTag  string
	QbitErrorTag string

	// Calibre.
	CalibreLibrary  string
	CalibreServer   string
	CalibreUsername string
	CalibrePassword string
	CalibredbBin    string
	AddDuplicates   bool

	// Hardcover GraphQL API.
	HardcoverURL   string
	HardcoverToken string

	// Daemon behaviour.
	PollInterval time.Duration
	StateFile    string
	DryRun       bool

	GeminiAPIKey string

	// pathMaps remap container paths (as reported by qBittorrent running in
	// Docker) to paths reachable from this host. Sorted longest-prefix first.
	pathMaps []pathMap
}

type pathMap struct {
	container string
	host      string
}

// Load reads .env (if present) then the process environment. Real environment
// variables always take precedence over values in the .env file.
func Load(envFile string) (*Config, error) {
	if err := loadDotEnv(envFile); err != nil {
		return nil, err
	}

	c := &Config{
		QbitURL:         getEnv("QBIT_URL", "http://localhost:8080"),
		QbitUsername:    os.Getenv("QBIT_USERNAME"),
		QbitPassword:    os.Getenv("QBIT_PASSWORD"),
		QbitTag:         getEnv("QBIT_TAG", "book"),
		QbitDoneTag:     getEnv("QBIT_DONE_TAG", "done"),
		QbitErrorTag:    getEnv("QBIT_ERROR_TAG", "error"),
		CalibreLibrary:  os.Getenv("CALIBRE_LIBRARY"),
		CalibreServer:   os.Getenv("CALIBRE_SERVER"),
		CalibreUsername: os.Getenv("CALIBRE_USERNAME"),
		CalibrePassword: os.Getenv("CALIBRE_PASSWORD"),
		CalibredbBin:    getEnv("CALIBREDB_BIN", "calibredb"),
		AddDuplicates:   getBool("ADD_DUPLICATES", false),
		HardcoverURL:    getEnv("HARDCOVER_URL", "https://api.hardcover.app/v1/graphql"),
		HardcoverToken:  os.Getenv("HARDCOVER_TOKEN"),
		GeminiAPIKey:    os.Getenv("GEMINI_API_KEY"),
		StateFile:       getEnv("STATE_FILE", "state.json"),
		DryRun:          getBool("DRY_RUN", false),
	}

	interval, err := time.ParseDuration(getEnv("POLL_INTERVAL", "5m"))
	if err != nil {
		return nil, fmt.Errorf("invalid POLL_INTERVAL: %w", err)
	}
	c.PollInterval = interval

	c.pathMaps, err = parsePathMap(os.Getenv("PATH_MAP"))
	if err != nil {
		return nil, err
	}

	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) validate() error {
	var missing []string
	if c.CalibreLibrary == "" {
		missing = append(missing, "CALIBRE_LIBRARY")
	}
	if c.HardcoverToken == "" {
		missing = append(missing, "HARDCOVER_TOKEN")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	return nil
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

// loadDotEnv parses a simple KEY=VALUE file and sets any variables that are not
// already present in the environment. Missing file is not an error.
func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		// Strip matching surrounding quotes.
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
			val = val[1 : len(val)-1]
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, val); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
