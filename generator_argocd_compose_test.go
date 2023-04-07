package kargo_test

import (
	"strings"
	"testing"

	"github.com/mumoshu/kargo"
	"github.com/stretchr/testify/require"
)

func TestGenerate_ArgoCD_Kompose(t *testing.T) {
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
			Kompose: &kargo.Kompose{},
			ArgoCD:  &kargo.ArgoCD{},
		}

		f(g, c)

		cmds, err := g.ExecCmds(c, targ)

		require.NoError(t, err)

		if len(cmds) == 0 && len(expected) == 0 {
			return
		}

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
			c.Name = "test"
			g.TailLogs = false
		}, []cmd{
			{
				Name: "bash",
				Args: []string{
					"-c",
					"argocd app create test --directory-recurse false --config-management-plugin=kargo ; argocd app set test --directory-recurse false --config-management-plugin=kargo",
				},
				Dir: "",
			},
		})
	})

	t.Run("apply with logs", func(t *testing.T) {
		run(t, kargo.Apply, func(g *kargo.Generator, c *kargo.Config) {
			g.TailLogs = true
		}, []cmd{
			{
				Name: "bash",
				Args: []string{
					"-c",
					"argocd app create test --directory-recurse false --config-management-plugin=kargo ; argocd app set test --directory-recurse false --config-management-plugin=kargo ; argocd app logs test --follow --tail=-1",
				},
				Dir: "",
			},
		})
	})

	t.Run("plan", func(t *testing.T) {
		run(t, kargo.Plan, func(g *kargo.Generator, c *kargo.Config) {
			g.TailLogs = false
		}, []cmd{})
	})

	t.Run("apply with vals", func(t *testing.T) {
		run(t, kargo.Apply, func(g *kargo.Generator, c *kargo.Config) {
			g.TailLogs = false
			c.Name = "test"
			c.Kompose.EnableVals = true
		}, []cmd{
			{
				Name: "bash",
				Args: []string{
					"-c",
					"argocd app create test --directory-recurse false --config-management-plugin=kargo ; argocd app set test --directory-recurse false --config-management-plugin=kargo",
				},
				Dir: "",
			},
		})
	})

	t.Run("plan with vals", func(t *testing.T) {
		run(t, kargo.Plan, func(g *kargo.Generator, c *kargo.Config) {
			g.TailLogs = false
			c.Kompose.EnableVals = true
		}, []cmd{})
	})
}
