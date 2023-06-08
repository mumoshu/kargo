package cmd

import "github.com/mumoshu/kargo"

type Option func(*kargo.Cmd)

func Args(args ...interface{}) Option {
	return func(c *kargo.Cmd) {
		c.Args = kargo.NewArgs(args...)
	}
}

func Dir(dir string) Option {
	return func(c *kargo.Cmd) {
		c.Dir = dir
	}
}

func New(id, name string, opts ...Option) kargo.Cmd {
	c := kargo.Cmd{
		ID:   id,
		Name: name,
	}

	for _, opt := range opts {
		opt(&c)
	}

	return c
}
