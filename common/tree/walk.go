package tree

func Walk(n Node, ctx Context) error {
	err := n.Enter(ctx)
	if err != nil {
		return n.WrapErr(ctx, err)
	}

	for _, item := range n.GetChildren() {
		err = Walk(item, ctx)
		if err != nil {
			return n.WrapErr(ctx, err)
		}
	}

	err = n.Exist(ctx)
	if err != nil {
		return n.WrapErr(ctx, err)
	}
	return nil
}
