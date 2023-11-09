package kargo

import (
	"fmt"
)

const (
	// FieldTagKargo is used to defining kargo flags.
	// Currently, this is used to exclude the field from the field-to-flag conversion.
	// Example: Name `yaml:"name" kargo:""`
	FieldTagKargo = "kargo"

	// FieldTagArgoCD is used to defining helm-upgrade flags
	FieldTagCompose   = "compose"
	FieldTagHelm      = "helm"
	FieldTagKustomize = "kustomize"
	FieldTagKompose   = "kompose"
	// FieldTagArgoCDApp is used to defining argocd-app-create flags
	FieldTagArgoCDApp = "argocd-app"
)

type Config struct {
	// Name is the application name.
	// It defaults to the basename of the path if
	// kargo is run as a command.
	Name      string     `yaml:"name" argocd-app:",arg"`
	Path      string     `yaml:"path" kargo:""`
	PathFrom  string     `yaml:"pathFrom" kargo:""`
	Env       []Env      `yaml:"env" argocd-app:"plugin-env"`
	Compose   *Compose   `yaml:"compose"`
	Kompose   *Kompose   `yaml:"kompose"`
	Kustomize *Kustomize `yaml:"kustomize"`
	Helm      *Helm      `yaml:"helm"`
	ArgoCD    *ArgoCD    `yaml:"argocd"`
}

type Env struct {
	Name      string `yaml:"name"`
	Value     string `yaml:"value"`
	ValueFrom string `yaml:"valueFrom"`
}

func (e Env) KargoValue(get GetValue) (string, error) {
	v := e.Value
	if e.ValueFrom != "" {
		var err error
		v, err = get(e.ValueFrom)
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s=%s", e.Name, v), nil
}

type Compose struct {
	EnableVals bool `yaml:"enableVals" kargo:""`
}

type Kompose struct {
	EnableVals bool `yaml:"enableVals" kargo:""`
}

const (
	KustomizeStrategyBuildAndKubectlApply = "BuildAndKubectlApply"
	KustomizeStrategySetImageAndCreatePR  = "SetImageAndCreatePullRequest"
)

type Kustomize struct {
	// Strategy is the strategy to be used for the deployment.
	//
	// The supported values are:
	// - BuildAndKubectlApply
	// - SetImageAndCreatePullRequest
	//
	// BuildAndKubectlApply is the default strategy.
	// It runs kustomize build and kubectl apply to deploy the application.
	//
	// SetImageAndCreatePullRequest runs kustomize edit set image and creates a pull request.
	// It's useful to trigger a deployment workflow in CI/CD.
	Strategy string          `yaml:"strategy" kargo:""`
	Images   KustomizeImages `yaml:"images" argocd-app:"kustomize-image"`
	Git      KustomizeGit    `yaml:"git" kargo:""`
}

type KustomizeGit struct {
	Repo   string `yaml:"repo" kargo:""`
	Branch string `yaml:"branch" kargo:""`
	Path   string `yaml:"path" kargo:""`
}

type KustomizeImages []KustomizeImage

func (i KustomizeImages) KargoAppendArgs(args *Args, key string) (*Args, error) {
	var images *Args
	for _, img := range i {
		var s *Args
		s = s.AppendStrings(img.Name)
		if img.NewName != "" {
			s = s.AppendStrings("=" + img.NewName)
		}
		if img.NewTag != "" {
			s = s.AppendStrings(":")
			s = s.AppendStrings(img.NewTag)
		} else if img.NewDigestFrom != "" {
			s = s.AppendStrings("@")
			s = s.AppendValueFromOutput(img.NewDigestFrom)
		} else if img.NewTagFrom != "" {
			s = s.AppendStrings(":")
			s = s.AppendValueFromOutput(img.NewTagFrom)
		} else {
			return nil, fmt.Errorf("either newTag or newDigestFrom must be set")
		}
		images = images.Append(NewJoin(s))
	}

	if key == "argocd" {
		args = args.Append(args, "--kustomize-image")
	}
	args = args.Append(args, images)

	return args, nil
}

var _ KargoArgsAppender = KustomizeImages{}

type KustomizeImage struct {
	Name          string `yaml:"name"`
	NewName       string `yaml:"newName"`
	NewTag        string `yaml:"newTag"`
	NewTagFrom    string `yaml:"newTagFrom"`
	NewDigestFrom string `yaml:"newDigestFrom"`
}

type Helm struct {
	Repo        string   `yaml:"repo" helm:""`
	Chart       string   `yaml:"chart" helm:"" argocd-app:"helm-chart"`
	Version     string   `yaml:"version" argocd-app:"revision"`
	Set         []Set    `yaml:"set" helm:"set" argocd-app:"helm-set"`
	ValuesFiles []string `yaml:"valuesFiles" helm:"values" argocd-app:"values"`
}

func (s Set) KargoValue(get GetValue) (string, error) {
	return fmt.Sprintf("%s=%s", s.Name, s.Value), nil
}

type ArgoCD struct {
	Repo string `yaml:"repo" kargo:""`
	// Branch is the branch to be used for the deployment.
	// This isn't part of the arguments for argocd-repo-add because
	// it doesn't support branch.
	// However, we use it when you want to push manifests to a branch
	// and trigger a deployment.
	Branch   string `yaml:"branch" kargo:""`
	RepoFrom string `yaml:"repoFrom" kargo:""`

	RepoSSHPrivateKeyPath     string `yaml:"repoSSHPrivateKeyPath" kargo:""`
	RepoSSHPrivateKeyPathFrom string `yaml:"repoSSHPrivateKeyPathFrom" kargo:""`

	Path     string `yaml:"path" kargo:""`
	PathFrom string `yaml:"pathFrom" kargo:""`

	Upload []Upload `yaml:"upload" kargo:""`

	DirRecurse bool `yaml:"dirRecurse" argocd-app:"directory-recurse,paramless"`

	// Server is the ArgoCD server to be used for the deployment.
	Server string `yaml:"server" kargo:""`
	// ServerFrom is the key to be used to get the ArgoCD server from the environment.
	ServerFrom string `yaml:"serverFrom" kargo:""`
	// Username is the username to be used for the deployment.
	Username string `yaml:"username" kargo:""`
	// UsernameFrom is the key to be used to get the username from the environment.
	UsernameFrom string `yaml:"usernameFrom" kargo:""`
	// Password is the password to be used for the deployment.
	Password string `yaml:"password" kargo:""`
	// PasswordFrom is the key to be used to get the password from the environment.
	PasswordFrom string `yaml:"passwordFrom" kargo:""`
	// Insecure is set to true if the user wants to skip TLS verification.
	Insecure bool `yaml:"insecure" kargo:""`
	// InsecureFrom is the key to be used to get the insecure flag from the environment.
	InsecureFrom string `yaml:"insecureFrom" kargo:""`

	// Project is the ArgoCD project to be used for the deployment.
	Project string `yaml:"project" argocd-app:"project"`
	// Push is set to true if the user wants kargo to automatically
	// - git-clone the repo
	// - git-add the files in the config.Path
	// - git-commit
	// - git-push
	// so that it triggers the deployment.
	Push bool `yaml:"push" kargo:""`

	// DestName is the name of the K8s cluster where the deployment is to be done.
	DestName string `yaml:"name" kargo:""`
	// DestNameFrom is the key to be used to get the target K8s cluster name from the environment.
	DestNameFrom string `yaml:"nameFrom" kargo:""`

	// DestNamespace is the namespace to be used for the deployment.
	DestNamespace string `yaml:"namespace" kargo:""`
	// DestServer is the Kubernetes API endpoint of the cluster where the deployment is to be done.
	DestServer string `yaml:"destServer" kargo:""`
	// DestServerFrom is the key to be used to get the target Kubernetes API endpoint from the environment.
	DestServerFrom string `yaml:"destServerFrom" kargo:""`
	// ConfigManagementPlugin is the config management plugin to be used.
	ConfigManagementPlugin string `yaml:"configManagementPlugin" argocd-app:"config-management-plugin"`
}

type Upload struct {
	Local  string `yaml:"local" kargo:""`
	Remote string `yaml:"remote" kargo:""`
}

type GetValue func(key string) (string, error)
