package kargo

import "strings"

type Join struct {
	Args *Args
}

func NewJoin(args *Args) *Join {
	return &Join{
		Args: args,
	}
}

func (b *Join) KargoValue(get GetValue) (string, error) {
	args, err := b.Args.Collect(get)
	if err != nil {
		return "", err
	}
	return strings.Join(args, ""), nil
}
