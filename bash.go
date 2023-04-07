package kargo

import "strings"

type BashScript struct {
	Script *Args
}

func NewBashScript(args *Args) *BashScript {
	return &BashScript{
		Script: args,
	}
}

func (b *BashScript) KargoValue(get GetValue) (string, error) {
	script, err := b.Script.Collect(get)
	if err != nil {
		return "", err
	}
	return strings.Join(script, " "), nil
}
