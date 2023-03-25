package kargo

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type config struct {
	FooBar string `flag1:"foo-bar"`
	Baz    string
	AFrom  string `flag1:"a"`
	B      string `flag1:""`
	C      string `kargo:""`
	Bool1  bool   `flag1:"bool1"`
}

func (c config) AppendArgs(args []string, get GetValue, key string) (*[]string, error) {
	if key == "flag2" {
		args = append(args, fmt.Sprintf("--prefixed-%s=%s", "foobar", c.FooBar))
		return &args, nil
	}

	return nil, nil
}

func TestAppendArgs(t *testing.T) {
	ok := &config{FooBar: "123", Baz: "456", AFrom: "ok", B: "b"}
	ng := &config{FooBar: "123", Baz: "456", AFrom: "ng", B: "b"}

	getValue := func(key string) (string, error) {
		if key != "ok" {
			return "", fmt.Errorf("unable to obtain value for key %q", key)
		}

		return strings.ToUpper(key), nil
	}

	t.Run("ok/unknown", func(t *testing.T) {
		check(t, ok, getValue, "unknown", []string{"--foobar=123", "--baz=456", "--afrom=OK", "--b=b", "--bool1=false"}, nil)
	})

	t.Run("ok/flag1", func(t *testing.T) {
		check(t, ok, getValue, "flag1", []string{"--foo-bar=123", "--baz=456", "--a=OK", "--bool1=false"}, nil)
	})

	t.Run("ok/flag2", func(t *testing.T) {
		check(t, ok, getValue, "flag2", []string{"--prefixed-foobar=123"}, nil)
	})

	t.Run("ng/flag1", func(t *testing.T) {
		check(t, ng, getValue, "flag1", nil, errors.New("field AFrom: unable to get value: unable to obtain value for key \"ng\""))
	})
}

func check(t *testing.T, input interface{}, get GetValue, key string, want []string, wantErr error) {
	t.Helper()

	args, err := AppendArgs(nil, input, get, key)
	if wantErr == nil {
		require.NoError(t, err)
	} else {
		require.EqualError(t, err, wantErr.Error())
	}
	require.Equal(t, want, args)
}
