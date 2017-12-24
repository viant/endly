package sso

import (
	"errors"
)

//SignUpRequest represents signup request
type SignUpRequest struct {
	*User
	DataOfBirth string `json:"dateOfBirth"`
	Password    string `json:"password"`
	LandingPage string `json:"landingPage"`
}

//SignUpResponse represents signup response
type SignUpResponse struct {
	*BaseResponse
	*User
	LandingPage string `json:"landingPage"`
}

//SignInRequest represents signin request
type SignInRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	RememberMe  string `json:"rememberMe"`
	LandingPage string `json:"landingPage"`
}

//SignInResponse represents signin response
type SignInResponse struct {
	*BaseResponse
	*User
	LandingPage string `json:"landingPage"`
}

//BaseResponse represents base response
type BaseResponse struct {
	Status      string `json:"status"`
	Error       string `json:"error"`
	ErrorSource string `json:"errorSource"`
}

//Validate check request all data if it is provided or valid.
func (r *SignUpRequest) Validate() (string, error) {
	if r.User == nil || r.Name == "" {
		return "name", errors.New("name was empty")
	}
	if r.User == nil || r.Email == "" {
		return "email", errors.New("email was empty")
	}
	if r.Password == "" {
		return "password", errors.New("password was empty")
	}
	if r.DataOfBirth == "" {
		return "dataOfBirth", errors.New("data of birth was empty")
	}
	return "", nil
}
