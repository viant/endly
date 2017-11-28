package sso


type SignUpRequest struct {
	Email string
	Name string
	Password string
	DataOfBirth string
	LandingPage string
}



type SignUpResponse struct {
	Status string
	Message string
	LandingPage string
}
