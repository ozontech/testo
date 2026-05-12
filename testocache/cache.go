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
	"cmp"
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode"
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
// E.g. "myplugin-prefix-*"
//
// If cache is disabled (see [Disabled]), this function returns [ErrDisabled].
func Keys(pattern string) (keys []string, err error) {
	dir, err := cacheDir()
	if err != nil {
		return nil, err
	}

	kvMu.RLock()
	defer kvMu.RUnlock()

	return fs.Glob(os.DirFS(dir), pattern)
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

	path := filepath.Join(dir, sanitizeFilename(key))

	return os.ReadFile(path)
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

	path := filepath.Join(dir, sanitizeFilename(key))

	return os.WriteFile(path, value, permFile)
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

	path := filepath.Join(dir, sanitizeFilename(key))

	return os.Remove(path)
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

func sanitizeFilename(name string) string {
	var sb strings.Builder

	sb.Grow(len(name))

	const (
		invalid     = `\/<>:\"|?*.`
		replacement = '-'
	)

	for _, r := range name {
		switch {
		case r == 0, unicode.IsControl(r), strings.ContainsRune(invalid, r):
			sb.WriteRune(replacement)

		default:
			sb.WriteRune(r)
		}
	}

	return sb.String()
}
