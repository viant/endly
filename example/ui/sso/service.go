package sso

import (
	"github.com/viant/dsc"
	"sync"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"github.com/viant/toolbox"
)



type Service interface{

	SignUp(*SignUpRequest) *BaseResponse

}

type service struct {
	config *Config
	manager dsc.Manager
	mutex *sync.Mutex
	registry map[string]*User
}


func setResponseError(response *BaseResponse, errorSource, message string) {
	response.Status = "error"
	response.ErrorSource = errorSource
	response.Error = message
}


func (s *service) getUser(email string) (*User, error) {
	if result, existing := s.registry[email];existing {
		return result, nil
	}
	return nil, nil
}


func (s *service) persistUser(user *User) error {
	s.registry[user.Email] = user
	return nil
}


func (s *service) SignUp(request *SignUpRequest) *BaseResponse {
	response := &BaseResponse{
		Status:"ok",
	}

	 errorSource, err :=request.Validate();
	 if err != nil {
		setResponseError(response, errorSource, fmt.Sprintf("%v", err))
		return response
	}

	if user, err := s.getUser(request.Email); user != nil || err != nil {
		if err != nil {
			setResponseError(response, "system", fmt.Sprintf("unable to check user store: %v", err))
			return response
		}
		setResponseError(response, "user", fmt.Sprintf("email %v has been already registered", request.Email))
		return response
	}

	user := request.User
	user.EncryptedPassword, err = HashPassword(request.Password)
	if err != nil {
		setResponseError(response, "system", fmt.Sprintf("unable to hash user password: %v", err))
		return response
	}
	user.DataOfBirth, err = toolbox.ToTime(request.DataOfBirth, toolbox.DateFormatToLayout("yyyy-MM-dd"))
	if err != nil {
		setResponseError(response, "dataOfBirth", fmt.Sprintf("%v: %v", request.DataOfBirth,  err))
		return response
	}
	err =  s.persistUser(user)
	if err != nil {
		setResponseError(response, "system", fmt.Sprintf("unable persist user", user.Email,  err))
		return response
	}
	response.LandingPage = request.LandingPage
	return response
}



func NewService(config *Config) (Service, error) {
	return &service{
		config:config,
		mutex: &sync.Mutex{},
		registry: make(map[string]*User),
	}, nil
}



func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}


