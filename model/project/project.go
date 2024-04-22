package project

// Project represents a project
type Project struct {
	ID          string `json:"SessionID,omitempty"`
	Name        string `json:"NAME,omitempty"`
	Description string `json:"DESCRIPTION,omitempty"`
}
