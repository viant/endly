package transfer

import 	"github.com/viant/afs/matcher"

//Matcher represents transfer source matcher
type Matcher struct {
	matcher.Basic
	UpdatedBefore string
	UpdatedAfter  string
}
