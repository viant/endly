package ast


type Binary struct {
	X Node
	Op string
	Y Node
}

func (b *Binary) Stringify() string {
	return b.X.Stringify() + " " + b.Op + " " + b.Y.Stringify()
}