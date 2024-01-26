package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v56/github"
	"golang.org/x/oauth2"
)

const (
	CommandCreatePullRequest         = "create-pullrequest"
	FlagCreatePullRequestDir         = "dir"
	FlagCreatePullRequestTitle       = "title"
	FlagCreatePullRequestBody        = "body"
	FlagCreatePullRequestHead        = "head"
	FlagCreatePullRequestBase        = "base"
	FlagCreatePullRequestAssigneeIDs = "assignee-ids"
	FlagCreatePullRequestTokenEnv    = "token-env"
	FlagCreatePullRequestDryRun      = "dry-run"
	FlagCreatePullRequestOutputFile  = "output-file"
)

type CreatePullRequestOptions struct {
	Dir   string
	Title string
	Body  string
	Head  string
	Base  string
	// OutputFile is the path to the file to write the pull request info to.
	OutputFile string
	// AssigneeIDs is the list of GitHub user IDs to assign to the pull request.
	// Each ID can be either an integer or a string.
	AssigneeIDs []string
	TokenEnv    string
	DryRun      bool
}

// PullRequest is a pull request on GitHub that
// is created by kargo / CreatePullRequest function.
type PullRequest struct {
	ID      int64  `json:"id" yaml:"id"`
	NodeID  string `json:"nodeID" yaml:"nodeID"`
	Number  int    `json:"number" yaml:"number"`
	Head    string `json:"head" yaml:"head"`
	HTMLURL string `json:"htmlURL" yaml:"htmlURL"`
}

// CreatePullRequest creates a pull request on GitHub.
// dir is the directory of the repository.
// title is the title of the pull request.
// body is the body of the pull request.
// head is the branch to merge from.
// base is the branch to merge to.
// token is the GitHub token.
// It returns the PullRequest object on success.
func CreatePullRequest(ctx context.Context, opts CreatePullRequestOptions) (*PullRequest, error) {
	dir := opts.Dir
	title := opts.Title
	body := opts.Body
	head := opts.Head
	base := opts.Base
	tokenEnv := opts.TokenEnv
	token := os.Getenv(tokenEnv)

	if token == "" {
		return nil, fmt.Errorf("%s must be set", FlagCreatePullRequestTokenEnv)
	}

	if head == "" {
		return nil, fmt.Errorf("head must be set")
	}

	if base == "" {
		return nil, fmt.Errorf("base must be set")
	}

	if head == "main" || head == "master" {
		return nil, fmt.Errorf("head must not be %s", head)
	}

	if dir == "" {
		return nil, fmt.Errorf("dir must be set")
	}

	repo, err := getRepository(ctx, dir)
	if err != nil {
		return nil, err
	}
	if opts.DryRun {
		fmt.Printf("dry-run: create pull request on %s/%s from %s to %s\n", repo.Owner, repo.Name, head, base)

		fmt.Printf("dry-run: showing git-diff between %s and %s\n", base, head)
		c := exec.CommandContext(ctx, "git", "diff", "--stat", "--patch-with-raw", base+".."+head)
		c.Dir = dir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return nil, fmt.Errorf("running git diff: %w", err)
		}

		return nil, nil
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
		return nil, fmt.Errorf("calling pull request creation API: %w", err)
	}

	if pr == nil {
		return nil, fmt.Errorf("assertion error: pull request is nil: %v", opts)
	}

	if len(opts.AssigneeIDs) > 0 {
		_, _, err = client.Issues.AddAssignees(ctx, repo.Owner, repo.Name, pr.GetNumber(), opts.AssigneeIDs)
		if err != nil {
			return nil, fmt.Errorf("calling add assignees API: %w", err)
		}
	}

	r := &PullRequest{
		ID:      pr.GetID(),
		NodeID:  pr.GetNodeID(),
		Number:  pr.GetNumber(),
		HTMLURL: pr.GetHTMLURL(),
	}
	if h := pr.GetHead(); h != nil {
		r.Head = h.GetRef()
	}

	if opts.OutputFile != "" {
		f, err := os.Create(opts.OutputFile)
		if err != nil {
			return nil, fmt.Errorf("creating output file: %w", err)
		}
		defer f.Close()

		if err := json.NewEncoder(f).Encode(r); err != nil {
			return nil, fmt.Errorf("writing output file: %w", err)
		}

		if err := f.Close(); err != nil {
			return nil, fmt.Errorf("closing output file: %w", err)
		}
	}

	return r, nil
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
	o := strings.TrimSpace(string(gitRemoteGetURLOut))

	u, err := url.Parse(o)
	if err != nil {
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

	if u.Scheme != "https" {
		return nil, fmt.Errorf("expected https scheme but got %s", u.Scheme)
	}

	p := strings.TrimLeft(u.Path, "/")

	s := strings.Split(p, "/")
	if len(s) != 2 {
		return nil, fmt.Errorf("expected 2 parts in url path but got %d: %s", len(s), p)
	}

	return &Repository{
		Owner: s[0],
		Name:  strings.TrimSuffix(s[1], ".git"),
	}, nil
}
