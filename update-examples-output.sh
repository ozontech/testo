#!/bin/sh

set -eux

DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)

GOTEST="go test -tags example -v -count=1 -parallel=1"

find "$DIR/examples" -name "*_*" -type d -exec sh -c "$GOTEST \"\$0\" >\"\$0/output.golden\" 2>&1" {} \;
