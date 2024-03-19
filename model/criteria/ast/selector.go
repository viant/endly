package ast


type Selector struct {
	X string
}

func (b *Selector) Stringify() string {
	return  b.X
}
