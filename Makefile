MAKEFLAGS += --always-make
TEST_FLAGS ?= 

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
	go test -coverprofile=coverage.out -coverpkg=./... ./...
	go tool cover -func coverage.out

# get test coverage-ci
.PHONY: coverage-ci
coverage-ci:
	go test $(TEST_FLAGS) -coverprofile=coverage.unit.out -coverpkg=./... ./...
	go test $(TEST_FLAGS) -tags e2e -coverprofile=coverage.e2e.out -coverpkg=./... ./... || true
	go run github.com/dlespiau/covertool@latest merge -o coverage.out coverage.unit.out coverage.e2e.out
	go tool cover -func=coverage.out
	rm -f coverage.unit.out coverage.e2e.out

# visualize test coverage
coverage-html: coverage
	go tool cover -html coverage.out

install:
	go install ./cmd/testo

update-examples-output:
	./update-examples-output.sh
