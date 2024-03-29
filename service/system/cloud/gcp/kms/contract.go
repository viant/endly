package kms

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/service/system/cloud/gcp"
	"github.com/viant/toolbox"
	"google.golang.org/api/cloudkms/v1"
)

type KeyInfo struct {
	Region string
	Key    string
	Ring   string
	keyURI string
}

// DeployKeyRequest represents a deploy KeyInfo request
type DeployKeyRequest struct {
	KeyInfo
	Labels  map[string]string
	Purpose string
	*Policy
	PolicyVersion int64
	ringURI       string
	parent        string
}

// DeployKeyRequest represents a deploy KeyInfo response
type DeployKeyResponse struct {
	*cloudkms.CryptoKey
	Policy *Policy
}

/*
   members:
   - serviceAccount:engineering-e2e-test-sc@viant-e2e.iam.gserviceaccount.com
   role: roles/cloudkms.admin


*/

// EncryptRequest represents encrypt request
type EncryptRequest struct {
	KeyInfo
	PlainBase64Text string
	PlainData       []byte
	Source          *location.Resource
	Dest            *location.Resource
}

// EncryptResponse represents encrypt response
type EncryptResponse struct {
	CipherData       []byte
	CipherBase64Text string
}

// DecryptRequest represents decrypt response
type DecryptRequest struct {
	KeyInfo
	CipherData       []byte
	CipherBase64Text string
	Source           *location.Resource
}

// DecryptResponse represents decrypt response
type DecryptResponse struct {
	PlainData []byte
	PlainText string
}

// NewEncryptRequest creates a new DecryptRequest
func NewDecryptRequest(region, ring, keyId string, data []byte) *DecryptRequest {
	return &DecryptRequest{
		KeyInfo: KeyInfo{
			Key:    keyId,
			Region: region,
			Ring:   ring,
		},
		CipherData: data,
	}
}

// NewEncryptRequest creates a new EncryptRequest
func NewEncryptRequest(region, ring, keyId string, plainData []byte) *EncryptRequest {
	return &EncryptRequest{
		KeyInfo: KeyInfo{
			Key:    keyId,
			Region: region,
			Ring:   ring,
		},
		PlainData: plainData,
	}
}

// NewDeployKeyRequest creates a new DeployKeyRequest
func NewDeployKeyRequest(region, ring, keyId, purpose string) *DeployKeyRequest {
	return &DeployKeyRequest{
		KeyInfo: KeyInfo{
			Key:    keyId,
			Region: region,
			Ring:   ring,
		},
		Purpose: purpose,
	}
}

// Init initializes request
func (r *EncryptRequest) Init() error {
	return r.KeyInfo.Init()
}

// Init initializes request
func (r *DecryptRequest) Init() error {
	return r.KeyInfo.Init()
}

// Init initializes key
func (r *KeyInfo) Init() error {
	if r.Region == "" {
		r.Region = gcp.DefaultRegion
	}
	r.keyURI = fmt.Sprintf("projects/${gcp.projectID}/locations/%v/keyRings/%v/cryptoKeys/%v",
		r.Region,
		r.Ring, r.Key)
	return nil
}

func (r *EncryptRequest) Validate() error {
	if r.Key == "" {
		return errors.New("key was empty")
	}
	if r.Ring == "" {
		return errors.New("ring was empty")
	}
	if len(r.PlainData) == 0 && r.Source == nil {
		return errors.New("plainData was empty")
	}
	return nil
}

var allowedPurposes = map[string]bool{
	"ENCRYPT_DECRYPT":                true,
	"CRYPTO_KEY_PURPOSE_UNSPECIFIED": true,
	"ASYMMETRIC_SIGN":                true,
	"ASYMMETRIC_DECRYPT":             true,
}

func (r *KeyInfo) Validate() error {
	if r.Key == "" {
		return errors.New("name was empty")
	}

	if r.Ring == "" {
		return errors.New("ring was empty")
	}
	return nil
}

func (r *DeployKeyRequest) Validate() error {
	if r.Purpose == "" {
		return errors.New("purpose was empty")
	}
	if !allowedPurposes[r.Purpose] {
		return fmt.Errorf("unsupported purpose: %v, supported: %v", r.Purpose, toolbox.MapKeysToStringSlice(allowedPurposes))
	}
	return r.KeyInfo.Validate()
}

func (r *DeployKeyRequest) Init() error {
	if r.Region == "" {
		r.Region = gcp.DefaultRegion
	}
	if err := r.KeyInfo.Init(); err != nil {
		return err
	}
	r.ringURI = fmt.Sprintf("projects/${gcp.projectID}/locations/%v/keyRings/%v", r.Region, r.Ring)
	r.parent = fmt.Sprintf("projects/${gcp.projectID}/locations/%v", r.Region)
	return nil
}
