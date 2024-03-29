package ast

type Literal struct {
	Value string
	Type  string
	Quote string
}


func (b *Literal) Stringify() string {
	switch b.Type {
		case "string":
			if b.Quote != "" {
				return b.Quote + b.Value + b.Quote
			}
	}
	return b.Value
}