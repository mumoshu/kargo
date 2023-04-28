package kargo

import (
	"errors"
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
	Args *Args
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
		args    *Args
		appArgs *Args
		err     error
	)

	appArgs, err = AppendArgs(appArgs, c, FieldTagArgoCDApp)
	if err != nil {
		return nil, err
	}

	if c.Helm != nil {
		appArgs, err = AppendArgs(appArgs, c.Helm, FieldTagArgoCDApp)
		if err != nil {
			return nil, err
		}
	} else if c.Kustomize != nil {
		appArgs, err = AppendArgs(appArgs, c.Kustomize, FieldTagArgoCDApp)
		if err != nil {
			return nil, err
		}
	} else if c.Compose != nil {
		return nil, fmt.Errorf("compose is not supported with argocd")
	}

	var path string
	if c.ArgoCD.Path != "" {
		path = c.ArgoCD.Path
	}

	var cmds []Cmd

	if c.ArgoCD.Push {
		localRepoDir := strings.ReplaceAll(filepath.Base(c.ArgoCD.Repo), ".git", "")

		gitCloneArgs := NewArgs("clone", c.ArgoCD.Repo, localRepoDir)
		gitClone := Cmd{Name: "git", Args: gitCloneArgs}

		localConfigDir := filepath.Join(localRepoDir, path)

		cpArgs := NewArgs("-r", filepath.Join(c.Path, "*"), localConfigDir)
		cp := Cmd{Name: "cp", Args: cpArgs}

		gitAddArgs := NewArgs("-c",
			fmt.Sprintf(
				"cd %s && git add .",
				localRepoDir,
			),
		)
		gitAdd := Cmd{Name: "bash", Args: gitAddArgs}

		gitCommitPushArgs := NewArgs("-c",
			"git commit -m 'automated commit' && git push",
		)
		gitCommitPush := Cmd{Name: "bash", Args: gitCommitPushArgs}

		gitDiffArgs := NewArgs("-c",
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

	{
		var server string
		if c.ArgoCD.Server != "" {
			server = c.ArgoCD.Server
		} else if c.ArgoCD.ServerFrom != "" {
			server, err = g.GetValue(c.ArgoCD.ServerFrom)
			if err != nil {
				return nil, fmt.Errorf("unable to obtain argocd.ServerFrom: %v", err)
			}
		}
		if server != "" {
			args = args.AppendStrings("--server", server)
		}
	}

	{
		var username string
		if c.ArgoCD.Username != "" {
			username = c.ArgoCD.Username
		} else if c.ArgoCD.UsernameFrom != "" {
			username, err = g.GetValue(c.ArgoCD.UsernameFrom)
			if err != nil {
				return nil, fmt.Errorf("unable to obtain argocd.UsernameFrom: %v", err)
			}
		}
		if username != "" {
			args = args.AppendStrings("--username", username)
		}
	}

	{
		var password string
		if c.ArgoCD.Password != "" {
			password = c.ArgoCD.Password
		} else if c.ArgoCD.PasswordFrom != "" {
			password, err = g.GetValue(c.ArgoCD.PasswordFrom)
			if err != nil {
				return nil, fmt.Errorf("unable to obtain argocd.PasswordFrom: %v", err)
			}
		}
		if password != "" {
			args = args.AppendStrings("--password", password)
		}
	}

	{
		var insecure string
		if c.ArgoCD.Insecure {
			insecure = fmt.Sprintf("%v", c.ArgoCD.Insecure)
		} else if c.ArgoCD.InsecureFrom != "" {
			insecure, err = g.GetValue(c.ArgoCD.InsecureFrom)
			if err != nil {
				return nil, fmt.Errorf("unable to obtain argocd.InsecureFrom: %v", err)
			}
		}
		if insecure != "" {
			if insecure != "true" && insecure != "false" {
				return nil, fmt.Errorf("invalid argocd.InsecureFrom value: %s", insecure)
			}

			args = args.AppendStrings("--insecure", insecure)
		}
	}

	appArgs = appArgs.CopyFrom(args)
	{
		var pluginName string
		if c.ArgoCD.ConfigManagementPlugin != "" {
			pluginName = c.ArgoCD.ConfigManagementPlugin
		} else if c.Kompose != nil {
			pluginName = "kargo"
		}
		if pluginName != "" {
			appArgs = appArgs.AppendStrings("--config-management-plugin=" + pluginName)
		}

		var image string
		if c.ArgoCD.Image != "" {
			image = c.ArgoCD.Image
		} else if c.ArgoCD.ImageFrom != "" {
			image, err = g.GetValue(c.ArgoCD.ImageFrom)
			if err != nil {
				return nil, fmt.Errorf("unable to obtain argocd.ImageFrom: %v", err)
			}
		}
		if image != "" {
			appArgs = appArgs.AppendStrings("--image", image)
		}

		if path != "" {
			appArgs = appArgs.AppendStrings("--path", path)
		}

		if c.ArgoCD.Repo != "" {
			appArgs = appArgs.AppendStrings("--repo", c.ArgoCD.Repo)
		}

		destNamespace := c.ArgoCD.DestNamespace
		if destNamespace != "" {
			appArgs = appArgs.AppendStrings("--dest-namespace", destNamespace)
		}

		destServer := c.ArgoCD.DestServer
		if destServer != "" {
			appArgs = appArgs.AppendStrings("--dest-server", destServer)
		}

		if c.ArgoCD.DirRecurse {
			appArgs = appArgs.AppendStrings("--directory-recurse")
		}
	}

	if args.Len() == 0 {
		return nil, errors.New("unable to generate argocd commands: specify argocd connection-related fields in your config")
	} else if appArgs.Len() == 0 {
		return nil, errors.New("unable to generate argocd commands: specify argocd app-related fields in your config")
	} else {
		var script *Args

		script = script.Append("argocd", "proj", "create", c.ArgoCD.Project)
		script = script.Append(args)
		script = script.Append(";")
		script = script.Append("argocd", "app", "create")
		script = script.Append(appArgs)
		script = script.Append(";")
		script = script.Append("argocd", "app", "set")
		script = script.Append(appArgs)

		if g.TailLogs {
			script = script.Append(";")
			script = script.Append("argocd", "app", "logs", c.Name, "--follow", "--tail=-1")
		}

		cmds = append(cmds, Cmd{
			Name: "bash",
			Args: NewArgs("-c", NewBashScript(script)),
		})
	}

	return cmds, nil
}

func (g *Generator) cmds(c *Config, t Target) ([]Cmd, error) {
	var (
		args *Args
		err  error
	)

	if c.Compose != nil {
		args, err = AppendArgs(args, c.Compose, FieldTagCompose)
		if err != nil {
			return nil, err
		}

		dir := c.Path
		file := "docker-compose.yml"
		if strings.HasSuffix(dir, ".yml") {
			file = filepath.Base(dir)
			dir = filepath.Dir(dir)
		}

		composeArgs := NewArgs("compose", "-f", file, args)

		upArgs := NewArgs("up")
		if !g.TailLogs {
			upArgs = upArgs.Append("-d")
		}

		convArgs := NewArgs().Append(composeArgs)
		convArgs = convArgs.Append("convert")

		switch t {
		case Apply:
			if c.Compose.EnableVals {
				return []Cmd{
					{
						Name: "vals",
						Args: NewArgs("exec", "--stream-yaml", file, "--", "docker", "compose", "-f", "-", upArgs),
						Dir:  dir,
					},
				}, nil
			}

			composeUp := Cmd{
				Name: "docker",
				Args: NewArgs(composeArgs, upArgs),
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
		args, err = AppendArgs(args, c.Helm, FieldTagHelm)
		if err != nil {
			return nil, err
		}

		repo := filepath.Base(c.Helm.Repo)
		helmRepoAdd := Cmd{Name: "helm", Args: NewArgs("repo", "add", repo, c.Helm.Repo)}
		helmUpgradeArgs := NewArgs("upgrade", "--install", c.Name, repo+"/"+c.Helm.Chart, args)

		switch t {
		case Apply:
			helmUpgrade := Cmd{Name: "helm", Args: helmUpgradeArgs}
			return []Cmd{helmRepoAdd, helmUpgrade}, nil
		case Plan:
			helmDiffArgs := NewArgs("diff", helmUpgradeArgs)
			helmDiff := Cmd{Name: "helm", Args: helmDiffArgs}
			return []Cmd{helmRepoAdd, helmDiff}, nil
		}
	} else if c.Kustomize != nil {
		args, err = AppendArgs(args, c.Kustomize, FieldTagKustomize)
		if err != nil {
			return nil, err
		}

		kustomizeEdit := Cmd{
			Name: "kustomize",
			Args: NewArgs("edit", "set", "image", args),
			Dir:  c.Path,
		}

		tmpFile := filepath.Join(g.TempDir, "kustomize-built.yaml")

		kustomizeBuildArgs := NewArgs("build", "--output="+tmpFile)
		if c.Path != "" {
			kustomizeBuildArgs = kustomizeBuildArgs.Append(c.Path)
		}

		kustomizeBuild := Cmd{
			Name: "kustomize",
			Args: kustomizeBuildArgs,
		}

		kubectlArgs := NewArgs("-f", tmpFile, "--server-side=true")

		kubectlDiff := Cmd{
			Name: "kubectl",
			Args: NewArgs("diff", kubectlArgs),
		}

		kubectlApply := Cmd{
			Name: "kubectl",
			Args: NewArgs("apply", kubectlArgs),
		}

		switch t {
		case Apply:
			return []Cmd{kustomizeEdit, kustomizeBuild, kubectlApply}, nil
		case Plan:
			return []Cmd{kustomizeEdit, kustomizeBuild, kubectlDiff}, nil
		}
	} else if c.Kompose != nil {
		args, err = AppendArgs(args, c.Kompose, FieldTagKustomize)
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
			komposeConvertArgs = append(komposeConvertArgs, args.MustCollect(g.GetValue)...)
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
				args := NewArgs(
					"exec",
					"--stream-yaml",
					file,
					"--",
					"bash",
					"-c",
					strings.Join(script, " ")+tailArgs(),
				)
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
			args := NewArgs(
				"-c",
				strings.Join(script, " ")+tailArgs(),
			)
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
			args := NewArgs(
				"-c",
				strings.Join(script, " "),
			)
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
			Args: NewArgs("diff", kubectlArgs),
		}

		kubectlApply := Cmd{
			Name: "kubectl",
			Args: NewArgs("apply", kubectlArgs),
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
