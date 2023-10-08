package supplementer

import (
	"regexp"
	"sort"

	"github.com/icyflame/kindle-my-clippings-parser/internal/parser"
	"go.uber.org/zap"
)

type Bookcision struct {
	SourceRegex *regexp.Regexp
	Logger      *zap.Logger
}

// Merge ...
func (b *Bookcision) Merge(kindleInput parser.Clippings, bookcision parser.Clippings) (parser.Clippings, error) {
	output := make(parser.Clippings, 0)

	sort.Sort(kindleInput)
	sort.Sort(bookcision)

	j := 0
	for _, kindle := range kindleInput {
		if b.SourceRegex.MatchString(kindle.Source) && kindle.Text == parser.KindleClippingLimitMessage {
			for ; j < len(bookcision); j++ {
				// Kindle has moved ahead of bookcision, so increase bookcision and see whether we
				// find something that matches
				if bookcision[j].LocationInSource.Start < kindle.LocationInSource.Start {
					continue
				}

				// Bookcision has moved past Kindle, so there is no hope of finding anything in
				// Bookcision either. Just break out of this loop.
				if bookcision[j].LocationInSource.Start > kindle.LocationInSource.Start {
					break
				}

				// Bookcision has a clipping from the same location as Kindle. Replace the text from
				// Bookcision into the Kindle list.
				if bookcision[j].LocationInSource.Start == kindle.LocationInSource.Start {
					b.Logger.Debug("supplemented using bookcision", zap.Any("original", kindle), zap.Any("supplemented", bookcision[j]))
					kindle.Text = bookcision[j].Text
					break
				}
			}
		}

		output = append(output, kindle)
	}

	return output, nil
}
