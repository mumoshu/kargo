package kargo

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Generator generates commands and config files
// required to plan and apply the deployment denoted by
// Config.
// This and Config is usually the two most interesting structs
// when you are going to use kargo as a Go library.
type Generator struct {
	GetValue GetValue
	// TempDir is the directory to write kustomize-build output
	// for use by kubectl-apply.
	TempDir string
	// TailLogs is set to true if you want kargo to tail the logs
	TailLogs bool
}

type Target int

const (
	Plan = iota
	Apply
)

type Cmd struct {
	Name string
	Args []string
	Dir  string
}

func (g *Generator) ExecCmds(c *Config, t Target) ([]Cmd, error) {
	if c.ArgoCD != nil {
		return g.cmdsArgoCD(c, t)
	}

	return g.cmds(c, t)
}

func (g *Generator) cmdsArgoCD(c *Config, t Target) ([]Cmd, error) {
	var (
		args []string
		err  error
	)

	args, err = AppendArgs(args, c.ArgoCD, g.GetValue, FieldTagArgoCD)
	if err != nil {
		return nil, err
	}

	if c.Helm != nil {
		args, err = AppendArgs(args, c.Helm, g.GetValue, FieldTagArgoCD)
		if err != nil {
			return nil, err
		}
	} else if c.Kustomize != nil {
		args, err = AppendArgs(args, c.Kustomize, g.GetValue, FieldTagArgoCD)
		if err != nil {
			return nil, err
		}
	}

	path := "."
	if c.Path != "" {
		path = c.Path
	}

	var cmds []Cmd

	if c.ArgoCD.Push {
		dir := strings.ReplaceAll(filepath.Base(c.ArgoCD.Repo), ".git", "")

		gitCloneArgs := append([]string{"clone"}, c.ArgoCD.Repo, dir)
		gitClone := Cmd{Name: "git", Args: gitCloneArgs}

		cpArgs := append([]string{"-r"}, filepath.Join(path, "*"), dir)
		cp := Cmd{Name: "cp", Args: cpArgs}

		gitAddArgs := append([]string{"-c"},
			fmt.Sprintf(
				"cd %s && git add .",
				dir,
			),
		)
		gitAdd := Cmd{Name: "bash", Args: gitAddArgs}

		gitCommitPushArgs := append([]string{"-c"},
			"git commit -m 'automated commit' && git push",
		)
		gitCommitPush := Cmd{Name: "bash", Args: gitCommitPushArgs}

		gitDiffArgs := append([]string{"-c"},
			"git diff",
		)
		gitDiff := Cmd{Name: "bash", Args: gitDiffArgs}

		cmds = append(cmds, gitClone, cp, gitAdd)

		if t == Plan {
			cmds = append(cmds, gitDiff)
		} else {
			cmds = append(cmds, gitCommitPush)
		}
	}

	pluginName := fmt.Sprintf("%s-%s", c.ArgoCD.Namespace, c.Name)
	args = append(args, "--config-management-plugin="+pluginName)
	// create or update the config manangement plugin configmap
	// with the generated ConfigManagementPlugin data.
	// and if not yet done so, patch the argocd repo server with the updated configmap
	// or restart the argocd repo server
	// OR
	// git-commit/push the cmp config file or the configmap containing it to a repo
	// so that some automation redeploys argocd-repo-server with it...
	// kargo cmp --namespace $argons $argo_repo_server_deploy apply/diff --name $plugin_name --type kompose_vals

	if t == Plan {
		// TODO
		// - Add some command to diff argocd-app-create changes
		// - kargo cmp --namespace $argons $argo_repo_server_deploy diff --name $plugin_name --type kompose_vals
		return append([]Cmd{}, cmds...), nil
	}

	argocdAppCreateArgs := append([]string{"app", "create"}, args...)
	argocdAppCreate := Cmd{Name: "argocd", Args: argocdAppCreateArgs}

	return append([]Cmd{argocdAppCreate}, cmds...), nil
}

func (g *Generator) cmds(c *Config, t Target) ([]Cmd, error) {
	var (
		args []string
		err  error
	)

	if c.Compose != nil {
		args, err = AppendArgs(args, c.Compose, g.GetValue, FieldTagCompose)
		if err != nil {
			return nil, err
		}

		dir := c.Path
		file := "docker-compose.yml"
		if strings.HasSuffix(dir, ".yml") {
			file = filepath.Base(dir)
			dir = filepath.Dir(dir)
		}

		composeArgs := append([]string{"compose", "-f", file}, args...)

		upArgs := []string{"up"}
		if !g.TailLogs {
			upArgs = append(upArgs, "-d")
		}

		convArgs := append([]string{}, composeArgs...)
		convArgs = append(convArgs, "convert")

		switch t {
		case Apply:
			if c.Compose.EnableVals {
				return []Cmd{
					{
						Name: "vals",
						Args: append([]string{"exec", "--stream-yaml", file, "--", "docker", "compose", "-f", "-"}, upArgs...),
						Dir:  dir,
					},
				}, nil
			}

			composeUp := Cmd{
				Name: "docker",
				Args: append(append([]string{}, composeArgs...), upArgs...),
				Dir:  dir,
			}
			return []Cmd{composeUp}, nil
		case Plan:
			composeConv := Cmd{
				Name: "docker",
				Args: convArgs,
				Dir:  dir,
			}
			return []Cmd{composeConv}, nil
		}
	}

	if c.Helm != nil {
		args, err = AppendArgs(args, c.Helm, g.GetValue, FieldTagHelm)
		if err != nil {
			return nil, err
		}

		repo := filepath.Base(c.Helm.Repo)
		helmRepoAdd := Cmd{Name: "helm", Args: []string{"repo", "add", repo, c.Helm.Repo}}
		helmUpgradeArgs := append([]string{"upgrade", "--install", c.Name, repo + "/" + c.Helm.Chart}, args...)

		switch t {
		case Apply:
			helmUpgrade := Cmd{Name: "helm", Args: helmUpgradeArgs}
			return []Cmd{helmRepoAdd, helmUpgrade}, nil
		case Plan:
			helmDiffArgs := append([]string{"diff"}, helmUpgradeArgs...)
			helmDiff := Cmd{Name: "helm", Args: helmDiffArgs}
			return []Cmd{helmRepoAdd, helmDiff}, nil
		}
	} else if c.Kustomize != nil {
		args, err = AppendArgs(args, c.Kustomize, g.GetValue, FieldTagKustomize)
		if err != nil {
			return nil, err
		}

		kustomizeEdit := Cmd{
			Name: "kustomize",
			Args: append([]string{"edit", "set", "image"}, args...),
			Dir:  c.Path,
		}

		tmpFile := filepath.Join(g.TempDir, "kustomize-built.yaml")

		kustomizeBuildArgs := []string{"build", "--output=" + tmpFile}
		if c.Path != "" {
			kustomizeBuildArgs = append(kustomizeBuildArgs, c.Path)
		}

		kustomizeBuild := Cmd{
			Name: "kustomize",
			Args: kustomizeBuildArgs,
		}

		kubectlArgs := []string{"-f", tmpFile, "--server-side=true"}

		kubectlDiff := Cmd{
			Name: "kubectl",
			Args: append([]string{"diff"}, kubectlArgs...),
		}

		kubectlApply := Cmd{
			Name: "kubectl",
			Args: append([]string{"apply"}, kubectlArgs...),
		}

		switch t {
		case Apply:
			return []Cmd{kustomizeEdit, kustomizeBuild, kubectlApply}, nil
		case Plan:
			return []Cmd{kustomizeEdit, kustomizeBuild, kubectlDiff}, nil
		}
	} else if c.Kompose != nil {
		args, err = AppendArgs(args, c.Kompose, g.GetValue, FieldTagKustomize)
		if err != nil {
			return nil, err
		}

		dir := c.Path
		file := "docker-compose.yml"
		if strings.HasSuffix(dir, ".yml") {
			file = filepath.Base(dir)
			dir = filepath.Dir(dir)
		}

		komposeConvertArgs := func(f, out string) []string {
			komposeConvertArgs := []string{"convert"}
			if out != "" {
				komposeConvertArgs = append(komposeConvertArgs, "--output="+out)
			} else {
				komposeConvertArgs = append(komposeConvertArgs, "--stdout")
			}
			if c.Path != "" {
				komposeConvertArgs = append(komposeConvertArgs, "-f", f)
			}
			komposeConvertArgs = append(komposeConvertArgs, args...)
			return komposeConvertArgs
		}

		kubectlArgs := func(f string) []string {
			return []string{"--server-side", "-f", f}
		}

		tailArgs := func() string {
			if g.TailLogs {
				return " && stern -l kompose.io.service!="
			}
			return ""
		}

		switch t {
		case Apply:
			if c.Kompose.EnableVals {
				script := append([]string{"kompose"}, komposeConvertArgs(
					"-",
					"",
				)...)
				script = append(script, "|", "kubectl", "apply")
				script = append(script, kubectlArgs("-")...)
				args := []string{
					"exec",
					"--stream-yaml",
					file,
					"--",
					"bash",
					"-c",
					strings.Join(script, " ") + tailArgs(),
				}
				return []Cmd{
					{
						Name: "vals",
						Args: args,
						Dir:  dir,
					},
				}, nil
			}

			script := append([]string{"kompose"}, komposeConvertArgs(
				file,
				"",
			)...)
			script = append(script, "|", "kubectl", "apply")
			script = append(script, kubectlArgs("-")...)
			args := []string{
				"-c",
				strings.Join(script, " ") + tailArgs(),
			}
			return []Cmd{
				{
					Name: "bash",
					Args: args,
					Dir:  dir,
				},
			}, nil
		case Plan:
			script := append([]string{"kompose"}, komposeConvertArgs(
				file,
				"",
			)...)
			script = append(script, "|", "kubectl", "diff")
			script = append(script, kubectlArgs("-")...)
			args := []string{
				"-c",
				strings.Join(script, " "),
			}
			return []Cmd{
				{
					Name: "bash",
					Args: args,
					Dir:  dir,
				},
			}, nil
		}
	} else {
		path := "."
		if c.Path != "" {
			path = c.Path
		}

		kubectlArgs := []string{"-f", path, "--server-side=true"}

		kubectlDiff := Cmd{
			Name: "kubectl",
			Args: append([]string{"diff"}, kubectlArgs...),
		}

		kubectlApply := Cmd{
			Name: "kubectl",
			Args: append([]string{"apply"}, kubectlArgs...),
		}

		switch t {
		case Apply:
			return []Cmd{kubectlApply}, nil
		case Plan:
			return []Cmd{kubectlDiff}, nil
		}
	}

	return nil, nil
}
