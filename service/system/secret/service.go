package secret

import (
	"encoding/json"
	"fmt"
	"github.com/viant/afs/url"
	"github.com/viant/afsc/gs"
	"github.com/viant/endly"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/firebase"
	"github.com/viant/scy/auth/gcp"
	"github.com/viant/scy/auth/gcp/client"
	"github.com/viant/scy/auth/jwt/signer"
	"github.com/viant/scy/auth/jwt/verifier"
	"github.com/viant/scy/cred"
	"github.com/viant/scy/cred/secret"
	"google.golang.org/api/option"
	"reflect"
	"time"
)

// ServiceID represents a system process service id
const ServiceID = "secret"

type service struct {
	*endly.AbstractService
	*scy.Service
	Secret *secret.Service
}

func (s *service) secure(context *endly.Context, request *SecureRequest) (*SecureResponse, error) {
	var secret *scy.Secret

	if request._target != nil {
		instance := reflect.New(request._target).Interface()
		if err := json.Unmarshal(request.Source, instance); err != nil {
			return nil, err
		}
		secret = scy.NewSecret(instance, request.Resource)
	} else {
		secret = scy.NewSecret(string(request.Source), request.Resource)
	}
	err := s.Service.Store(context.Background(), secret)
	if err != nil {
		return nil, err
	}
	return &SecureResponse{}, nil
}

func (s *service) firebaseAuth(context *endly.Context, request *FirebaseAuthRequest) (*FirebaseAuthResponse, error) {

	gcpService := gcp.New(client.NewScy())
	var opts []option.ClientOption
	if serviceSecret := request.ServiceSecret; serviceSecret != nil && serviceSecret.URL != "" {
		request.ServiceSecret.SetTarget(reflect.TypeOf(&cred.Generic{}))
		aSecret, err := s.loadSecret(context, request.ServiceSecret)
		if err != nil {
			return nil, err
		}
		opts = append(opts, option.WithCredentialsJSON([]byte(aSecret.String())))
	} else {
		tokenSource := gcpService.TokenSource("https://www.googleapis.com/auth/cloud-platform")
		opts = append(opts, option.WithTokenSource(tokenSource))
	}
	identity, err := firebase.New(context.Background(), nil, opts...)
	if err != nil {
		return nil, err
	}

	var genericCred *cred.Generic
	if request.Credentials == nil {
		if request.Secret == nil {
			return nil, fmt.Errorf("secret was empty")
		}
		request.Secret.SetTarget(reflect.TypeOf(&cred.Generic{}))
		aSecret, err := s.loadSecret(context, request.Secret)
		if err != nil {
			return nil, err
		}
		genericCred = aSecret.Target.(*cred.Generic)
	}
	token, err := identity.InitiateBasicAuth(context.Background(), genericCred.Username, genericCred.Password)
	if err != nil {
		return nil, err
	}
	result := &FirebaseAuthResponse{
		Token: token,
	}
	return result, nil
}

func (s *service) loadSecret(context *endly.Context, resource *scy.Resource) (*scy.Secret, error) {
	var err error
	var aSecret *scy.Secret
	if url.IsRelative(resource.URL) {
		loc := resource.URL
		if resource.Key != "" {
			loc = loc + "|" + resource.Key
		}
		if aSecret, err = s.Secret.Lookup(context.Background(), secret.Resource(loc)); err != nil {
			return nil, err
		}
	} else {
		resource.SetTarget(reflect.TypeOf(&cred.Generic{}))
		if aSecret, err = s.Service.Load(context.Background(), resource); err != nil {
			return nil, err
		}
	}
	return aSecret, nil
}

func (s *service) reveal(context *endly.Context, request *RevealRequest) (*RevealResponse, error) {
	if serviceSecret := request.ServiceSecret; serviceSecret != nil && serviceSecret.URL != "" {
		request.ServiceSecret.SetTarget(reflect.TypeOf(&cred.Generic{}))
		aSecret, err := s.loadSecret(context, request.ServiceSecret)
		if err != nil {
			return nil, err
		}
		request.Options = append(request.Options, gs.ClientOptions([]option.ClientOption{option.WithCredentialsJSON([]byte(aSecret.String()))}))
	}
	secret, err := s.Service.Load(context.Background(), request.Resource)
	if err != nil {
		return nil, err
	}
	response := &RevealResponse{}
	response.Data = secret.String()
	switch actual := secret.Target.(type) {
	case *cred.Generic:
		response.Generic = actual
	case *cred.Basic:
		response.Basic = actual
	case *cred.JwtConfig:
		response.JWT = actual
	case *cred.Aws:
		response.AWS = actual
	case *cred.SSH:
		response.SSH = actual
	case *cred.SHA1:
		response.SHA1 = actual
	}
	return response, nil
}

func (s *service) signJWT(context *endly.Context, request *SignJWTRequest) (*SignJWTResponse, error) {
	jwtSigner := signer.New(&signer.Config{RSA: request.PrivateKey, HMAC: request.HMAC})

	if err := jwtSigner.Init(context.Background()); err != nil {
		return nil, err
	}

	var claims interface{} = request.Claims
	if request.Claims == nil || request.UseClaimsMap {
		claims = request.ClaimsMap
	}
	token, err := jwtSigner.Create(time.Duration(request.ExpiryInSec)*time.Second, claims)
	if err != nil {
		return nil, err
	}
	response := &SignJWTResponse{
		TokenString: token,
	}
	return response, nil
}

func (s *service) verifyJWT(context *endly.Context, request *VerifyJWTRequest) (*VerifyJWTResponse, error) {
	jwtVerifier := verifier.New(&verifier.Config{RSA: []*scy.Resource{request.PublicKey}, CertURL: request.CertURL, HMAC: request.HMAC})
	if err := jwtVerifier.Init(context.Background()); err != nil {
		return nil, err
	}
	jwtClaims, err := jwtVerifier.VerifyClaims(context.Background(), request.Token)
	response := &VerifyJWTResponse{Valid: true}
	if err != nil {
		response.Valid = false
		response.Error = err.Error()
		return response, nil
	}
	response.Token, _ = jwtVerifier.ValidaToken(context.Background(), request.Token)
	response.Claims = jwtClaims
	return response, nil
}

func (s *service) registerRoutes() {

	s.Register(&endly.Route{
		Action: "secure",
		RequestInfo: &endly.ActionInfo{
			Description: "secures secrets",
		},
		RequestProvider: func() interface{} {
			return &SecureRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SecureResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SecureRequest); ok {
				return s.secure(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "reveal",
		RequestInfo: &endly.ActionInfo{
			Description: "reveals secrets",
		},
		RequestProvider: func() interface{} {
			return &RevealRequest{}
		},
		ResponseProvider: func() interface{} {
			return &RevealResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RevealRequest); ok {
				return s.reveal(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "signJWT",
		RequestInfo: &endly.ActionInfo{
			Description: "signs JWT cliams",
		},
		RequestProvider: func() interface{} {
			return &SignJWTRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SignJWTResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SignJWTRequest); ok {
				return s.signJWT(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "signJWT",
		RequestInfo: &endly.ActionInfo{
			Description: "signs JWT claims",
		},
		RequestProvider: func() interface{} {
			return &SignJWTRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SignJWTResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SignJWTRequest); ok {
				return s.signJWT(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "verifyJWT",
		RequestInfo: &endly.ActionInfo{
			Description: "verify JWT claims",
		},
		RequestProvider: func() interface{} {
			return &VerifyJWTRequest{}
		},
		ResponseProvider: func() interface{} {
			return &VerifyJWTResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*VerifyJWTRequest); ok {
				return s.verifyJWT(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "firebaseAuth",
		RequestInfo: &endly.ActionInfo{
			Description: "authenticate with Firebase",
		},
		RequestProvider: func() interface{} {
			return &FirebaseAuthRequest{}
		},
		ResponseProvider: func() interface{} {
			return &FirebaseAuthResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*FirebaseAuthRequest); ok {
				return s.firebaseAuth(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

}

// New creates new system process service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
		Service:         scy.New(),
		Secret:          secret.New(),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
