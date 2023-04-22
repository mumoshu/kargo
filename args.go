package kargo

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
)

type Args struct {
	underlying []interface{}
}

func NewArgs(vs ...interface{}) *Args {
	return &Args{underlying: append([]interface{}{}, vs...)}
}

func (a *Args) Len() int {
	return len(a.underlying)
}

func (a *Args) AppendStrings(s ...string) *Args {
	for _, v := range s {
		a.underlying = append(a.underlying, v)
	}
	return a
}

func (a *Args) Append(vs ...interface{}) *Args {
	if a == nil {
		a = &Args{}
	}

	for _, v := range vs {
		v := v
		a.underlying = append(a.underlying, v)
	}
	return a
}

func (a *Args) AppendValueFromOutput(ref string) {
	a.underlying = append(a.underlying, DynArg{FromOutput: ref})
}

func (a *Args) AppendValueFromOutputWithPrefix(prefix, ref string) {
	a.underlying = append(a.underlying, DynArg{Prefix: prefix, FromOutput: ref})
}

func (a *Args) Visit(str func(string), out func(DynArg), flag func(KargoValueProvider)) {
	for _, x := range a.underlying {
		switch a := x.(type) {
		case string:
			str(a)
		case DynArg:
			out(a)
		case KargoValueProvider:
			flag(a)
		case *Args:
			a.Visit(str, out, flag)
		case []string:
			for _, s := range a {
				str(s)
			}
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
	}, func(a DynArg) {
		v, err := get(a.FromOutput)
		if err != nil {
			errors = append(errors, fmt.Errorf("after %s: %w", prev, err))
			return
		}
		args = append(args, a.Prefix+v)
		prev = a.FromOutput
	}, func(fvp KargoValueProvider) {
		v, err := fvp.KargoValue(get)
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
	}, func(a DynArg) {
		args = append(args, fmt.Sprintf("$(get %s with prefix %s)", a.FromOutput, a.Prefix))
	}, func(fvp KargoValueProvider) {
		args = append(args, fmt.Sprintf("%s", fvp))
	})

	return strings.Join(args, " ")
}

// DynArg is a dynamic argument that is resolved at runtime.
// It is used to compose a command-line argument like --foo=$bar,
// where $bar is a value of another kargo command.
type DynArg struct {
	// Prefix is a prefix to be prepended to the value of FromOutput.
	// For example, Prefix=foo= and FromOutput=bar will result in foo=$bar.
	// This is handy when you need to compose a command-line argument like --foo=$bar,
	// instead of --foo bar.
	Prefix string

	// FromOutput is a reference to an output of another kargo command.
	FromOutput string
}
