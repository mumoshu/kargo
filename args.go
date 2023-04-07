package kargo

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
)

type Args struct {
	underlying []interface{}
}

func (a *Args) Len() int {
	return len(a.underlying)
}

func (a *Args) Append(s ...string) *Args {
	for _, v := range s {
		a.underlying = append(a.underlying, v)
	}
	return a
}

func (a *Args) Add(vs ...interface{}) *Args {
	for _, v := range vs {
		v := v
		a.underlying = append(a.underlying, v)
	}
	return a
}

func (a *Args) AddValueFromOutput(ref string) {
	a.underlying = append(a.underlying, DynArg{FromOutput: ref})
}

func (a *Args) Visit(str func(string), out func(string), flag func(flagValueProvider)) {
	for _, x := range a.underlying {
		switch a := x.(type) {
		case string:
			str(a)
		case DynArg:
			out(a.FromOutput)
		case flagValueProvider:
			flag(a)
		default:
			panic(fmt.Sprintf("unexpected type(%T) of item: %q", a, a))
		}
	}
}

func (a *Args) Collect(get func(string) (string, error)) ([]string, error) {
	if a == nil {
		return nil, nil
	}

	var (
		prev   string
		args   []string
		errors []error
	)

	a.Visit(func(s string) {
		args = append(args, s)
		prev = s
	}, func(s string) {
		v, err := get(s)
		if err != nil {
			errors = append(errors, fmt.Errorf("after %s: %w", prev, err))
			return
		}
		args = append(args, v)
		prev = s
	}, func(fvp flagValueProvider) {
		v, err := fvp.FlagValue(get)
		if err != nil {
			errors = append(errors, fmt.Errorf("after %s: %w", prev, err))
			return
		}
		args = append(args, v)
		prev = v
	})

	if len(errors) == 1 {
		return nil, errors[0]
	}

	if len(errors) > 1 {
		return nil, multierror.Append(nil, errors...)
	}

	return args, nil
}

func (a *Args) MustCollect(get func(string) (string, error)) []string {
	got, err := a.Collect(get)
	if err != nil {
		panic(err)
	}
	return got
}

func (a *Args) String() string {
	var args []string

	a.Visit(func(s string) {
		args = append(args, s)
	}, func(s string) {
		args = append(args, fmt.Sprintf("$(get %s)", s))
	}, func(fvp flagValueProvider) {
		args = append(args, fmt.Sprintf("%s", fvp))
	})

	return strings.Join(args, " ")
}

type DynArg struct {
	FromOutput string
}
