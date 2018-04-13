package model

//Dummy represents a dummy object
type Dummy struct {
	Id     int        `column:"id" primaryKey:"true" autoincrement:"true" `
	Name   string     `column:"name"`
	TypeId *int       `column:"type_id" json:",omitempty"`
	Type   *DummyType `transient:"true"`
}
