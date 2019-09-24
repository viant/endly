package copy

import (
	"github.com/viant/afs/matcher"
	"github.com/viant/afs/option"
	"github.com/viant/toolbox"
	"time"
)

//Matcher represents transfer source matcher
type Matcher struct {
	*matcher.Basic
	UpdatedBefore string
	UpdatedAfter  string
}



//Matcher return match handler or error
func (m Matcher) Matcher() (match option.Matcher, err error) {
	useTimeBased := m.UpdatedBefore != "" || m.UpdatedAfter != ""
	useBasic :=  m.Basic != nil
	var before, after *time.Time
	if m.UpdatedAfter != "" {
		if after, err = toolbox.TimeAt(m.UpdatedAfter);err != nil {
			return nil, err
		}
	}
	if m.UpdatedBefore != "" {
		if before, err = toolbox.TimeAt(m.UpdatedBefore);err != nil {
			return nil, err
		}
	}
	var matchers = make([]option.Matcher, 0)
	if useBasic {
		basic, err := matcher.NewBasic(m.Prefix, m.Suffix, m.Filter)
		if err != nil {
			return nil,  err
		}
		match = basic.Match
		matchers = append(matchers, basic.Match)
	}
	if useTimeBased {
		return matcher.NewModification(before, after, matchers...).Match, nil
	}
	return match, err
}

