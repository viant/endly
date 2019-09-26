package copy

import (
	"errors"
	"github.com/viant/afs/option"
	"github.com/viant/afs/storage"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"strings"
)

//Rule represents transfer rule
type Rule struct {
	Matcher  *Matcher
	Compress bool `description:"flag to compress asset before sending over wire and to decompress (this option is only supported on scp or file scheme)"` //flag to compress asset before sending over wirte and to decompress (this option is only supported on scp or file proto)
	Substitution
	Source *url.Resource `required:"true" description:"source asset or directory"`
	Dest   *url.Resource `required:"true" description:"destination asset or directory"`
}

//New creates a new transfer
func New(source, dest *url.Resource, compress, expand bool, replace map[string]string) *Rule {
	return &Rule{
		Source:   source,
		Dest:     dest,
		Compress: compress,
		Substitution: Substitution{
			Expand:  expand,
			Replace: replace,
		},
	}
}

func (r Rule) Clone() *Rule {
	return &Rule{
		Source:   r.Source,
		Dest:     r.Dest,
		Compress: r.Compress,
		Matcher:  r.Matcher,
		Substitution: Substitution{
			Expand:  r.Expand,
			Replace: r.Replace,
			ExpandIf:    r.ExpandIf,
		},
	}
}

//SourceStorageOpts returns rule source store options
func (r *Rule) SourceStorageOpts(context *endly.Context) ([]storage.Option, error) {
	var result = make([]storage.Option, 0)
	if r.Matcher != nil {
		matcher, err := r.Matcher.Matcher()
		if err != nil {
			return nil, err
		}
		result = append(result, matcher)
	}
	return result, nil
}

//DestStorageOpts returns rule destination store options
func (r *Rule) DestStorageOpts(context *endly.Context, udfModifier option.Modifier) ([]storage.Option, error) {
	var result = make([]storage.Option, 0)
	if udfModifier != nil {
		result = append(result, udfModifier)
	} else if r.Expand || len(r.Replace) > 0 {
		modifier, err := NewModifier(context, r.ExpandIf, r.Replace, r.Expand)
		if err != nil {
			return nil, err
		}
		result = append(result, modifier)
	}
	return result, nil
}

//Init initialises transfer
func (r *Rule) Init() error {
	if r.Source != nil {
		if !strings.HasPrefix(r.Source.URL, "$") {
			if err := r.Source.Init(); err != nil {
				return err
			}
		}
	}
	if r.Dest != nil {
		if !strings.HasPrefix(r.Dest.URL, "$") {
			if err := r.Dest.Init(); err != nil {
				return err
			}
		}
	}
	return nil
}

//Validate checks if request is valid
func (r *Rule) Validate() error {
	if r.Source == nil {
		return errors.New("source was empty")
	}
	if r.Source.URL == "" {
		return errors.New("source.URL was empty")
	}
	if r.Dest == nil {
		return errors.New("dest was empty")
	}
	if r.Dest.URL == "" {
		return errors.New("dest.URL was empty")
	}
	return nil
}
