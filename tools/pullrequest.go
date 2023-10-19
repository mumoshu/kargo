package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v56/github"
	"golang.org/x/oauth2"
)

const (
	CommandCreatePullRequest      = "create-pullrequest"
	FlagCreatePullRequestDir      = "dir"
	FlagCreatePullRequestTitle    = "title"
	FlagCreatePullRequestBody     = "body"
	FlagCreatePullRequestHead     = "head"
	FlagCreatePullRequestBase     = "base"
	FlagCreatePullRequestTokenEnv = "token-env"
	FlagCreatePullRequestDryRun   = "dry-run"
)

type CreatePullRequestOptions struct {
	Dir      string
	Title    string
	Body     string
	Head     string
	Base     string
	TokenEnv string
	DryRun   bool
}

// CreatePullRequest creates a pull request on GitHub.
// dir is the directory of the repository.
// title is the title of the pull request.
// body is the body of the pull request.
// head is the branch to merge from.
// base is the branch to merge to.
// token is the GitHub token.
// It returns the URL of the pull request.
func CreatePullRequest(ctx context.Context, opts CreatePullRequestOptions) (string, error) {
	dir := opts.Dir
	title := opts.Title
	body := opts.Body
	head := opts.Head
	base := opts.Base
	tokenEnv := opts.TokenEnv
	token := os.Getenv(tokenEnv)

	if token == "" {
		return "", fmt.Errorf("%s must be set", FlagCreatePullRequestTokenEnv)
	}

	if head == "" {
		return "", fmt.Errorf("head must be set")
	}

	if base == "" {
		return "", fmt.Errorf("base must be set")
	}

	if head == "main" || head == "master" {
		return "", fmt.Errorf("head must not be %s", head)
	}

	if dir == "" {
		return "", fmt.Errorf("dir must be set")
	}

	repo, err := getRepository(ctx, dir)
	if err != nil {
		return "", err
	}
	if opts.DryRun {
		fmt.Printf("dry-run: create pull request on %s/%s from %s to %s\n", repo.Owner, repo.Name, head, base)

		fmt.Printf("dry-run: showing git-diff between %s and %s\n", base, head)
		c := exec.CommandContext(ctx, "git", "diff", "--stat", "--patch-with-raw", base+".."+head)
		c.Dir = dir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return "", fmt.Errorf("running git diff: %w", err)
		}

		return "", nil
	}

	httpClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	))
	client := github.NewClient(httpClient)
	pr, _, err := client.PullRequests.Create(ctx, repo.Owner, repo.Name, &github.NewPullRequest{
		Title: &title,
		Body:  &body,
		Head:  &head,
		Base:  &base,
	})
	if err != nil {
		return "", fmt.Errorf("calling pull request creation API: %w", err)
	}
	return pr.GetHTMLURL(), nil
}

type Repository struct {
	Owner string
	Name  string
}

// GetRepository returns the GitHub repository of the remote of the
// local repository in dir.
func getRepository(ctx context.Context, dir string) (*Repository, error) {
	c := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	c.Dir = dir
	out, err := c.Output()
	if err != nil {
		return nil, err
	}
	return parseRepository(out)
}

// parseRepository parses the GitHub repository from the output of
// git remote get-url.
func parseRepository(gitRemoteGetURLOut []byte) (*Repository, error) {
	s := strings.Split(string(gitRemoteGetURLOut), ":")
	if len(s) != 2 {
		return nil, fmt.Errorf("expected 2 parts but got %d", len(s))
	}

	s = strings.Split(strings.TrimSuffix(s[1], "\n"), "/")
	if len(s) != 2 {
		return nil, fmt.Errorf("expected 2 parts but got %d", len(s))
	}

	return &Repository{
		Owner: s[0],
		Name:  strings.TrimSuffix(s[1], ".git"),
	}, nil
}
