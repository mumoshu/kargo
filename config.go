package kargo

import (
	"fmt"
	"strings"
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
	Path      string     `yaml:"path" argocd-app:""`
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

type Kustomize struct {
	Images KustomizeImages `yaml:"images" argocd-app:"kustomize-image"`
}

type KustomizeImages []KustomizeImage

func (i KustomizeImages) KargoAppendArgs(args []string, key string) (*[]string, error) {
	var images []string
	for _, img := range i {
		var s string
		if img.NewName == "" {
			s = img.Name + ":" + img.NewTag
		} else {
			s = img.Name + "=" + img.NewName + ":" + img.NewTag
		}
		images = append(images, s)
	}

	if key == "argocd" {
		args = append(args, "--kustomize-image")
		args = append(args, strings.Join(images, ","))
	} else {
		args = append(args, images...)
	}

	return &args, nil
}

type KustomizeImage struct {
	Name    string `yaml:"name"`
	NewName string `yaml:"newName"`
	NewTag  string `yaml:"newTag"`
}

type Helm struct {
	Repo    string `yaml:"repo" helm:""`
	Chart   string `yaml:"chart" helm:"" argocd-app:"helm-chart"`
	Version string `yaml:"version" argocd-app:"revision"`
	Set     []Set  `yaml:"set" helm:"set" argocd-app:"helm-set"`
}

func (s Set) KargoValue(get GetValue) (string, error) {
	return fmt.Sprintf("%s=%s", s.Name, s.Value), nil
}

type ArgoCD struct {
	Repo       string `yaml:"repo"`
	Path       string `yaml:"path"`
	DirRecurse bool   `yaml:"dirRecurse" argocd-app:"directory-recurse"`

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

	// DestNamespace is the namespace to be used for the deployment.
	DestNamespace string `yaml:"namespace" kargo:""`
	// DestServer is the Kubernetes API endpoint of the cluster where the deployment is to be done.
	DestServer string `yaml:"destServer" kargo:""`
	// DestServerFrom is the key to be used to get the target Kubernetes API endpoint from the environment.
	DestServerFrom string `yaml:"destServerFrom" kargo:""`
	// Image is the image to be used for the deployment.
	Image string `yaml:"image" kargo:""`
	// ImageFrom is the key to be used to get the image from the environment.
	ImageFrom string `yaml:"imageFrom" kargo:""`
	// ConfigManagementPlugin is the config management plugin to be used.
	ConfigManagementPlugin string `yaml:"configManagementPlugin" argocd-app:"config-management-plugin"`
}

type GetValue func(key string) (string, error)
