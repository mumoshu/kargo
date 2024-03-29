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
			ArgoCD: &kargo.ArgoCD{
				Repo:     "exmaple.com/myrepo",
				DestName: "myekscluster",
				Path:     "to/where/push/manifests",
			},
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
			c.ArgoCD.Project = "testproj"
			c.ArgoCD.Server = "https://localhost:8080"
		}, []cmd{
			{
				Name: "bash",
				Args: []string{
					"-vxc",
					"argocd login https://localhost:8080 ; argocd proj create testproj --server https://localhost:8080 ; aws eks update-kubeconfig --name myekscluster --alias myekscluster ; argocd cluster add myekscluster ; argocd repo add exmaple.com/myrepo ; argocd app create test --directory-recurse --project testproj --server https://localhost:8080 --dest-name myekscluster --config-management-plugin=kargo --path to/where/push/manifests --repo exmaple.com/myrepo ; argocd app set test --directory-recurse --project testproj --server https://localhost:8080 --dest-name myekscluster --config-management-plugin=kargo --path to/where/push/manifests --repo exmaple.com/myrepo",
				},
				Dir: "",
			},
		})
	})

	t.Run("apply with logs", func(t *testing.T) {
		run(t, kargo.Apply, func(g *kargo.Generator, c *kargo.Config) {
			g.TailLogs = true
			c.ArgoCD.Project = "testproj"
			c.ArgoCD.Server = "https://localhost:8080"
		}, []cmd{
			{
				Name: "bash",
				Args: []string{
					"-vxc",
					"argocd login https://localhost:8080 ; argocd proj create testproj --server https://localhost:8080 ; aws eks update-kubeconfig --name myekscluster --alias myekscluster ; argocd cluster add myekscluster ; argocd repo add exmaple.com/myrepo ; argocd app create test --directory-recurse --project testproj --server https://localhost:8080 --dest-name myekscluster --config-management-plugin=kargo --path to/where/push/manifests --repo exmaple.com/myrepo ; argocd app set test --directory-recurse --project testproj --server https://localhost:8080 --dest-name myekscluster --config-management-plugin=kargo --path to/where/push/manifests --repo exmaple.com/myrepo ; argocd app logs test --follow --tail=-1",
				},
				Dir: "",
			},
		})
	})

	t.Run("plan", func(t *testing.T) {
		run(t, kargo.Plan, func(g *kargo.Generator, c *kargo.Config) {
			g.TailLogs = false
			c.ArgoCD.Server = "https://localhost:8080"
		}, []cmd{})
	})

	t.Run("apply with vals", func(t *testing.T) {
		run(t, kargo.Apply, func(g *kargo.Generator, c *kargo.Config) {
			g.TailLogs = false
			c.Name = "test"
			c.Kompose.EnableVals = true
			c.ArgoCD.Project = "testproj"
			c.ArgoCD.Server = "https://localhost:8080"
		}, []cmd{
			{
				Name: "bash",
				Args: []string{
					"-vxc",
					"argocd login https://localhost:8080 ; argocd proj create testproj --server https://localhost:8080 ; aws eks update-kubeconfig --name myekscluster --alias myekscluster ; argocd cluster add myekscluster ; argocd repo add exmaple.com/myrepo ; argocd app create test --directory-recurse --project testproj --server https://localhost:8080 --dest-name myekscluster --config-management-plugin=kargo --path to/where/push/manifests --repo exmaple.com/myrepo ; argocd app set test --directory-recurse --project testproj --server https://localhost:8080 --dest-name myekscluster --config-management-plugin=kargo --path to/where/push/manifests --repo exmaple.com/myrepo",
				},
				Dir: "",
			},
		})
	})

	t.Run("plan with vals", func(t *testing.T) {
		run(t, kargo.Plan, func(g *kargo.Generator, c *kargo.Config) {
			g.TailLogs = false
			c.Kompose.EnableVals = true
			c.ArgoCD.Server = "https://localhost:8080"
		}, []cmd{})
	})
}
