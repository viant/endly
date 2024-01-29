package kms

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"google.golang.org/api/cloudkms/v1"
	"io/ioutil"
	"log"
)

const (
	//ServiceID Google cloudkms Service ID.
	ServiceID = "gcp/kms"
)

// no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	client := &cloudkms.Service{}
	routes, err := gcp.BuildRoutes(client, nil, getClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}
	for _, route := range routes {
		route.OnRawRequest = InitRequest
		s.Register(route)
	}
	s.Register(&endly.Route{
		Action: "deployKey",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "deployKey", &DeployKeyRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &DeployKeyResponse{}),
		},
		RequestProvider: func() interface{} {
			return &DeployKeyRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DeployKeyResponse{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DeployKeyRequest); ok {
				output, err := s.deploy(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "deployKey", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "encrypt",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "encrypt", &EncryptRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &EncryptResponse{}),
		},
		RequestProvider: func() interface{} {
			return &EncryptRequest{}
		},
		ResponseProvider: func() interface{} {
			return &EncryptResponse{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*EncryptRequest); ok {
				output, err := s.encrypt(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "deployKey", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "decrypt",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "decrypt", &DecryptRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &DecryptResponse{}),
		},
		RequestProvider: func() interface{} {
			return &DecryptRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DecryptResponse{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DecryptRequest); ok {
				output, err := s.decrypt(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "deployKey", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

func (s *service) getKeyPolicy(ctx context.Context, service *cloudkms.ProjectsLocationsKeyRingsCryptoKeysService, keyURI string) (*Policy, error) {
	policyCall := service.GetIamPolicy(keyURI)
	policyCall.Context(ctx)
	policyResponse, err := policyCall.Do()
	if err != nil {
		err = toolbox.ReclassifyNotFoundIfMatched(err, keyURI)
		if toolbox.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	return &Policy{
		Bindings: policyResponse.Bindings,
		Version:  policyResponse.Version,
	}, nil
}

func (s *service) updateKeyPolicy(policy *Policy, ctx context.Context, service *cloudkms.ProjectsLocationsKeyRingsCryptoKeysService, keyURI string) (err error) {
	request := &cloudkms.SetIamPolicyRequest{Policy: &cloudkms.Policy{
		Bindings: policy.Bindings,
	}}
	setPolicyCall := service.SetIamPolicy(keyURI, request)
	setPolicyCall.Context(ctx)
	_, err = setPolicyCall.Do()
	return err
}

func (s *service) deploy(context *endly.Context, request *DeployKeyRequest) (*DeployKeyResponse, error) {
	response := &DeployKeyResponse{}
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}

	keyRingService := cloudkms.NewProjectsLocationsKeyRingsService(client.service)
	ringURI := gcp.ExpandMeta(context, request.ringURI)
	ringGetCall := keyRingService.Get(ringURI)
	ringGetCall.Context(client.Context())
	keyRing, err := ringGetCall.Do()
	err = toolbox.ReclassifyNotFoundIfMatched(err, ringURI)
	if err != nil && !toolbox.IsNotFoundError(err) {
		return nil, err
	}
	parent := gcp.ExpandMeta(context, request.parent)
	if keyRing == nil {
		keyRing = &cloudkms.KeyRing{}
		createRingCall := keyRingService.Create(parent, keyRing)
		createRingCall.KeyRingId(request.Ring)
		createRingCall.Context(client.Context())
		keyRing, err = createRingCall.Do()
		if err != nil {
			return nil, err
		}
	}
	service := cloudkms.NewProjectsLocationsKeyRingsCryptoKeysService(client.service)
	keyURI := keyRing.Name + "/cryptoKeys/" + request.Key
	keyCall := service.Get(keyURI)
	keyCall.Context(client.Context())
	cryptoKey, err := keyCall.Do()
	err = toolbox.ReclassifyNotFoundIfMatched(err, ringURI)
	if err != nil && !toolbox.IsNotFoundError(err) {
		return nil, err
	}
	if cryptoKey != nil {
		response.CryptoKey = cryptoKey
		if response.Policy, err = s.getKeyPolicy(client.Context(), service, keyURI); err != nil {
			return nil, err
		}
		if ShallUpdatePolicy(response.Policy, request.Policy) {
			response.Policy = request.Policy
			err = s.updateKeyPolicy(request.Policy, client.Context(), service, keyURI)
		}
		return response, err
	}
	createCall := service.Create(keyRing.Name, &cloudkms.CryptoKey{
		Purpose: request.Purpose,
		Labels:  request.Labels,
	})
	createCall.Context(client.Context())
	createCall.CryptoKeyId(request.Key)
	key, err := createCall.Do()
	if err == nil {
		if request.Policy != nil {
			err = s.updateKeyPolicy(request.Policy, client.Context(), service, keyURI)
		}
		response.Policy = request.Policy
		response.CryptoKey = key
		return response, err
	}
	return nil, err
}

func (s *service) encrypt(context *endly.Context, request *EncryptRequest) (*EncryptResponse, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	keyURI := gcp.ExpandMeta(context, request.keyURI)
	service := cloudkms.NewProjectsLocationsKeyRingsCryptoKeysService(client.service)

	if request.Source != nil {
		if request.PlainBase64Text, err = request.Source.DownloadBase64(); err != nil {
			return nil, err
		}
	}

	plainBase64Text := request.PlainBase64Text
	if len(request.PlainData) > 0 {
		plainBase64Text = base64.StdEncoding.EncodeToString(request.PlainData)
	}

	call := service.Encrypt(keyURI, &cloudkms.EncryptRequest{
		Plaintext: plainBase64Text,
	})
	call.Context(client.Context())
	response, err := call.Do()
	if err != nil {
		return nil, err
	}
	cipherData, err := base64.StdEncoding.DecodeString(response.Ciphertext)
	if err != nil {
		return nil, err
	}

	if request.Dest != nil {

		credentials := request.Dest.Credentials
		if credentials != "" {
			if location, err := context.Secrets.CredentialsLocation(credentials); err == nil {
				credentials = location
			}
		}
		storageService, err := storage.NewServiceForURL(request.Dest.URL, credentials)
		if err != nil {
			return nil, err
		}
		if err = storageService.Upload(request.Dest.URL, bytes.NewReader(cipherData)); err != nil {
			return nil, err
		}
	}

	return &EncryptResponse{
		CipherData:       cipherData,
		CipherBase64Text: response.Ciphertext,
	}, nil
}

func (s *service) decrypt(context *endly.Context, request *DecryptRequest) (*DecryptResponse, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	keyURI := gcp.ExpandMeta(context, request.keyURI)
	service := cloudkms.NewProjectsLocationsKeyRingsCryptoKeysService(client.service)
	if request.Source != nil {
		storageService, err := storage.NewServiceForURL(request.Source.URL, request.Source.Credentials)
		if err != nil {
			return nil, err
		}
		reader, err := storage.Download(storageService, request.Source.URL)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		data, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		_, err = base64.StdEncoding.DecodeString(string(data))
		if err == nil {
			request.CipherBase64Text = string(data)
		} else {
			request.CipherBase64Text = base64.StdEncoding.EncodeToString(data)
		}
	}

	cipherText := request.CipherBase64Text
	if len(request.CipherData) > 0 {
		cipherText = base64.StdEncoding.EncodeToString(request.CipherData)
	}

	if cipherText == "" {
		return nil, fmt.Errorf("cipher data was empty")
	}
	decryptCall := service.Decrypt(keyURI, &cloudkms.DecryptRequest{
		Ciphertext: cipherText,
	})
	decryptCall.Context(client.Context())
	response, err := decryptCall.Do()
	if err != nil {
		return nil, errors.Wrap(err, "failed to decrypt with "+keyURI)
	}
	plainData, _ := base64.StdEncoding.DecodeString(response.Plaintext)
	return &DecryptResponse{
		PlainData: plainData,
		PlainText: response.Plaintext,
	}, nil
}

// New creates a new cloudkms service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
