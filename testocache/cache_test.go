package testocache

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

var testFlagsMu sync.Mutex

func withCacheDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	withCacheConfig(t, dir, false)

	return dir
}

func withDisabledCache(t *testing.T) string {
	t.Helper()

	dir := filepath.Join(t.TempDir(), "cache")
	withCacheConfig(t, dir, true)

	return dir
}

func withCacheConfig(t *testing.T, dir string, disabled bool) {
	t.Helper()

	testFlagsMu.Lock()

	oldDir := *flagDir
	oldDisable := *flagDisable

	*flagDir = dir
	*flagDisable = disabled

	t.Cleanup(func() {
		*flagDir = oldDir
		*flagDisable = oldDisable

		testFlagsMu.Unlock()
	})
}

func TestInvalidKey(t *testing.T) {
	t.Parallel()

	const invalid = "foo\x00bar"

	t.Run("set", func(t *testing.T) {
		t.Parallel()

		err := Set(invalid, []byte("..."))
		if !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("err is not ErrInvalidKey: %v", err)
		}
	})

	t.Run("get", func(t *testing.T) {
		t.Parallel()

		_, err := Get(invalid)
		if !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("err is not ErrInvalidKey: %v", err)
		}
	})

	t.Run("remove", func(t *testing.T) {
		t.Parallel()

		err := Remove(invalid)
		if !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("err is not ErrInvalidKey: %v", err)
		}
	})

	t.Run("keys", func(t *testing.T) {
		t.Parallel()

		_, err := Keys(invalid)
		if !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("err is not ErrInvalidKey: %v", err)
		}
	})
}

func TestDisabledCache(t *testing.T) {
	dir := withDisabledCache(t)

	if !Disabled() {
		t.Fatal("Disabled returned false")
	}

	if err := Set("key", nil); !errors.Is(err, ErrDisabled) {
		t.Fatalf("package cache: want ErrDisabled, got %v", err)
	}
	if err := Namespace("plugin").Set("key", nil); !errors.Is(err, ErrDisabled) {
		t.Fatalf("namespace cache: want ErrDisabled, got %v", err)
	}

	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("disabled cache created its directory: %v", err)
	}
}

func TestFlow(t *testing.T) {
	withCacheDir(t)

	for _, tt := range []struct {
		name  string
		key   string
		value []byte
	}{
		{name: "plain", key: "my-key", value: []byte("lorem ipsum\ndolor sit \t\tamet")},
		{name: "tilde", key: "key~with~tilde", value: []byte("other value")},
		{name: "unicode and binary", key: "ключ/данные", value: []byte{0, 1, 2, 0, 255}},
		{name: "long key", key: strings.Repeat("long-", 100), value: bytes.Repeat([]byte("value"), 100)},
		{name: "empty", key: "", value: nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := Set(tt.key, tt.value)
			if err != nil {
				t.Fatalf("failed to set cache: %v", err)
			}

			got, err := Get(tt.key)
			if err != nil {
				t.Errorf("failed to get cache: %v", err)
			}

			if !bytes.Equal(got, tt.value) {
				t.Errorf("get cache: want %q, got %q", tt.value, got)
			}
		})
	}

	keys, err := Keys("*key*")
	if err != nil {
		t.Fatalf("failed to get keys: %v", err)
	}

	slices.Sort(keys)

	wantKeys := []string{"key~with~tilde", "my-key"}
	if !slices.Equal(keys, wantKeys) {
		t.Fatalf("keys: want %v, got %v", wantKeys, keys)
	}

	emptyKeys, err := Keys("")
	if err != nil {
		t.Fatalf("get empty key: %v", err)
	}
	// Set("", nil) above creates one cache entry whose key is the empty string.
	if !slices.Equal(emptyKeys, []string{""}) {
		t.Fatalf("empty keys: want [\"\"], got %q", emptyKeys)
	}

	for _, k := range keys {
		err = Remove(k)
		if err != nil {
			t.Errorf("remove key %q: %v", k, err)
		}
	}
	if err := Remove(""); err != nil {
		t.Errorf("remove empty key: %v", err)
	}

	_, err = Get("unknown-key")
	if !errors.Is(err, ErrNotFound) {
		t.Fatal("expected not found error")
	}
}

func TestKeysIgnoresNonCacheEntries(t *testing.T) {
	dir := withCacheDir(t)

	if err := Set("my-key", []byte("value")); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(dir, "foreign-file"),
		[]byte("not a cache entry"),
		permFile,
	); err != nil {
		t.Fatalf("write foreign file: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(dir, hash("requested-key")),
		[]byte("other-key\x00value"),
		permFile,
	); err != nil {
		t.Fatalf("write mismatched cache entry: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(dir, hash("malformed-key")),
		[]byte("missing separator"),
		permFile,
	); err != nil {
		t.Fatalf("write malformed cache entry: %v", err)
	}

	keys, err := Keys("*")
	if err != nil {
		t.Fatalf("keys: %v", err)
	}

	if !slices.Equal(keys, []string{"my-key"}) {
		t.Fatalf("keys: want [my-key], got %v", keys)
	}
}

func TestGetRejectsInvalidEntries(t *testing.T) {
	dir := withCacheDir(t)

	for _, tt := range []struct {
		key  string
		data string
	}{
		{key: "mismatched", data: "other-key\x00value"},
		{key: "malformed", data: "missing separator"},
	} {
		if err := os.WriteFile(
			filepath.Join(dir, hash(tt.key)),
			[]byte(tt.data),
			permFile,
		); err != nil {
			t.Fatalf("write %s entry: %v", tt.key, err)
		}
		if _, err := Get(tt.key); !errors.Is(err, ErrNotFound) {
			t.Fatalf("get %s entry: want ErrNotFound, got %v", tt.key, err)
		}
	}
}

func TestGetAndRemoveRejectSymlink(t *testing.T) {
	dir := withCacheDir(t)

	target := filepath.Join(dir, "symlink-target")
	if err := os.WriteFile(target, []byte("symlink-key\x00value"), permFile); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}

	link := filepath.Join(dir, hash("symlink-key"))
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("create symlink: %v", err)
	}

	if _, err := Get("symlink-key"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get symlink entry: want ErrNotFound, got %v", err)
	}

	if err := Remove("symlink-key"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("remove symlink entry: want ErrNotFound, got %v", err)
	}

	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("rejected symlink should not be removed: %v", err)
	}
}

func TestRemoveRejectsMissingAndMismatchedEntries(t *testing.T) {
	dir := withCacheDir(t)

	if err := Remove("missing-key"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("remove missing key: want ErrNotFound, got %v", err)
	}

	p := filepath.Join(dir, hash("requested-key"))
	if err := os.WriteFile(p, []byte("other-key\x00value"), permFile); err != nil {
		t.Fatalf("write cache file: %v", err)
	}

	err := Remove("requested-key")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("remove mismatched key: want ErrNotFound, got %v", err)
	}

	if _, err := os.Stat(p); err != nil {
		t.Fatalf("mismatched entry should not be removed: %v", err)
	}
}

func TestSetReplacesExistingValue(t *testing.T) {
	withCacheDir(t)

	if err := Set("key", []byte("old")); err != nil {
		t.Fatalf("set old value: %v", err)
	}

	if err := Set("key", []byte("new")); err != nil {
		t.Fatalf("set new value: %v", err)
	}

	got, err := Get("key")
	if err != nil {
		t.Fatalf("get value: %v", err)
	}

	if string(got) != "new" {
		t.Fatalf("value: want new, got %q", got)
	}
}

func TestNamespaceCache(t *testing.T) {
	withCacheDir(t)

	caches := []struct {
		name  string
		cache Cache
	}{
		{name: "default", cache: Cache{}},
		{name: "first", cache: Namespace("plugin/first")},
		{name: "second", cache: Namespace("plugin/second")},
	}

	for _, tt := range caches {
		if err := tt.cache.Set("key", []byte(tt.name)); err != nil {
			t.Fatalf("set %s value: %v", tt.name, err)
		}
	}
	for _, tt := range caches {
		value, err := tt.cache.Get("key")
		if err != nil {
			t.Fatalf("get %s value: %v", tt.name, err)
		}
		if string(value) != tt.name {
			t.Fatalf("%s value: want %q, got %q", tt.name, tt.name, value)
		}
	}

	value, err := Get("key")
	if err != nil || string(value) != "default" {
		t.Fatalf("package cache value: want %q, got %q, err %v", "default", value, err)
	}

	value, err = Namespace("").Get("key")
	if err != nil || string(value) != "default" {
		t.Fatalf("empty namespace value: want %q, got %q, err %v", "default", value, err)
	}
}

func TestNamespaceValidation(t *testing.T) {
	withCacheDir(t)

	if err := Namespace("plugin\x00bad").Set("key", nil); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("set with invalid namespace: want ErrInvalidKey, got %v", err)
	}

	cache := Namespace(strings.Repeat("long-namespace-", 100))

	if err := cache.Set("key", []byte("value")); err != nil {
		t.Fatalf("set with long namespace: %v", err)
	}

	value, err := cache.Get("key")
	if err != nil {
		t.Fatalf("get with long namespace: %v", err)
	}
	if string(value) != "value" {
		t.Fatalf("value: want value, got %q", value)
	}
}

func TestGitignoreIsPreserved(t *testing.T) {
	dir := withCacheDir(t)
	const custom = "custom contents\n"

	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(custom), permFile); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	if err := Set("key", nil); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if string(got) != custom {
		t.Fatalf(".gitignore was overwritten: want %q, got %q", custom, got)
	}
}

func TestGitignoreIsCreated(t *testing.T) {
	dir := withCacheDir(t)

	if err := Set("key", nil); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if string(got) != "/*\n" {
		t.Fatalf(".gitignore: want %q, got %q", "/*\n", got)
	}
}

func TestWriteFileAtomicCleansTemporaryFileAfterRenameError(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, permDir); err != nil {
		t.Fatalf("create target directory: %v", err)
	}

	if err := writeFileAtomic(target, []byte("value")); err == nil {
		t.Fatal("write over directory returned nil")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "target" {
		t.Fatalf("temporary file was not cleaned up: %v", entries)
	}
}

func TestHashCompatibility(t *testing.T) {
	t.Parallel()

	const want = "310vounr00im1"
	if got := hash("compatibility-key"); got != want {
		t.Fatalf("key hash changed: want %q, got %q", want, got)
	}
}
