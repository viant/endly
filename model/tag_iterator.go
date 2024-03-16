package model

import (
	"fmt"
	"github.com/viant/toolbox"
	"strings"
)

// TagIterator represents tag iterator to produce TagIndex
type TagIterator struct {
	Template string
	Min      int
	Max      int
	index    int
}

// Has return true if iterator has not been exhausted
func (i *TagIterator) Has() bool {
	return i.index <= i.Max
}

// Next eturns increment counter and checks if it is has next.
func (i *TagIterator) Next() bool {
	i.index++
	return i.Has()
}

// Index returns an index of the iterator
func (i *TagIterator) Index() string {
	return fmt.Sprintf(i.Template, i.index)
}

func decodeIteratorIfPresent(key string, result *Tag) string {
	iteratorStartPosition := strings.Index(key, "{")
	if iteratorStartPosition != -1 {
		iteratorEndPosition := strings.Index(key, "}")
		if iteratorEndPosition != -1 {
			iteratorConstrain := key[iteratorStartPosition+1 : iteratorEndPosition]
			pair := strings.Split(iteratorConstrain, "..")
			for i, value := range pair {
				pair[i] = strings.TrimSpace(value)
			}
			if len(pair) == 2 {
				result.Iterator = &TagIterator{
					Template: "%0" + toolbox.AsString(len(pair[1])) + "d",
					Min:      toolbox.AsInt(pair[0]),
					Max:      toolbox.AsInt(pair[1]),
					index:    0,
				}
				result.Iterator.index = result.Iterator.Min
				key = string(key[:iteratorStartPosition])
			}
		}
	}
	return key
}
