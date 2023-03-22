package secret

import (
	"context"
	"encoding/json"
	"fmt"
	sjwt "github.com/golang-jwt/jwt/v4"
	"github.com/viant/afs"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/scy/cred"
	"github.com/viant/toolbox/url"
	"reflect"
)

type SecureRequest struct {
	Target    string
	_target   reflect.Type
	Source    []byte
	SourceURL string
	*scy.Resource
}

func (r *SecureRequest) Init() error {
	if r.SourceURL != "" {
		fs := afs.New()
		data, err := fs.DownloadWithURL(context.Background(), r.SourceURL)
		if err != nil {
			return err
		}
		r.Source = data
	}

	if r.Resource == nil {
		return nil
	}
	target, err := cred.TargetType(r.Target)
	if err != nil {
		return err
	}
	r._target = target
	if target != nil {
		r.Resource.SetTarget(target)
	}
	return nil
}

func (r *SecureRequest) Validate() error {
	if r.Resource == nil {
		return fmt.Errorf("URL was empty")
	}
	return nil
}

type SecureResponse struct{}

type RevealRequest struct {
	*scy.Resource
	Target  string
	_target reflect.Type
}

func (r *RevealRequest) Init() error {
	target, err := cred.TargetType(r.Target)
	if err != nil {
		return err
	}
	if target != nil {
		r.Resource.SetTarget(target)
	}
	return nil
}

func (r *RevealRequest) Validate() error {
	if r.Resource == nil {
		return fmt.Errorf("URL was empty")
	}
	return nil
}

type RevealResponse struct {
	URL     string
	Target  string
	Data    string
	Generic *cred.Generic
	AWS     *cred.Aws
	SHA1    *cred.SHA1
	SSH     *cred.SSH
	JWT     *cred.JwtConfig
	Basic   *cred.Basic
}

type SignJWTRequest struct {
	PrivateKey   *scy.Resource
	HMAC         *scy.Resource
	ExpiryInSec  int
	ClaimsURL    string
	UseClaimsMap bool
	ClaimsMap    map[string]interface{}
	Claims       *jwt.Claims
}

func (r *SignJWTRequest) Init() error {
	if r.ClaimsURL != "" {
		fs := afs.New()
		data, err := fs.DownloadWithURL(context.Background(), r.ClaimsURL)
		if err != nil {
			return err
		}
		r.Claims = &jwt.Claims{}
		if err = json.Unmarshal(data, r.Claims); err != nil {
			return err
		}
		if err = json.Unmarshal(data, &r.ClaimsMap); err != nil {
			return err
		}
	}
	if r.ExpiryInSec == 0 {
		r.ExpiryInSec = 360
	}
	return nil
}

func (r *SignJWTRequest) Validate() error {
	if r.Claims == nil && len(r.ClaimsMap) == 0 {
		return fmt.Errorf("claims was empty")
	}
	if r.PrivateKey == nil {
		return fmt.Errorf("PrivateKey „as empty")
	}
	return nil
}

type SignJWTResponse struct {
	TokenString string
}

type VerifyJWTRequest struct {
	PublicKey *scy.Resource
	HMAC      *scy.Resource
	CertURL   string
	Token     string
}

func (r *VerifyJWTRequest) Init() error {
	return nil
}

func (r *VerifyJWTRequest) Validate() error {
	if r.Token == "" {
		return fmt.Errorf("Token was empty")
	}
	if r.CertURL == "" && r.PublicKey == nil {
		return fmt.Errorf("PublicKey and CertURL were empty")
	}
	return nil
}

type VerifyJWTResponse struct {
	Error  string
	Valid  bool
	Token  *sjwt.Token
	Claims *jwt.Claims
}

//NewSecureRequestFromURL creates a request from URL
func NewSecureRequestFromURL(URL string) (*SecureRequest, error) {
	var request = &SecureRequest{}
	resource := url.NewResource(URL)
	return request, resource.Decode(request)
}

//NewRevealRequestFromURL creates a request from URL
func NewRevealRequestFromURL(URL string) (*RevealRequest, error) {
	var request = &RevealRequest{}
	resource := url.NewResource(URL)
	return request, resource.Decode(request)
}

//NewSignJWTRequest creates a request from URL
func NewSignJWTRequest(URL string) (*SignJWTRequest, error) {
	var request = &SignJWTRequest{}
	resource := url.NewResource(URL)
	return request, resource.Decode(request)
}

//NewVerifyJWTResponse creates a request from URL
func NewVerifyJWTResponse(URL string) (*VerifyJWTResponse, error) {
	var request = &VerifyJWTResponse{}
	resource := url.NewResource(URL)
	return request, resource.Decode(request)
}
