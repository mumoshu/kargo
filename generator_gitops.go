package kargo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mumoshu/kargo/tools"
)

// gitOps generates a series of commands to:
// - git-clone the repo,
// - copy files from the local filesystem to the worktree,
// - modify files in the worktree,
// - git-add the modified files,
// - git-commit the changes,
// - and git-push the changes.
// The commands are generated in such a way that they can be
// used to plan or apply the deployment in a gitops environment.
func (g *Generator) gitOps(t Target, name, repo, branch, path string, copies []Upload, fileModCmds []Cmd, doPR bool) ([]Cmd, error) {
	if t == Apply && len(g.ToolsCommand) == 0 {
		return nil, errors.New("ToolsCommand is required to run kargo tools")
	}

	if g.TempDir == "" {
		return nil, errors.New("TempDir is required to use GitOps support")
	}

	const (
		remoteName = "origin"
	)

	var cmds []Cmd

	localRepoDir := filepath.Join(g.TempDir, "kargo-gitops", name)

	var script *Args

	gitCloneArgs := NewArgs("clone", repo, localRepoDir)
	script = script.Append("git", gitCloneArgs, ";")
	// gitClone := Cmd{Name: "git", Args: gitCloneArgs}

	formatDateTime := func(t time.Time) string {
		return t.Format("20060102150405")
	}
	datetime := formatDateTime(time.Now())
	if g.ToolName == "" {
		return nil, errors.New("ToolName is required to use GitOps support")
	}
	branchName := g.ToolName + "-" + datetime

	baseBranch := "main"
	if branch != "" {
		baseBranch = branch
	}
	script = script.Append("(", "cd", localRepoDir, "&&", "git", "fetch", remoteName, "&&", "git", "stash", "&&", "git", "checkout", "-b", branchName, remoteName+"/"+baseBranch, "&&", "git", "rebase", remoteName+"/"+baseBranch, ")")

	runGitCheckoutScript := Cmd{
		Name: "bash",
		Args: NewArgs("-vxc", NewBashScript(script)),
	}

	var (
		copyLocal  *Args
		copyRemote *Args
	)

	var fileCopies []Cmd

	for _, u := range copies {
		if p := u.Local; p != "" {
			copyLocal = copyLocal.AppendStrings(p)
		}

		if p := u.Remote; p != "" {
			copyRemote = copyRemote.AppendStrings(p)
		}

		cpArgs := NewArgs("cp", "-r", NewJoin(NewArgs(copyLocal, string(os.PathSeparator), "*")), NewJoin(NewArgs(localRepoDir, string(os.PathSeparator), copyRemote)))
		cp := Cmd{Name: "bash", Args: NewArgs("-vxc", NewBashScript(cpArgs))}

		fileCopies = append(fileCopies, cp)
	}

	fileModScript := g.scriptWithinDir(filepath.Join(localRepoDir, path), fileModCmds)

	runFileModScript := Cmd{Name: "bash", Args: NewArgs("-vxc", NewBashScript(fileModScript))}

	var gitAddArgs *Args
	gitAddArgs = gitAddArgs.Append("cd", localRepoDir, ";")
	gitAddArgs = gitAddArgs.Append("git", "add", ".")
	runGitAddScript := Cmd{Name: "bash", Args: NewArgs("-vxc", NewBashScript(gitAddArgs))}

	var gitCommitArgs *Args
	gitCommitArgs = gitCommitArgs.Append(
		"cd", localRepoDir, ";",
	)
	gitCommitArgs = gitCommitArgs.Append(
		"git", "commit", "-m", "'automated commit'",
	)
	gitCommit := Cmd{Name: "bash", Args: NewArgs("-vxc", NewBashScript(gitCommitArgs))}

	var gitPushArgs *Args
	gitPushArgs = gitPushArgs.Append(
		"cd", localRepoDir, ";",
	)
	gitPushArgs = gitPushArgs.Append(
		"git", "push", remoteName, branchName,
	)
	gitPush := Cmd{Name: "bash", Args: NewArgs("-vxc", NewBashScript(gitPushArgs))}

	// var gitDiffArgs *Args
	// gitDiffArgs = gitDiffArgs.Append(
	// 	"cd", localRepoDir, ";",
	// )
	// gitDiffArgs = gitDiffArgs.Append(
	// 	"git", "diff",
	// )
	// gitDiff := Cmd{Name: "bash", Args: NewArgs("-vxc", NewBashScript(gitDiffArgs))}

	cmds = append(cmds, runGitCheckoutScript)
	cmds = append(cmds, fileCopies...)
	cmds = append(cmds, runFileModScript)
	cmds = append(cmds, runGitAddScript)
	cmds = append(cmds, gitCommit)

	if !doPR {
		return cmds, nil
	}

	tokenEnv := "KARGO_TOOLS_GITHUB_TOKEN"
	var toolArgs []string
	toolArgs = append(toolArgs, g.ToolsCommand[1:]...)
	toolArgs = append(toolArgs, tools.CommandCreatePullRequest,
		"--"+tools.FlagCreatePullRequestDir, localRepoDir,
		"--"+tools.FlagCreatePullRequestTitle, "Deploy "+name,
		"--"+tools.FlagCreatePullRequestBody, "Deploy "+name,
		"--"+tools.FlagCreatePullRequestHead, branchName,
		"--"+tools.FlagCreatePullRequestBase, baseBranch,
		"--"+tools.FlagCreatePullRequestTokenEnv, tokenEnv,
	)
	if os.Getenv("KANVAS_DRY_RUN") == "true" || t == Plan {
		toolArgs = append(toolArgs, "--"+tools.FlagCreatePullRequestDryRun, "true")
	}
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return nil, fmt.Errorf("unable to generate gitops commands: %s is required", "GITHUB_TOKEN")
	}
	kargoToolsCreatePullRequest := Cmd{
		Name:   g.ToolsCommand[0],
		Args:   NewArgs(toolArgs),
		AddEnv: map[string]string{tokenEnv: githubToken},
	}
	cmds = append(cmds, gitPush, kargoToolsCreatePullRequest)

	return cmds, nil
}
