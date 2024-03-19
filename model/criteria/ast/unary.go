package ast

type Unary struct {
	X Node
	Op string
}

func (b *Unary) Stringify() string {
	return b.Op + " " + b.X.Stringify()
}

