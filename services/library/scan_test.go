package library

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/abibby/page/config"
	"github.com/abibby/page/services/hardcover"
	"github.com/stretchr/testify/require"
	"gosalusa.com/database/builder"
)

func TestScanPreservesPathsAndImportsLocalMetadata(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	directory := filepath.Join(root, "Old Author (9)", "Old Book (42)")
	format := writeFile(t, filepath.Join(directory, "original.mobi"), "book")
	track := writeFile(t, filepath.Join(directory, "Disc 1", "01.mp3"), "audio")
	cover := writeFile(t, filepath.Join(directory, "cover.jpg"), "cover")
	opf := writeFile(t, filepath.Join(directory, "metadata.opf"), `<package xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf"><metadata>
<dc:title>Local title</dc:title><dc:creator>Old Author</dc:creator><dc:creator>Coauthor</dc:creator>
<dc:identifier opf:scheme="ISBN">9781234567890</dc:identifier>
<dc:identifier opf:scheme="hardcover-id">123</dc:identifier>
<dc:description>Local description</dc:description><dc:language>eng</dc:language><dc:subject>Fiction</dc:subject>
</metadata></package>`)
	original := readFile(t, opf)
	store, err := New(ctx, &config.Config{LibraryPath: root}, newDatabase(t))
	require.NoError(t, err)
	require.NoError(t, store.Scan(ctx))
	require.NoError(t, store.Scan(ctx))
	books, err := store.List(ctx, nil)
	require.NoError(t, err)
	require.Len(t, books, 1)
	b := books[0]
	require.Equal(t, directory, b.Directory)
	require.ElementsMatch(t, []string{format, track}, b.Formats)
	require.Equal(t, 9, b.Size)
	require.Equal(t, cover, b.Cover)
	require.Equal(t, "Local title", b.Title)
	require.Equal(t, "Old Author & Coauthor", b.Authors)
	require.Equal(t, "9781234567890", b.ISBN)
	require.Equal(t, "123", b.Identifiers["hardcover-id"])
	require.Equal(t, "Local description", b.Description)
	require.Equal(t, []string{"eng"}, []string(b.Languages))
	require.Equal(t, []string{"Fiction"}, []string(b.Tags))
	require.Equal(t, original, readFile(t, opf))
	require.Equal(t, "book", string(readFile(t, format)))
	require.Equal(t, "audio", string(readFile(t, track)))
	// A normal import reuses the scanned book and the original author path.
	id, err := store.AddEmptyBook(ctx, &hardcover.Book{HardcoverID: 123})
	require.NoError(t, err)
	require.Equal(t, b.ID, id)
	id, err = store.AddEmptyBook(ctx, &hardcover.Book{Title: "Next", Authors: []string{"Old Author"}})
	require.NoError(t, err)
	next, err := store.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, filepath.Dir(directory), filepath.Dir(next.Directory))
}

func TestScanReadsEPUBMetadata(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	name := writeFile(t, filepath.Join(root, "Folder Author", "Folder Title", "book.EPUB"), "")
	f, err := os.Create(name)
	require.NoError(t, err)
	z := zip.NewWriter(f)
	for path, contents := range map[string]string{
		"META-INF/container.xml": `<container><rootfiles><rootfile full-path="content.opf"/></rootfiles></container>`,
		"content.opf": `<package xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf"><metadata>
<dc:title>Embedded title</dc:title><dc:creator>Embedded author</dc:creator><dc:identifier opf:scheme="ISBN">9781234567890</dc:identifier>
</metadata></package>`,
	} {
		w, err := z.Create(path)
		require.NoError(t, err)
		_, err = w.Write([]byte(contents))
		require.NoError(t, err)
	}
	require.NoError(t, z.Close())
	require.NoError(t, f.Close())
	store, err := New(ctx, &config.Config{LibraryPath: root}, newDatabase(t))
	require.NoError(t, err)
	require.NoError(t, store.Scan(ctx))
	books, err := store.List(ctx, nil)
	require.NoError(t, err)
	require.Len(t, books, 1)
	require.Equal(t, "Embedded title", books[0].Title)
	require.Equal(t, "Embedded author", books[0].Authors)
	require.Equal(t, "9781234567890", books[0].ISBN)
}

func TestScanSkipsIndexedBooksAndUnrelatedFiles(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store, err := New(ctx, &config.Config{LibraryPath: root}, newDatabase(t))
	require.NoError(t, err)
	source := writeFile(t, filepath.Join(t.TempDir(), "existing.epub"), "already indexed")
	id, err := store.AddBook(ctx, source, false, &hardcover.Book{Title: "Existing", Authors: []string{"Author"}})
	require.NoError(t, err)
	before, err := store.Get(ctx, id)
	require.NoError(t, err)
	writeFile(t, filepath.Join(before.Directory, "metadata.opf"), "invalid metadata must not be read")
	writeFile(t, filepath.Join(root, "Author", "Unrelated", "notes.txt"), "notes")
	writeFile(t, filepath.Join(root, "Author (5)", "Audio (12)", "01.MP3"), "audio")
	require.NoError(t, store.Scan(ctx))
	after, err := store.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, before, after)
	books, err := store.List(ctx, &ListOptions{Search: "title:Audio"})
	require.NoError(t, err)
	require.Len(t, books, 1)
	require.Equal(t, "Audio", books[0].Title)
	require.Equal(t, "Author", books[0].Authors)
	books, err = store.List(ctx, nil)
	require.NoError(t, err)
	require.Len(t, books, 2)
	// The second folder for the same author must not replace the first mapping.
	id, err = store.AddEmptyBook(ctx, &hardcover.Book{Title: "Next", Authors: []string{"Author"}})
	require.NoError(t, err)
	next, err := store.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "Author"), filepath.Dir(next.Directory))
}

func TestScanContinuesAfterFailureAndRetries(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Author", "Bad", "book.pdf"), "book")
	opf := writeFile(t, filepath.Join(root, "Author", "Bad", "metadata.opf"), "<broken")
	writeFile(t, filepath.Join(root, "Author", "Good", "book.pdf"), "book")
	db := newDatabase(t)
	store, err := New(ctx, &config.Config{LibraryPath: root}, db)
	require.NoError(t, err)
	require.ErrorContains(t, store.Scan(ctx), "Bad")
	books, err := store.List(ctx, nil)
	require.NoError(t, err)
	require.Len(t, books, 1)
	require.Equal(t, "Good", books[0].Title)
	require.NoError(t, os.Remove(opf))
	require.NoError(t, store.Scan(ctx))
	books, err = store.List(ctx, nil)
	require.NoError(t, err)
	require.Len(t, books, 2)
}

func TestScanDryRunCancellationAndMissingRoot(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Author", "Book", "book.pdf"), "book")
	db := newDatabase(t)
	store, err := New(ctx, &config.Config{LibraryPath: root, DryRun: true}, db)
	require.NoError(t, err)
	require.NoError(t, store.Scan(ctx))
	for _, table := range []string{"library_books", "library_authors"} {
		count, err := builder.NewBuilder().WithContext(ctx).From(table).Count(db)
		require.NoError(t, err)
		require.Zero(t, count)
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	require.ErrorIs(t, store.Scan(ctx), context.Canceled)
	store.root = filepath.Join(root, "missing")
	require.ErrorIs(t, store.Scan(context.Background()), os.ErrNotExist)
}

func TestConcurrentScansDoNotDuplicate(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Author", "Book", "book.pdf"), "book")
	store, err := New(ctx, &config.Config{LibraryPath: root}, newDatabase(t))
	require.NoError(t, err)
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for range 8 {
		wg.Go(func() { results <- store.Scan(ctx) })
	}
	wg.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	books, err := store.List(ctx, nil)
	require.NoError(t, err)
	require.Len(t, books, 1)
}

func TestScanIgnoresSymlinks(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	outside := writeFile(t, filepath.Join(t.TempDir(), "book.pdf"), "outside")
	bookDir := filepath.Join(root, "Author", "Book")
	require.NoError(t, os.MkdirAll(bookDir, 0755))
	require.NoError(t, os.Symlink(outside, filepath.Join(bookDir, "link.pdf")))
	require.NoError(t, os.Symlink(filepath.Dir(outside), filepath.Join(root, "Linked Author")))
	require.NoError(t, os.Symlink(filepath.Dir(outside), filepath.Join(root, "Author", "Linked Book")))
	store, err := New(ctx, &config.Config{LibraryPath: root}, newDatabase(t))
	require.NoError(t, err)
	require.NoError(t, store.Scan(ctx))
	books, err := store.List(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, books)
}
