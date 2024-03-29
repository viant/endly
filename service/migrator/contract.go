package migrator

//MigratePostmanResponse represents a path to the workflow files
type MigratePostmanResponse struct {
	OutputPath string
	Success    bool
	Message    string
}

//MigratePostmanRequest represents a path to the postman files
type MigratePostmanRequest struct {
	CollectionPath string
	OutputPath     string
}
