package packageslite

import (
	"go/token"
	"testing"
)

func BenchmarkLoad(b *testing.B) {
	b.Chdir("/Users/vzbarashchenko/Code/ozon/mass-crm-api")

	conf := Config{
		FSet: token.NewFileSet(),
		Tags: "e2e,smoke,functional,integration",
	}

	for b.Loop() {
		_, err := Load(conf, "./...")
		if err != nil {
			b.Fatal(err)
		}
	}
}
