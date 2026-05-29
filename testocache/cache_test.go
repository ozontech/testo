package testocache

import (
	"slices"
	"testing"
)

func Test(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		Key   string
		Value string
	}{
		{Key: "my-key", Value: "lorem ipsum\ndolor sit \t\tamet"},
		{Key: "key~with~tilde", Value: "other value"},
	} {
		t.Run("with key: "+tt.Key, func(t *testing.T) {
			err := Set(tt.Key, []byte(tt.Value))
			if err != nil {
				t.Fatalf("failed to set cache: %v", err)
			}

			got, err := Get(tt.Key)
			if err != nil {
				t.Errorf("failed to get cache: %v", err)
			}

			if string(got) != tt.Value {
				t.Errorf("get cache: want %q, got %q", tt.Value, got)
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

	for _, k := range keys {
		err = Remove(k)
		if err != nil {
			t.Errorf("remove key %q: %v", k, err)
		}
	}
}
