package notifier

import "github.com/icyflame/kindle-my-clippings-parser/internal/parser"

// Notifier is the interface that any notifier which is going to send clippings somewhere should
// implement.
type Notifier interface {
	Notify(parser.Clipping) error
}

type Source struct {
	Name, Email string
}
