package utils

import (
	"bytes"
	"fmt"

	"github.com/icyflame/kindle-my-clippings-parser/internal/parser"
	"v.io/x/lib/textutil"
)

// MakePlaintextEmailFromClipping ...
func MakePlaintextEmailFromClipping(c parser.Clipping) (string, error) {
	var buffer bytes.Buffer
	w := textutil.NewUTF8WrapWriter(&buffer, 80)
	if _, err := w.Write([]byte(c.Text)); err != nil {
		return "", fmt.Errorf("could not make plaintext email > %w", err)
	}

	if err := w.Flush(); err != nil {
		return "", fmt.Errorf("could not flush all input to output buffer > %w", err)
	}

	return fmt.Sprintf(`Today's Excerpt

Today's excerpt is a highlight created on %s.

%s

-- %s

`,
		c.CreateTime.Format("2006-01-02"), buffer.String(), c.Source,
	), nil
}
