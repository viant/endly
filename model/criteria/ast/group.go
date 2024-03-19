package ast

type Group struct {
	X Node
}

func (g *Group) Stringify() string {
	return "(" + g.X.Stringify() + ")"
}