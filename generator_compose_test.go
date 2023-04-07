package kargo_test

import (
	"strings"
	"testing"

	"github.com/mumoshu/kargo"
	"github.com/stretchr/testify/require"
)

type cmd struct {
	Name string
	Args []string
	Dir  string
}

func TestGenerate_Compose(t *testing.T) {
	run := func(t *testing.T, targ kargo.Target, f func(fg *kargo.Generator, fc *kargo.Config), expected []cmd) {
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

		var got []cmd
		for _, c := range cmds {
			got = append(got, cmd{
				Name: c.Name,
				Args: c.Args.MustCollect(g.GetValue),
				Dir:  c.Dir,
			})
		}
		require.Equal(t, expected, got)
	}

	t.Run("apply", func(t *testing.T) {
		run(t, kargo.Apply, func(g *kargo.Generator, c *kargo.Config) {
			g.TailLogs = false
		}, []cmd{
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
		}, []cmd{
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
		}, []cmd{
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

	t.Run("apply with vals", func(t *testing.T) {
		run(t, kargo.Apply, func(g *kargo.Generator, c *kargo.Config) {
			g.TailLogs = false
			c.Compose.EnableVals = true
		}, []cmd{
			{
				Name: "vals",
				Args: []string{
					"exec",
					"--stream-yaml",
					"docker-compose.yml",
					"--",
					"docker",
					"compose",
					"-f",
					"-",
					"up",
					"-d",
				},
				Dir: "testdata/compose",
			},
		})
	})

	t.Run("plan with vals", func(t *testing.T) {
		run(t, kargo.Plan, func(g *kargo.Generator, c *kargo.Config) {
			g.TailLogs = false
			c.Compose.EnableVals = true
		}, []cmd{
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
