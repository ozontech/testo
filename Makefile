MAKEFLAGS += --always-make

all: generate fmt lint test test-examples tidy

# run tests
test:
	go test -race -shuffle=on ./...

# check that examples output is correct
test-examples:
	go test -tags e2e -count=1 ./examples_test.go

# format source code
fmt:
	golangci-lint fmt

# lint source code
lint:
	golangci-lint run --tests=false

# run code generation
generate:
	go generate ./...

# tidy up go.mod
tidy:
	go mod tidy -v

doc:
	go run golang.org/x/pkgsite/cmd/pkgsite@latest -open

# get test coverage
coverage:
	go test -coverprofile=profile.cov ./...
	go tool cover -func profile.cov

# visualize test coverage
coverage-html: coverage
	go tool cover -html profile.cov

install:
	go install ./cmd/testo

update-examples-output:
	./update-examples-output.sh
