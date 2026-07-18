// Package hardcover is a small client for the Hardcover GraphQL API
// (https://api.hardcover.app/v1/graphql), used to enrich book metadata.
package hardcover

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var ErrNotFound = errors.New("not found")

// Client queries the Hardcover GraphQL API.
type Client struct {
	endpoint string
	token    string
	http     *http.Client
	throttle <-chan time.Time
}

// Book is the normalised metadata returned to callers.
type Book struct {
	HardcoverID int
	Title       string
	Authors     []string
	Series      string
	SeriesIndex float64
	ReleaseYear int
	ISBN13      string
	ISBN10      string
	CoverURL    string
}

// New creates a client. token may include the leading "Bearer " prefix; it is
// added automatically if absent.
func New(endpoint, token string) *Client {
	token = strings.TrimSpace(token)
	if token != "" && !strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = "Bearer " + token
	}
	return &Client{
		endpoint: endpoint,
		token:    token,
		http:     &http.Client{Timeout: 30 * time.Second},
		throttle: time.Tick(time.Second),
	}
}

// graphql request/response envelopes.
type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type gqlError struct {
	Message string `json:"message"`
}

// edition/book shapes matching the queried fields.
type RawBook struct {
	ID            any    `json:"id"`
	Title         string `json:"title"`
	ReleaseYear   int    `json:"release_year"`
	Contributions []struct {
		Author struct {
			Name string `json:"name"`
		} `json:"author"`
	} `json:"contributions"`
	BookSeries *struct {
		Position float64 `json:"position"`
		Series   struct {
			Name string `json:"name"`
		} `json:"series"`
	} `json:"featured_series"`
	Image struct {
		URL string `json:"url"`
	} `json:"image"`
	ISBNs     []string `json:"isbns"`
	UserCount int      `json:"users_count"`
}

type apiEdition struct {
	Title  string  `json:"title"`
	ISBN13 string  `json:"isbn_13"`
	ISBN10 string  `json:"isbn_10"`
	Book   RawBook `json:"book"`
}

const bookParams = `
	id
	title
	release_year
	contributions { author { name } }
	featured_series: featured_book_series {
		position
		series { name }
	}
	image { url }
`

// LookupByISBN finds the edition matching an ISBN-13 or ISBN-10 and returns its
// parent book's metadata. Returns (nil, nil) when no edition matches.
func (c *Client) LookupByISBN(ctx context.Context, isbn string) (*Book, error) {
	query := `query ($isbn: String!) {
  editions(where: {_or: [{isbn_13: {_eq: $isbn}}, {isbn_10: {_eq: $isbn}}]}, limit: 1) {
    title
    isbn_13
    isbn_10
    book {` + bookParams + `}
  }
}`

	var resp struct {
		Editions []apiEdition `json:"editions"`
	}
	if err := c.do(ctx, query, map[string]any{"isbn": isbn}, &resp); err != nil {
		return nil, err
	}
	if len(resp.Editions) == 0 {
		return nil, ErrNotFound
	}
	ed := resp.Editions[0]
	book := bookFromAPI(ed.Book)
	book.ISBN13 = ed.ISBN13
	book.ISBN10 = ed.ISBN10
	if book.Title == "" {
		book.Title = ed.Title
	}
	return &book, nil
}

// SearchByTitleAuthor is the fallback when no ISBN is available. It matches
// books by title (case-insensitive), preferring popular results, then narrows
// by author when one is supplied. Returns (nil, nil) when nothing matches.
func (c *Client) SearchByTitleAuthor(ctx context.Context, title, author string) (*Book, error) {
	if strings.TrimSpace(title) == "" {
		return nil, nil
	}
	query := `query ($query: String!) {
  search(query: $query) {
    results
  }
}`
	type hit struct {
		Document RawBook `json:"document"`
	}
	var resp struct {
		Search struct {
			Results struct {
				Hits []hit `json:"hits"`
			} `json:"results"`
		} `json:"search"`
	}

	if err := c.do(ctx, query, map[string]any{"query": title}, &resp); err != nil {
		return nil, err
	}

	for _, h := range resp.Search.Results.Hits {
		if h.Document.UserCount < 100 {
			continue
		}
		b := h.Document
		if author = strings.ToLower(strings.TrimSpace(author)); author != "" {
			for _, contrib := range b.Contributions {
				if len(b.ISBNs) == 0 {
					continue
				}
				if strings.Contains(strings.ToLower(contrib.Author.Name), author) ||
					strings.Contains(author, strings.ToLower(contrib.Author.Name)) {
					book := bookFromAPI(b)
					return &book, nil
				}
			}
		} else {
			book := bookFromAPI(b)
			return &book, nil
		}
	}
	// book := bookFromAPI(resp.Search.Results.Hits[0].Document)
	return nil, ErrNotFound
}

func (c *Client) Query(ctx context.Context, q string) ([]RawBook, error) {
	if strings.TrimSpace(q) == "" {
		return []RawBook{}, nil
	}
	query := `query ($query: String!) {
  search(query: $query) {
    results
  }
}`
	type hit struct {
		Document RawBook `json:"document"`
	}
	var resp struct {
		Search struct {
			Results struct {
				Hits []hit `json:"hits"`
			} `json:"results"`
		} `json:"search"`
	}

	if err := c.do(ctx, query, map[string]any{"query": q}, &resp); err != nil {
		return nil, err
	}

	docs := make([]RawBook, len(resp.Search.Results.Hits))

	for i, h := range resp.Search.Results.Hits {
		docs[i] = h.Document
	}

	return docs, nil
}

func (c *Client) GetBook(ctx context.Context, id int) (*Book, error) {
	query := `query ($id: Int!) {
  		books(where: {id: {_eq: $id}}) {` + bookParams + `}
	}`

	var resp struct {
		Books []RawBook `json:"books"`
	}
	if err := c.do(ctx, query, map[string]any{"id": id}, &resp); err != nil {
		return nil, err
	}
	if len(resp.Books) == 0 {
		return nil, ErrNotFound
	}

	book := bookFromAPI(resp.Books[0])
	return &book, nil
}

func bookFromAPI(b RawBook) Book {
	var hcID int

	switch id := b.ID.(type) {
	case string:
		i, err := strconv.Atoi(id)
		if err == nil {
			hcID = i
		}
	case float64:
		hcID = int(id)
	}
	out := Book{
		HardcoverID: hcID,
		Title:       b.Title,
		ReleaseYear: b.ReleaseYear,
		CoverURL:    b.Image.URL,
	}
	for _, contrib := range b.Contributions {
		if name := strings.TrimSpace(contrib.Author.Name); name != "" {
			out.Authors = append(out.Authors, name)
		}
	}
	if b.BookSeries != nil {
		out.Series = b.BookSeries.Series.Name
		out.SeriesIndex = b.BookSeries.Position
	}
	return out
}

func (c *Client) do(ctx context.Context, query string, vars map[string]any, out any) error {
	<-c.throttle
	body, err := json.Marshal(gqlRequest{Query: query, Variables: vars})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("hardcover request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hardcover request: status %d", resp.StatusCode)
	}

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []gqlError      `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("hardcover decode: %w", err)
	}
	if len(envelope.Errors) > 0 {
		msgs := make([]string, len(envelope.Errors))
		for i, e := range envelope.Errors {
			msgs[i] = e.Message
		}
		return fmt.Errorf("hardcover graphql error: %s", strings.Join(msgs, "; "))
	}
	return json.Unmarshal(envelope.Data, out)
}

// SeriesIndexString renders the series index without a trailing ".0".
func (b Book) SeriesIndexString() string {
	if b.SeriesIndex == 0 {
		return ""
	}
	return strconv.FormatFloat(b.SeriesIndex, 'f', -1, 64)
}
