package supplementer

import "github.com/icyflame/kindle-my-clippings-parser/internal/parser"

type Supplementer interface {
	Merge(parser.Clippings, parser.Clippings) (parser.Clippings, error)
}
