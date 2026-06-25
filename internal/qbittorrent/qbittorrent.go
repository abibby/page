// Package qbittorrent is a minimal client for the qBittorrent WebUI API (v2).
package qbittorrent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"strings"
	"time"
)

// Client talks to a qBittorrent WebUI instance.
type Client struct {
	base string
	http *http.Client
}

// Torrent is the subset of /torrents/info fields this tool needs.
type Torrent struct {
	Hash        string  `json:"hash"`
	Name        string  `json:"name"`
	State       string  `json:"state"`
	SavePath    string  `json:"save_path"`
	ContentPath string  `json:"content_path"`
	Tags        string  `json:"tags"`
	Progress    float64 `json:"progress"`
}

// File is the subset of /torrents/files fields this tool needs.
type File struct {
	Name     string  `json:"name"`
	Index    int     `json:"index"`
	Size     int64   `json:"size"`
	Progress float64 `json:"progress"`
}

// New creates a client and authenticates against the WebUI, returning a client
// whose cookie jar carries the session ID (SID).
func New(base, username, password string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	c := &Client{
		base: strings.TrimRight(base, "/"),
		http: &http.Client{Jar: jar, Timeout: 30 * time.Second},
	}
	if err := c.login(username, password); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) login(username, password string) error {
	form := url.Values{"username": {username}, "password": {password}}
	req, err := http.NewRequest(http.MethodPost, c.base+"/api/v2/auth/login", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// qBittorrent rejects cross-site requests; set a matching Referer/Origin.
	req.Header.Set("Referer", c.base)
	req.Header.Set("Origin", c.base)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("qbittorrent login: %w", err)
	}
	defer resp.Body.Close()
	// body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("qbittorrent login failed: status %d", resp.StatusCode)
	}
	// // On bad credentials qBittorrent returns 200 with body "Fails.".
	// if strings.TrimSpace(string(body)) != "Ok." {
	// 	return fmt.Errorf("qbittorrent login rejected: %q", strings.TrimSpace(string(body)))
	// }
	return nil
}

// TorrentsByTag returns torrents carrying the given tag, optionally filtered by
// state (e.g. "completed"). Pass an empty filter for all states.
func (c *Client) TorrentsByTag(tag, filter string) ([]Torrent, error) {
	q := url.Values{"tag": {tag}}
	if filter != "" {
		q.Set("filter", filter)
	}
	var torrents []Torrent
	if err := c.getJSON("/api/v2/torrents/info?"+q.Encode(), &torrents); err != nil {
		return nil, err
	}
	return torrents, nil
}

func (c *Client) AddTag(torrent *Torrent, tags ...string) error {
	q := url.Values{
		"hashes": {torrent.Hash},
		"tags":   {strings.Join(tags, ",")},
	}

	if err := c.postJSON("/api/v2/torrents/addTags", q, nil); err != nil {
		return err
	}
	return nil
}

// Files returns the file list for a torrent.
func (c *Client) Files(hash string) ([]File, error) {
	q := url.Values{"hash": {hash}}
	var files []File
	if err := c.getJSON("/api/v2/torrents/files?"+q.Encode(), &files); err != nil {
		return nil, err
	}
	return files, nil
}

func (c *Client) getJSON(p string, out any) error {
	return c.request(http.MethodGet, p, nil, out)
}
func (c *Client) postJSON(p string, q url.Values, out any) error {
	return c.request(http.MethodPost, p, strings.NewReader(q.Encode()), out)
}

func (c *Client) request(method, p string, body io.Reader, out any) error {
	req, err := http.NewRequest(method, c.base+p, body)
	if err != nil {
		return err
	}
	req.Header.Set("Referer", c.base)
	if body != nil && body != http.NoBody {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s: status %d: %s", p, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// HasTag reports whether a torrent's comma-separated tag list contains tag.
func (t Torrent) HasTag(tag string) bool {
	for _, candidate := range strings.Split(t.Tags, ",") {
		if strings.TrimSpace(candidate) == tag {
			return true
		}
	}
	return false
}

// AbsPath returns the absolute container path of a file within the torrent,
// joining the torrent save path with the file's relative name.
func (t Torrent) AbsPath(f File) string {
	return path.Join(t.SavePath, f.Name)
}
