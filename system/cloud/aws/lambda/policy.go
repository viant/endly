package lambda


type Principal struct {
	Service string
}

type Statement struct {
	Sid string
	Effect string
	Action string
	Resource string
	Principal *Principal
	Condition map[string]map[string]string
}

type Policy struct {
	Version string
	ID string `json:"Id"`
	Statement []*Statement
}

