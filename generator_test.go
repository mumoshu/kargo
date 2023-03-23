package kargo_test

import (
	"strings"
	"testing"

	"github.com/mumoshu/kargo"
	"github.com/stretchr/testify/require"
)

func TestGenerate_ComposeApply(t *testing.T) {
	g := &kargo.Generator{
		GetValue: func(key string) (string, error) {
			return strings.ToUpper(key), nil
		},
		TailLogs: false,
	}

	c := &kargo.Config{
		Name:    "test",
		Path:    "testdata/compose",
		Compose: &kargo.Compose{},
	}

	t.Run("apply", func(t *testing.T) {
		cmds, err := g.ExecCmds(c, kargo.Apply)

		require.NoError(t, err)

		require.Equal(t, []kargo.Cmd{
			{
				Name: "docker",
				Args: []string{
					"compose",
					"-f",
					"docker-compose.yml",
					"up",
					"-d",
				},
				Dir: "testdata/compose",
			},
		}, cmds)
	})
}
