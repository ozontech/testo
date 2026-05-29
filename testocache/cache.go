// Package testocache provides caching primitives to be used by
// external plugins.
//
// By default, it stores cache in a directory "$TWD/.testo_cache",
// where "$TWD" refers to the "test working directory" (not necessary a project root).
// Usually, this is a directory where "_test.go" file, which calls this package, is located.
//
// Can be overridden passing "-cache.dir ~/My/Dir" flag to the "go test"
// command OR (with lesser priority) with environment variable "TESTO_CACHE_DIR".
//
// Caching can also be disabled with flag "-cache.disable" or environtment
// variable "TESTO_CACHE_DISABLE" (e.g. "=true").
package testocache

import (
	"bytes"
	"cmp"
	"errors"
	"flag"
	"hash/fnv"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
)

var (
	flagDir = flag.String(
		"cache.dir",
		cmp.Or(os.Getenv("TESTO_CACHE_DIR"), ".testo_cache"),
		"directory where the testo cache is stored",
	)
	flagDisable = flag.Bool(
		"cache.disable",
		parseBool(os.Getenv("TESTO_CACHE_DISABLE")),
		"disable caching in testo",
	)
)

// ErrDisabled indicates that caching is disabled.
var ErrDisabled = errors.New("cache is disabled")

const (
	permFile os.FileMode = 0o600
	permDir  os.FileMode = 0o750
)

// Disabled returns true if caching is disabled.
// It's up to the package user to handle disabled state,
// e.g. do not save objects in cache when this function returns true.
func Disabled() bool {
	return *flagDisable
}

var kvMu sync.RWMutex

// Keys returns all glob-matched keys by the given pattern.
//
// The pattern syntax is:
//
//	pattern:
//		{ term }
//	term:
//		'*'         matches any sequence of non-/ characters
//		'?'         matches any single non-/ character
//		'[' [ '^' ] { character-range } ']'
//		            character class (must be non-empty)
//		c           matches character c (c != '*', '?', '\\', '[')
//		'\\' c      matches character c
//
//	character-range:
//		c           matches character c (c != '\\', '-', ']')
//		'\\' c      matches character c
//		lo '-' hi   matches character c for lo <= c <= hi
//
// If cache is disabled (see [Disabled]), this function returns [ErrDisabled].
func Keys(pattern string) (keys []string, err error) {
	if _, err := path.Match(pattern, ""); err != nil {
		return nil, err
	}

	dir, err := cacheDir()
	if err != nil {
		return nil, err
	}

	kvMu.RLock()
	defer kvMu.RUnlock()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	keys = make([]string, 0, len(keys))

	for _, e := range entries {
		key, err := extractKey(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}

		if ok, _ := path.Match(pattern, key); ok {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

func extractKey(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, 16)

	for {
		n, err := io.ReadAtLeast(f, buf, 1)
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}

		before, _, ok := bytes.Cut(buf[:n], []byte{0})
		if ok {
			return string(before), nil
		}

		if errors.Is(err, io.EOF) {
			return "", nil
		}
	}
}

// Get cached object by the given key.
//
// If cache is disabled (see [Disabled]), this function returns [ErrDisabled].
func Get(key string) ([]byte, error) {
	dir, err := cacheDir()
	if err != nil {
		return nil, err
	}

	kvMu.RLock()
	defer kvMu.RUnlock()

	h, err := hash(key)
	if err != nil {
		return nil, err
	}

	p := filepath.Join(dir, h)

	value, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	_, after, ok := bytes.Cut(value, []byte{0})
	if !ok {
		return value, nil
	}

	return after, nil
}

// Set saves value to cache with the given key.
//
// If cache is disabled (see [Disabled]), this function returns [ErrDisabled].
func Set(key string, value []byte) error {
	dir, err := cacheDir()
	if err != nil {
		return err
	}

	kvMu.Lock()
	defer kvMu.Unlock()

	h, err := hash(key)
	if err != nil {
		return err
	}

	p := filepath.Join(dir, h)

	buf := bytes.NewBufferString(key)

	buf.Grow(1 + len(value))

	buf.WriteByte(0)
	buf.Write(value)

	return os.WriteFile(p, buf.Bytes(), permFile)
}

// Remove object from cache by the given key.
//
// If cache is disabled (see [Disabled]), this function returns [ErrDisabled].
func Remove(key string) error {
	dir, err := cacheDir()
	if err != nil {
		return err
	}

	kvMu.Lock()
	defer kvMu.Unlock()

	h, err := hash(key)
	if err != nil {
		return err
	}

	p := filepath.Join(dir, h)

	return os.Remove(p)
}

func cacheDir() (string, error) {
	if Disabled() {
		return "", ErrDisabled
	}

	dir := *flagDir

	if err := os.MkdirAll(dir, permDir); err != nil {
		return "", err
	}

	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("/*"), permFile); err != nil {
		return "", err
	}

	return dir, nil
}

func parseBool(s string) bool {
	b, _ := strconv.ParseBool(s)

	return b
}

func hash(key string) (string, error) {
	h := fnv.New64a()

	_, err := h.Write([]byte(key))
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(h.Sum64(), 36), nil
}
