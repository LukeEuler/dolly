package tree

type Node interface {
	GetChildren() []Node
	Enter(Context) error
	Exist(Context) error
	WrapErr(Context, error) error
}

type DefaultNode struct {
	Children []Node
}

func (n *DefaultNode) Enter(Context) error {
	return nil
}

func (n *DefaultNode) Exist(_ Context) error {
	return nil
}

func (n *DefaultNode) GetChildren() []Node {
	return n.Children
}

func (n *DefaultNode) WrapErr(_ Context, err error) error {
	return err
}
