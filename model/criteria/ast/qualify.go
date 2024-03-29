package ast

type Qualify struct {
	X Node
}

func (b *Qualify) Stringify() string {
	return  b.X.Stringify()
}