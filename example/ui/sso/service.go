package sso

import (
	"errors"
	"fmt"
	"github.com/viant/dsc"
	"github.com/viant/toolbox"
	"golang.org/x/crypto/bcrypt"
	"sync"
)


//Service represents sso service
type Service interface {
	SignUp(*SignUpRequest) *SignUpResponse

	SignIn(*SignUpRequest) *SignInResponse
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

func (s *service) SignUp(request *SignUpRequest) *SignUpResponse {
	response := &SignUpResponse{
		BaseResponse: &BaseResponse{
			Status: "ok",
		},
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

func (s service) SignIn(request *SignUpRequest) *SignInResponse {
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
