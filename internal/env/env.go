// Package env holds all environment variables used by Testo.
package env

import "os"

const (
	TestoStrict       Env = "TESTO_STRICT"
	TestoCacheDir     Env = "TESTO_CACHE_DIR"
	TestoCacheDisable Env = "TESTO_CACHE_DISABLE"
)

type Env string

func (e Env) Get() string {
	return os.Getenv(string(e))
}
