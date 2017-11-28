package sso

import "github.com/viant/dsc"

type Service interface{

	SignUp(*SignUpRequest) *SignUpResponse

}


type service struct {
	config *Config
	manager dsc.Manager
}

func (s *service) SignUp(*SignUpRequest) *SignUpResponse {
	return nil
}



func NewService(config *Config) (Service, error) {
	return &service{
		config:config,
	}, nil
}




