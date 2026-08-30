// Package testocache provides persistent caching primitives for plugins.
//
// The cache directory defaults to ".testo_cache" in the test working directory.
// It can be changed with -cache.dir or TESTO_CACHE_DIR.
// Caching can be disabled with -cache.disable or TESTO_CACHE_DISABLE.
//
// Package-level functions use the default keyspace. Use [Namespace] to create
// an isolated keyspace.
//
// The on-disk format is internal. Writes use a temporary file and [os.Rename].
// Malformed entries are treated as missing. Operations are synchronized within
// one process; cross-process locking is not provided.
package testocache

import (
	"bufio"
	"bytes"
	"cmp"
	"errors"
	"flag"
	"hash/fnv"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"sync"

	"github.com/ozontech/testo/internal/env"
	"github.com/ozontech/testo/internal/parse"
)

var (
	flagDir = flag.String(
		"cache.dir",
		cmp.Or(env.TestoCacheDir.Get(), ".testo_cache"),
		"directory where the testo cache is stored",
	)
	flagDisable = flag.Bool(
		"cache.disable",
		parse.Bool(env.TestoCacheDisable.Get()),
		"disable caching in testo",
	)
)

var (
	// ErrDisabled indicates that caching is disabled.
	ErrDisabled = errors.New("testocache: cache is disabled")

	// ErrInvalidKey indicates that passed key is invalid.
	// Currently, key is invalid if it contains a NUL-byte.
	ErrInvalidKey = errors.New("testocache: invalid key")

	// ErrNotFound indicates that value was not found for the passed key.
	ErrNotFound = errors.New("testocache: not found")
)

const (
	permFile         os.FileMode = 0o600
	permDir          os.FileMode = 0o750
	namespaceDirName             = ".namespaces"
)

// Disabled returns true if caching is disabled.
// It's up to the package user to handle disabled state,
// e.g. do not save objects in cache when this function returns true.
func Disabled() bool {
	return *flagDisable
}

var kvMu sync.RWMutex

// Cache is a scoped key-value cache.
//
// The zero value uses the package-level cache keyspace.
// Use [Namespace] to create one when several plugins or helpers need to share
// the same cache directory without sharing the same keyspace.
type Cache struct {
	namespace string
}

// Namespace returns a scoped cache for name.
//
// Namespaces are isolated from each other and from the package-level cache
// functions. An empty name selects the package-level cache keyspace.
// Namespace names must follow the same validity rules as cache keys.
func Namespace(name string) Cache {
	return Cache{namespace: name}
}

// Keys returns keys matching pattern using [path.Match] syntax.
// Non-cache files in the cache directory are ignored.
// If cache is disabled (see [Disabled]), this function returns [ErrDisabled].
func Keys(pattern string) (keys []string, err error) {
	return Cache{}.Keys(pattern)
}

// Keys is like [Keys] but uses c's keyspace.
func (c Cache) Keys(pattern string) (keys []string, err error) {
	if err := validatePattern(pattern); err != nil {
		return nil, err
	}

	dir, err := c.dir()
	if err != nil {
		return nil, err
	}

	return keysIn(dir, pattern)
}

func keysIn(dir, pattern string) (keys []string, err error) {
	kvMu.RLock()
	defer kvMu.RUnlock()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	keys = make([]string, 0, len(entries))

	for _, e := range entries {
		if !e.Type().IsRegular() || !isCacheFilename(e.Name()) {
			continue
		}

		key, ok, err := extractKey(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}

		if !ok {
			continue
		}

		if e.Name() != hash(key) {
			continue
		}

		if ok, _ := path.Match(pattern, key); ok {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

func extractKey(p string) (key string, ok bool, err error) {
	f, err := os.Open(p)
	if err != nil {
		return "", false, err
	}
	defer f.Close()

	key, err = bufio.NewReader(f).ReadString(0)
	if errors.Is(err, io.EOF) {
		return "", false, nil
	}

	if err != nil {
		return "", false, err
	}

	return key[:len(key)-1], true, nil
}

// Get cached object by the given key.
// Key must not contain a NUL-byte.
// Malformed cache entries or entries with mismatched stored keys are treated
// as missing.
//
// If cache is disabled (see [Disabled]), this function returns [ErrDisabled].
func Get(key string) ([]byte, error) {
	return Cache{}.Get(key)
}

// Get is like [Get] but uses c's keyspace.
func (c Cache) Get(key string) ([]byte, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	dir, err := c.dir()
	if err != nil {
		return nil, err
	}

	return getFrom(dir, key)
}

func getFrom(dir, key string) ([]byte, error) {
	kvMu.RLock()
	defer kvMu.RUnlock()

	p := filepath.Join(dir, hash(key))

	return readEntry(p, key)
}

// Set saves value to cache with the given key.
// Key must not contain a NUL-byte.
// Existing values for the same key are replaced.
//
// If cache is disabled (see [Disabled]), this function returns [ErrDisabled].
func Set(key string, value []byte) error {
	return Cache{}.Set(key, value)
}

// Set is like [Set] but uses c's keyspace.
func (c Cache) Set(key string, value []byte) error {
	if err := validateKey(key); err != nil {
		return err
	}

	dir, err := c.dir()
	if err != nil {
		return err
	}

	return setIn(dir, key, value)
}

func setIn(dir, key string, value []byte) error {
	kvMu.Lock()
	defer kvMu.Unlock()

	p := filepath.Join(dir, hash(key))

	buf := bytes.NewBufferString(key)

	buf.Grow(1 + len(value))

	buf.WriteByte(0)
	buf.Write(value)

	return writeFileAtomic(p, buf.Bytes())
}

// Remove object from cache by the given key.
// Key must not contain a NUL-byte.
// If the key is not present, this function returns [ErrNotFound].
//
// If cache is disabled (see [Disabled]), this function returns [ErrDisabled].
func Remove(key string) error {
	return Cache{}.Remove(key)
}

// Remove is like [Remove] but uses c's keyspace.
func (c Cache) Remove(key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	dir, err := c.dir()
	if err != nil {
		return err
	}

	return removeFrom(dir, key)
}

func (c Cache) scoped() bool {
	return c.namespace != ""
}

func removeFrom(dir, key string) error {
	kvMu.Lock()
	defer kvMu.Unlock()

	p := filepath.Join(dir, hash(key))

	if _, err := readEntry(p, key); err != nil {
		return err
	}

	if err := os.Remove(p); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotFound
		}

		return err
	}

	return nil
}

func (c Cache) dir() (string, error) {
	if c.scoped() {
		if err := validateKey(c.namespace); err != nil {
			return "", err
		}
	}

	dir, err := cacheDir()
	if err != nil {
		return "", err
	}

	if !c.scoped() {
		return dir, nil
	}

	dir = filepath.Join(dir, namespaceDirName, hash(c.namespace))

	if err := os.MkdirAll(dir, permDir); err != nil {
		return "", err
	}

	return dir, nil
}

func readEntry(p, key string) ([]byte, error) {
	linfo, err := os.Lstat(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	if !linfo.Mode().IsRegular() {
		return nil, ErrNotFound
	}

	f, err := os.Open(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}

		return nil, err
	}
	defer f.Close()

	// Stat the opened descriptor and require identity with the Lstat result,
	// so a symlink swapped in between the two calls cannot be followed.
	finfo, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if !finfo.Mode().IsRegular() || !os.SameFile(linfo, finfo) {
		return nil, ErrNotFound
	}

	value, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	storedKey, value, ok := bytes.Cut(value, []byte{0})
	if !ok || string(storedKey) != key {
		return nil, ErrNotFound
	}

	return value, nil
}

func cacheDir() (string, error) {
	if Disabled() {
		return "", ErrDisabled
	}

	dir := *flagDir

	if err := os.MkdirAll(dir, permDir); err != nil {
		return "", err
	}

	if err := ensureGitignore(dir); err != nil {
		return "", err
	}

	return dir, nil
}

func ensureGitignore(dir string) error {
	p := filepath.Join(dir, ".gitignore")

	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, permFile)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}

		return err
	}

	_, err = f.WriteString("/*\n")
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}

	if err != nil {
		if removeErr := os.Remove(p); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return errors.Join(err, removeErr)
		}
	}

	return err
}

func writeFileAtomic(p string, data []byte) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(p), ".tmp-*")
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			return
		}

		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	_, err = tmp.Write(data)
	if err != nil {
		return err
	}

	err = tmp.Sync()
	if err != nil {
		return err
	}

	err = tmp.Close()
	if err != nil {
		return err
	}

	return os.Rename(tmp.Name(), p)
}

func validateKey(key string) error {
	if slices.Contains([]byte(key), 0) {
		return ErrInvalidKey
	}

	return nil
}

func validatePattern(pattern string) error {
	if err := validateKey(pattern); err != nil {
		return err
	}

	if _, err := path.Match(pattern, ""); err != nil {
		return err
	}

	return nil
}

func isCacheFilename(name string) bool {
	h, err := strconv.ParseUint(name, 36, 64)
	if err != nil {
		return false
	}

	return strconv.FormatUint(h, 36) == name
}

func hash(key string) string {
	h := fnv.New64a()

	if _, err := h.Write([]byte(key)); err != nil {
		panic(err)
	}

	return strconv.FormatUint(h.Sum64(), 36)
}
