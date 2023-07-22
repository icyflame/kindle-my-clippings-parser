package utils

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/icyflame/kindle-my-clippings-parser/internal/parser"
	"v.io/x/lib/textutil"
)

// TextWidth is the width of the final email. All lines in the email will be less than this length.
const TextWidth = 100

// MakePlaintextEmailFromClipping ...
func MakePlaintextEmailFromClipping(c parser.Clipping) (string, error) {
	var buffer bytes.Buffer
	w := textutil.NewUTF8WrapWriter(&buffer, TextWidth)
	if _, err := w.Write([]byte(c.Text)); err != nil {
		return "", fmt.Errorf("could not make plaintext email > %w", err)
	}

	if err := w.Flush(); err != nil {
		return "", fmt.Errorf("could not flush all input to output buffer > %w", err)
	}

	clippingFormatted := strings.ReplaceAll("    "+buffer.String(), "\n", "\n    ")

	return fmt.Sprintf(`Today's excerpt is a highlight created on %s.

%s

-- %s

`,
		c.CreateTime.Format("2006-01-02"), clippingFormatted, c.Source,
	), nil
}
