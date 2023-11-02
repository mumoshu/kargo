package kargo

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
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

	// ToolsCommand is the command to run kargo tools.
	//
	// If you set this to e.g. `mycmd tools`, kargo will run
	// `mycmd tools <tool> <args...>` if needed.
	//
	// An example of a tool is `create-pullrequest`, whose command becomes
	// `mycmd tools create-pullrequest <args...>`.
	//
	// This needs to be set if you want to use kargo tools.
	// If this is not set and kargo required to run a tool,
	// kargo will return an error.
	ToolsCommand []string
}

type Target int

const (
	Plan = iota
	Apply
)

type Cmd struct {
	ID   string
	Name string
	Args *Args
	Dir  string
	// AddEnv is a map of environment variables to add to the command.
	// That is, the command will be run with the environment variables
	// specified in AddEnv in addition to the environment variables provided
	// by the current process(os.Environ).
	AddEnv map[string]string
}

func (c Cmd) ToArgs() *Args {
	return NewArgs(c.Name, c.Args)
}

func (c *Cmd) String() string {
	return fmt.Sprintf("%s %s", c.Name, c.Args)
}

func (g *Generator) ExecCmds(c *Config, t Target) ([]Cmd, error) {
	if c.ArgoCD != nil {
		return g.cmdsArgoCD(c, t)
	}

	return g.cmds(c, t)
}

func (g *Generator) cmdsArgoCD(c *Config, t Target) ([]Cmd, error) {
	var (
		args                       *Args
		loginArgs                  *Args
		appArgs                    *Args
		awsEKSUpdateKubeconfigArgs *Args
		clusterAddArgs             *Args
		repoAddArgs                *Args
		err                        error
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

	var (
		remotePath *Args
		cmds       []Cmd
	)

	if c.ArgoCD.Path != "" {
		remotePath = remotePath.AppendStrings(c.ArgoCD.Path)
	} else if c.ArgoCD.PathFrom != "" {
		remotePath = remotePath.AppendValueFromOutput(c.ArgoCD.PathFrom)
	}

	{
		if c.ArgoCD.Server != "" {
			args = args.AppendStrings("--server", c.ArgoCD.Server)

			loginArgs = loginArgs.AppendStrings(c.ArgoCD.Server)
		} else if c.ArgoCD.ServerFrom != "" {
			args = args.AppendStrings("--server")
			args = args.AppendValueFromOutput(c.ArgoCD.ServerFrom)

			loginArgs = loginArgs.AppendValueFromOutput(c.ArgoCD.ServerFrom)
		}
	}

	{
		if c.ArgoCD.Username != "" {
			loginArgs = loginArgs.AppendStrings("--username", c.ArgoCD.Username)
		} else if c.ArgoCD.UsernameFrom != "" {
			loginArgs = loginArgs.AppendStrings("--username")
			loginArgs = loginArgs.AppendValueFromOutput(c.ArgoCD.UsernameFrom)
		}
	}

	{
		if c.ArgoCD.Password != "" {
			loginArgs = loginArgs.AppendStrings("--password", c.ArgoCD.Password)
		} else if c.ArgoCD.PasswordFrom != "" {
			loginArgs = loginArgs.AppendStrings("--password")
			loginArgs = loginArgs.AppendValueFromOutput(c.ArgoCD.PasswordFrom)
		}
	}

	{
		if c.ArgoCD.Insecure {
			args = args.AppendStrings("--insecure")

			loginArgs = loginArgs.AppendStrings("--insecure")
		} else if c.ArgoCD.InsecureFrom != "" {
			args = args.AppendValueIfOutput("--insecure", c.ArgoCD.InsecureFrom)

			loginArgs = loginArgs.AppendValueIfOutput("--insecure", c.ArgoCD.InsecureFrom)
		}
	}

	appArgs = appArgs.CopyFrom(args)
	{
		if c.ArgoCD.DestName != "" {
			appArgs = appArgs.AppendStrings("--dest-name", c.ArgoCD.DestName)

			awsEKSUpdateKubeconfigArgs = awsEKSUpdateKubeconfigArgs.AppendStrings("--name", c.ArgoCD.DestName)
			awsEKSUpdateKubeconfigArgs = awsEKSUpdateKubeconfigArgs.AppendStrings("--alias", c.ArgoCD.DestName)

			clusterAddArgs = clusterAddArgs.AppendStrings(c.ArgoCD.DestName)
		} else if c.ArgoCD.DestNameFrom != "" {
			appArgs = appArgs.AppendStrings("--dest-name")
			appArgs = appArgs.AppendValueFromOutput(c.ArgoCD.DestNameFrom)

			awsEKSUpdateKubeconfigArgs = awsEKSUpdateKubeconfigArgs.AppendStrings("--name")
			awsEKSUpdateKubeconfigArgs = awsEKSUpdateKubeconfigArgs.AppendValueFromOutput(c.ArgoCD.DestNameFrom)
			awsEKSUpdateKubeconfigArgs = awsEKSUpdateKubeconfigArgs.AppendStrings("--alias")
			awsEKSUpdateKubeconfigArgs = awsEKSUpdateKubeconfigArgs.AppendValueFromOutput(c.ArgoCD.DestNameFrom)

			clusterAddArgs = clusterAddArgs.AppendValueFromOutput(c.ArgoCD.DestNameFrom)
		} else {
			return nil, errors.New("unable to generate argocd commands: specify argocd.DestName or argocd.DestNameFrom in your config")
		}

		var pluginName string
		if c.ArgoCD.ConfigManagementPlugin != "" {
			pluginName = c.ArgoCD.ConfigManagementPlugin
		} else if c.Kompose != nil {
			pluginName = "kargo"
		}
		if pluginName != "" {
			appArgs = appArgs.AppendStrings("--config-management-plugin=" + pluginName)
		}

		// TODO Remote path is required for ArgoCD App with Repo
		appArgs = appArgs.AppendStrings("--path")
		if remotePath == nil {
			return nil, errors.New("unable to generate argocd commands: specify argocd.Path or argocd.PathFrom in your config")
		}
		appArgs = appArgs.Append(remotePath)

		if c.ArgoCD.Repo != "" {
			appArgs = appArgs.AppendStrings("--repo", c.ArgoCD.Repo)

			repoAddArgs = repoAddArgs.AppendStrings(c.ArgoCD.Repo)
		} else if c.ArgoCD.RepoFrom != "" {
			appArgs = appArgs.AppendStrings("--repo")
			appArgs = appArgs.AppendValueFromOutput(c.ArgoCD.RepoFrom)

			repoAddArgs = repoAddArgs.AppendStrings("--repo")
			repoAddArgs = repoAddArgs.AppendValueFromOutput(c.ArgoCD.RepoFrom)
		} else {
			return nil, errors.New("unable to generate argocd commands: specify argocd.repo or argocd.repoFrom in your config")
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

	var proj string
	if c.ArgoCD.Project != "" {
		proj = c.ArgoCD.Project
	} else {
		proj = c.Name
	}

	var regex = regexp.MustCompile(`[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*`)
	if !regex.MatchString(proj) {
		return nil, fmt.Errorf("invalid argocd.Project value: %s", proj)
	}

	if c.ArgoCD.RepoSSHPrivateKeyPath != "" {
		repoAddArgs = repoAddArgs.AppendStrings("--ssh-private-key-path", c.ArgoCD.RepoSSHPrivateKeyPath)
	} else if c.ArgoCD.RepoSSHPrivateKeyPathFrom != "" {
		repoAddArgs = repoAddArgs.AppendStrings("--ssh-private-key-path")
		repoAddArgs = repoAddArgs.AppendValueFromOutput(c.ArgoCD.RepoSSHPrivateKeyPathFrom)
	}

	if args.Len() == 0 {
		return nil, errors.New("unable to generate argocd commands: specify argocd connection-related fields in your config")
	} else if appArgs.Len() == 0 {
		return nil, errors.New("unable to generate argocd commands: specify argocd app-related fields in your config")
	}

	push := c.ArgoCD.Push
	if len(c.ArgoCD.Upload) > 0 {
		push = true
	}

	if push {
		g, err := g.gitOps(t, c.Name, c.ArgoCD.Repo, c.ArgoCD.Upload, nil)
		if err != nil {
			return nil, fmt.Errorf("uanble to generate gitops commands: %w", err)
		}
		cmds = append(cmds, g...)
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

	var script *Args

	script = script.Append("argocd", "login")
	script = script.Append(loginArgs)
	script = script.Append(";")
	script = script.Append("argocd", "proj", "create", proj)
	script = script.Append(args)
	script = script.Append(";")
	script = script.Append("aws", "eks", "update-kubeconfig")
	script = script.Append(awsEKSUpdateKubeconfigArgs)
	script = script.Append(";")
	script = script.Append("argocd", "cluster", "add")
	script = script.Append(clusterAddArgs)
	script = script.Append(";")
	script = script.Append("argocd", "repo", "add")
	script = script.Append(repoAddArgs)
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
		Args: NewArgs("-vxc", NewBashScript(script)),
	})

	return cmds, nil
}

func (g *Generator) runWithinDir(dir string, cmds []Cmd) Cmd {
	script := g.scriptWithinDir(dir, cmds)

	runScript := Cmd{Name: "bash", Args: NewArgs("-vxc", NewBashScript(script))}

	return runScript
}

func (g *Generator) scriptWithinDir(dir string, cmds []Cmd) *Args {
	var script *Args
	script = script.Append("cd", dir, ";")
	for i, cmd := range cmds {
		script = script.Append(cmd.Name, cmd.Args)
		if i < len(cmds)-1 {
			script = script.Append("&&")
		}
	}
	return script
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

		var (
			chart string
			cmds  []Cmd
		)

		if c.Helm.Repo != "" {
			repo := filepath.Base(c.Helm.Repo)
			helmRepoAdd := Cmd{Name: "helm", Args: NewArgs("repo", "add", repo, c.Helm.Repo)}
			cmds = append(cmds, helmRepoAdd)
			// We treat Helm.Chart as a remote chart name
			// which means we need to add a repo name as prefix
			// resulting in a helm command like:
			// helm upgrade --install <name> <repo>/<chart>
			chart = repo + "/" + c.Helm.Chart
		} else {
			// We treat Helm.Chart as a path to a local chart
			// which means we don't need to add a repo name as prefix
			// resulting in a helm command like:
			// helm upgrade --install <name> <path>
			chart = c.Helm.Chart
		}

		if chart == "" {
			// If we have Path set and Chart unset, we treat Path as a path to a local chart
			chart = "."
		}

		// Note that helm-diff-upgrate flags are superset of helm-upgrade flags
		helmUpgradeArgs := NewArgs("upgrade", "--install", c.Name, chart, args)

		switch t {
		case Apply:
			helmUpgrade := Cmd{
				Name: "helm",
				Args: helmUpgradeArgs,
				Dir:  c.Path,
			}
			cmds = append(cmds, helmUpgrade)
			return cmds, nil
		case Plan:
			helmDiffArgs := NewArgs("diff", helmUpgradeArgs)
			helmDiff := Cmd{
				Name: "helm",
				Args: helmDiffArgs,
				Dir:  c.Path,
			}
			cmds = append(cmds, helmDiff)
			return cmds, nil
		}
	} else if c.Kustomize != nil {
		if c.Kustomize.Images != nil {
			args, err = AppendArgs(args, c.Kustomize.Images, FieldTagKustomize)
			if err != nil {
				return nil, err
			}
		}

		if args.Len() == 0 {
			return nil, fmt.Errorf("unable to generate kustomize commands: specify kubernetes.kustomize.images fields in your config")
		}

		kustomizeEdit := Cmd{
			Name: "kustomize",
			Args: NewArgs("edit", "set", "image", args),
			Dir:  c.Path,
		}

		if g.TempDir == "" {
			return nil, fmt.Errorf("TempDir is required to run kustomize")
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

		if c.Kustomize.Strategy == KustomizeStrategySetImageAndCreatePR {
			if c.Kustomize.Git.Repo == "" {
				return nil, fmt.Errorf("kustomize.git.repo is required for kustomize.strategy=%s", KustomizeStrategySetImageAndCreatePR)
			}
			setImageAndCreatePR, err := g.gitOps(t, c.Name, c.Kustomize.Git.Repo, nil, []Cmd{kustomizeEdit})
			if err != nil {
				return nil, fmt.Errorf("uanble to generate gitops commands: %w", err)
			}
			return setImageAndCreatePR, nil
		} else if c.Kustomize.Strategy == KustomizeStrategyBuildAndKubectlApply || c.Kustomize.Strategy == "" {
			switch t {
			case Apply:
				return []Cmd{kustomizeEdit, kustomizeBuild, kubectlApply}, nil
			case Plan:
				return []Cmd{kustomizeEdit, kustomizeBuild, kubectlDiff}, nil
			default:
				return nil, fmt.Errorf("unsupported target: %v", t)
			}
		} else {
			return nil, fmt.Errorf("unsupported kustomize strategy: %s", c.Kustomize.Strategy)
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
