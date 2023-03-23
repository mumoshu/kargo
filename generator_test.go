package kargo_test

import (
	"strings"
	"testing"

	"github.com/mumoshu/kargo"
	"github.com/stretchr/testify/require"
)

func TestGenerate_Compose(t *testing.T) {
	run := func(t *testing.T, targ kargo.Target, f func(fg *kargo.Generator, fc *kargo.Config), expected []kargo.Cmd) {
		t.Helper()

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

		f(g, c)

		cmds, err := g.ExecCmds(c, targ)

		require.NoError(t, err)

		require.Equal(t, expected, cmds)
	}

	t.Run("apply", func(t *testing.T) {
		run(t, kargo.Apply, func(g *kargo.Generator, c *kargo.Config) {
			g.TailLogs = false
		}, []kargo.Cmd{
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
		})
	})

	t.Run("apply with logs", func(t *testing.T) {
		run(t, kargo.Apply, func(g *kargo.Generator, c *kargo.Config) {
			g.TailLogs = true
		}, []kargo.Cmd{
			{
				Name: "docker",
				Args: []string{
					"compose",
					"-f",
					"docker-compose.yml",
					"up",
				},
				Dir: "testdata/compose",
			},
		})
	})

	t.Run("plan", func(t *testing.T) {
		run(t, kargo.Plan, func(g *kargo.Generator, c *kargo.Config) {
			g.TailLogs = false
		}, []kargo.Cmd{
			{
				Name: "docker",
				Args: []string{
					"compose",
					"-f",
					"docker-compose.yml",
					"convert",
				},
				Dir: "testdata/compose",
			},
		})
	})
}
