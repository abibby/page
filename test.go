package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/abibby/page/internal/bookmeta"
	"github.com/abibby/page/internal/hardcover"
	"github.com/abibby/page/internal/opf"
	"github.com/davecgh/go-spew/spew"
	bolt "go.etcd.io/bbolt"
)

var BucketTestResults = &Table[*TestResult]{
	bucket: []byte("test-results"),
}

type TestResult struct {
	Path               string `json:"path"`
	HardcoverIDMatches bool   `json:"id_matches"`
	MissingBook        bool   `json:"missing_book"`
	Error              string `json:"error"`
}

func runTests(ctx context.Context, a *app) {

	db, err := bolt.Open("./test.bolt", 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		return BucketTestResults.InitTable(tx)
	})
	if err != nil {
		panic(err)
	}

	valid := 0
	erred := 0
	missing := 0
	invalid := 0

	err = db.View(func(tx *bolt.Tx) error {
		return BucketTestResults.Each(tx, func(k []byte, t *TestResult) error {
			// fmt.Printf("%s %#v\n", t.Path, t.Error)
			if t.Error == "" {
				if !t.HardcoverIDMatches {
					invalid++
					// fmt.Printf("%s\n", t.Path)
				} else {
					valid++
				}
			} else if t.MissingBook {
				missing++
			} else {
				erred++
			}
			return nil
		})
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Valid: %d | Invalid: %d| Missed: %d | Errors: %d\n", valid, invalid, missing, erred)

	authorDirs, err := os.ReadDir(a.cfg.CalibreLibrary)
	if err != nil {
		panic(err)
	}

	for _, authorDir := range authorDirs {
		if !authorDir.IsDir() || strings.HasPrefix(authorDir.Name(), ".") {
			continue
		}
		authorPath := path.Join(a.cfg.CalibreLibrary, authorDir.Name())
		bookDirs, err := os.ReadDir(authorPath)
		if err != nil {
			panic(err)
		}

		for _, bookDir := range bookDirs {
			if !bookDir.IsDir() || strings.HasPrefix(bookDir.Name(), ".") {
				continue
			}
			bookTest(db, a, path.Join(authorPath, bookDir.Name()))
		}

	}

}

func bookTest(db *bolt.DB, a *app, bookPath string) error {
	calibreMeta, err := readOPF(path.Join(bookPath, "metadata.opf"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	bookFiles, err := os.ReadDir(bookPath)
	if err != nil {
		return err
	}

	for _, f := range bookFiles {
		if !bookmeta.Supported(f.Name()) {
			continue
		}
		formatPath := path.Join(bookPath, f.Name())

		err = db.Update(func(tx *bolt.Tx) error {
			key := []byte(formatPath)
			result, err := BucketTestResults.Get(tx, key)
			if err != nil {
				return err
			}
			if result != nil {
				if !result.HardcoverIDMatches && result.Error == "" && strings.Contains(result.Path, "Shadow and Bone") {
					log.Printf("mismatched %s", formatPath)
					hcBook, meta, err := a.findBook(context.Background(), formatPath)
					spew.Dump(hcBook, meta, err)
					os.Exit(1)
				}
				return nil
			}
			hcBook, _, err := a.findBook(context.Background(), formatPath)
			if hcBook == nil {
				hcBook = &hardcover.Book{}
			}
			var strErr string
			if err != nil {
				strErr = err.Error()
			}
			return BucketTestResults.Put(tx, key, &TestResult{
				Path:               formatPath,
				HardcoverIDMatches: calibreMeta.Identifier("HARDCOVER-ID") == fmt.Sprint(hcBook.HardcoverID),
				MissingBook:        errors.Is(err, ErrNoBook),
				Error:              strErr,
			})
		})
		if err != nil {
			log.Printf("db update failed: %v", err)
		}
	}
	return nil
}

func readOPF(p string) (*opf.Metadata, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	meta := &opf.Package{}

	err = xml.Unmarshal(b, meta)
	if err != nil {
		return nil, err
	}
	return &meta.Metadata, nil
}

type Table[T any] struct {
	bucket []byte
}

func (t *Table[T]) InitTable(tx *bolt.Tx) error {
	_, err := tx.CreateBucketIfNotExists(t.bucket)
	return err
}

func (t *Table[T]) Get(tx *bolt.Tx, k []byte) (T, error) {
	b := tx.Bucket(t.bucket).Get(k)
	if b == nil {
		var zero T
		return zero, nil
	}
	return t.unmarshal(b)
}

func (t *Table[T]) Put(tx *bolt.Tx, k []byte, v T) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return tx.Bucket(t.bucket).Put(k, b)
}

func (t *Table[T]) Each(tx *bolt.Tx, fn func(k []byte, t T) error) error {
	return tx.Bucket(t.bucket).ForEach(func(k, v []byte) error {
		row, err := t.unmarshal(v)
		if err != nil {
			return err
		}
		return fn(k, row)
	})
}

func (t *Table[T]) unmarshal(b []byte) (T, error) {
	typ := reflect.TypeFor[T]()
	var val reflect.Value
	if typ.Kind() == reflect.Pointer {
		val = reflect.New(typ.Elem())
	} else {
		val = reflect.New(typ)
	}
	v := val.Interface()
	err := json.Unmarshal(b, v)
	if err != nil {
		var zero T
		return zero, err
	}

	if typ.Kind() != reflect.Pointer {
		val = val.Elem()
	}

	return val.Interface().(T), nil
}
