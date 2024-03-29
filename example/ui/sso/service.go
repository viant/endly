package sso

import (
	"errors"
	"fmt"
	"github.com/viant/dsc"
	"github.com/viant/toolbox"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
	"sync"
)

// Service represents sso service
type Service interface {
	SignUp(*SignUpRequest, *http.Request) *SignUpResponse

	SignIn(*SignInRequest) *SignInResponse
}

type service struct {
	config    *Config
	manager   dsc.Manager
	mutex     *sync.Mutex
	dsManager dsc.Manager
}

func setResponseError(response *BaseResponse, errorSource, message string) {
	response.Status = "error"
	response.ErrorSource = errorSource
	response.Error = message
}

func (s *service) getUser(email string) (*User, error) {
	var user = &User{}
	success, err := s.dsManager.ReadSingle(user, "SELECT email, name, hashedPassword, dateOfBirth FROM users WHERE email = ?", []interface{}{email}, nil)
	if err != nil {
		return nil, err
	}
	if success {
		return user, nil
	}
	return nil, nil
}

func (s *service) persistUser(user *User) error {
	_, _, err := s.dsManager.PersistSingle(user, "users", nil)
	return err
}

func (s *service) isIPEligible(httpRequest *http.Request) (bool, error) {
	IP := extractIp(httpRequest)
	var URL = fmt.Sprintf(s.config.IPLookupURL, IP)
	var response = &IpInfo{}
	err := toolbox.RouteToService("get", URL, nil, response)
	if err != nil {
		return false, err
	}
	return response.CountryCode == "" || strings.ToUpper(response.CountryCode) == "US", nil
}

func (s *service) SignUp(request *SignUpRequest, httpRequest *http.Request) *SignUpResponse {
	response := &SignUpResponse{
		BaseResponse: &BaseResponse{
			Status: "ok",
		},
	}
	ipEligible, err := s.isIPEligible(httpRequest)
	if err != nil {
		setResponseError(response.BaseResponse, "system", fmt.Sprintf("%v", err))
		return response
	}
	if !ipEligible {
		setResponseError(response.BaseResponse, "system", "registration from outside of US is no supported")
		return response
	}
	errorSource, err := request.Validate()
	if err != nil {
		setResponseError(response.BaseResponse, errorSource, fmt.Sprintf("%v", err))
		return response
	}

	if user, err := s.getUser(request.Email); user != nil || err != nil {
		if err != nil {
			setResponseError(response.BaseResponse, "system", fmt.Sprintf("unable to check user store: %v", err))
			return response
		}
		setResponseError(response.BaseResponse, "email", fmt.Sprintf("email %v has been already registered", request.Email))
		return response
	}

	user := request.User

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), 14)
	if err != nil {
		setResponseError(response.BaseResponse, "system", fmt.Sprintf("unable to hash user password: %v", err))
		return response
	}
	user.HashedPassword = string(hashedPassword)
	user.DateOfBirth, err = toolbox.ToTime(request.DataOfBirth, toolbox.DateFormatToLayout("yyyy-MM-dd"))
	if err != nil {
		setResponseError(response.BaseResponse, "dataOfBirth", fmt.Sprintf("%v: %v", request.DataOfBirth, err))
		return response
	}
	err = s.persistUser(user)
	if err != nil {
		setResponseError(response.BaseResponse, "system", fmt.Sprintf("unable persist user %v, %v", user.Email, err))
		return response
	}
	response.User = user
	response.LandingPage = request.LandingPage
	return response
}

func (s service) SignIn(request *SignInRequest) *SignInResponse {
	var response = &SignInResponse{
		BaseResponse: &BaseResponse{
			Status: "ok",
		},
	}
	user, err := s.getUser(request.Email)
	if err != nil {
		setResponseError(response.BaseResponse, "system", fmt.Sprintf("unable to get user %v", err))
		return response
	}

	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(request.Password)) != nil {
		setResponseError(response.BaseResponse, "email", "unable to find a user for provided credentials")
		return response
	}

	response.User = user
	response.LandingPage = request.LandingPage
	return response
}

// NewService creates a new SSO service.
func NewService(config *Config) (Service, error) {
	if config.DsConfig == nil {
		return nil, errors.New("DsConfig was empty")
	}

	manager, err := dsc.NewManagerFactory().Create(config.DsConfig)
	if err != nil {
		return nil, err
	}

	return &service{
		dsManager: manager,
		config:    config,
		mutex:     &sync.Mutex{},
	}, nil
}
