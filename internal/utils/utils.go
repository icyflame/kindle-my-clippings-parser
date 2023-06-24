package utils

import (
	"bytes"
	"fmt"

	"github.com/icyflame/kindle-my-clippings-parser/internal/parser"
	"v.io/x/lib/textutil"
)

// MakePlaintextEmailFromClipping ...
func MakePlaintextEmailFromClipping(c parser.Clipping) string {
	var buffer bytes.Buffer
	w := textutil.NewUTF8WrapWriter(&buffer, 80)
	if _, err := w.Write([]byte(c.Text)); err != nil {
		fmt.Println("there was an error: %v", err)
	}

	return fmt.Sprintf(`Today's Excerpt

Today's excerpt is a highlight created on %s.

> %s

-- %s (p.%s)

`,
		c.CreateTime.Format("2006-01-02"), c.Text, c.Source, c.Page,
	)
}
