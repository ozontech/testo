package gopls

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Position struct {
	File         string
	Line, Column int
}

func (p Position) String() string {
	return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column)
}

type ReferencesOpts struct {
	Position Position
	Tags     string
}

func References(ctx context.Context, opts ReferencesOpts) ([]Position, error) {
	cmd := exec.CommandContext(ctx, "gopls", "references", opts.Position.String())

	cmd.Env = os.Environ()

	if opts.Tags != "" {
		cmd.Env = append(cmd.Env, "GOFLAGS=-tags="+opts.Tags)
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gopls: %w", err)
	}

	var positions []Position

	for line := range bytes.Lines(out) {
		pos, ok := parsePosition(string(line))
		if !ok {
			continue
		}

		positions = append(positions, pos)
	}

	return positions, nil
}

func parsePosition(s string) (Position, bool) {
	const colon = ":"

	i := strings.LastIndex(s, colon)
	if i == -1 {
		return Position{}, false
	}

	j := strings.LastIndex(s[:i], colon)
	if j == -1 {
		return Position{}, false
	}

	columnStart, _, _ := strings.Cut(s[i+1:], "-")

	column, err := strconv.ParseInt(columnStart, 10, 32)
	if err != nil {
		return Position{}, false
	}

	line, err := strconv.ParseInt(s[j+1:i], 10, 32)
	if err != nil {
		return Position{}, false
	}

	return Position{
		File:   s[:j],
		Line:   int(line),
		Column: int(column),
	}, true
}
