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
	// FieldTagArgoCD is used to defining argocd-app-create flags
	FieldTagArgoCD = "argocd"
)

type Config struct {
	// Name is the application name.
	// It defaults to the basename of the path if
	// kargo is run as a command.
	Name      string     `yaml:"name" argocd:",arg"`
	Path      string     `yaml:"path" argocd:""`
	Env       []Env      `yaml:"env" argocd:"plugin-env"`
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

func (e Env) FlagValue(get GetValue) (string, error) {
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
	Images KustomizeImages `yaml:"images"`
}

type KustomizeImages []KustomizeImage

func (i KustomizeImages) AppendArgs(args []string, get GetValue, key string) (*[]string, error) {
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
	Chart   string `yaml:"chart" helm:"" argocd:"helm-chart"`
	Version string `yaml:"version" argocd:"revision"`
	Set     []Set  `yaml:"set" helm:"set" argocd:"helm-set"`
}

func (s Set) FlagValue(get GetValue) (string, error) {
	return fmt.Sprintf("%s=%s", s.Name, s.Value), nil
}

type ArgoCD struct {
	Repo       string `yaml:"repo"`
	Path       string `yaml:"path"`
	DirRecurse bool   `yaml:"dirRecurse" argocd:"directory-recurse"`
	Namespace  string `yaml:"namespace" argocd:"dest-namespace"`
	ServerFrom string `yaml:"serverFrom" argocd:"server"`
	Project    string `yaml:"project" argocd:"project"`
	// Push is set to true if the user wants kargo to automatically
	// - git-clone the repo
	// - git-add the files in the config.Path
	// - git-commit
	// - git-push
	// so that it triggers the deployment.
	Push bool `yaml:"push" argocd:""`
	// Image is the image to be used for the deployment.
	Image string `yaml:"image" kargo:""`
	// ImageFrom is the key to be used to get the image from the environment.
	ImageFrom string `yaml:"imageFrom" kargo:""`
	// ConfigManagementPlugin is the config management plugin to be used.
	ConfigManagementPlugin string `yaml:"configManagementPlugin" argocd:"config-management-plugin"`
}

type GetValue func(key string) (string, error)
