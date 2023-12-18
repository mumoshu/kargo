package kargo

import (
	"os"
	"strings"
)

// PullRequestOptions is the options for creating a pull request.
// It reads the options from environment variables.
// The environment variables' names are prefixed with the tool name.
//
// For example, if the tool name is "mytool", the environment variables
// are prefixed like "MYTOOL_PULLREQUEST_ASSIGNEE_IDS".
//
// The supported environment variables are:
// - <tool name>_PULLREQUEST_ASSIGNEE_IDS
//
// The value of <tool name>_PULLREQUEST_ASSIGNEE_IDS is a comma-separated list of GitHub user IDs.
// Each ID can be either an integer or a string.
func (g *Generator) prOptsFromEnv() PullRequestOptions {
	var opts PullRequestOptions
	env := strings.ToUpper(g.ToolName) + "_PULLREQUEST_ASSIGNEE_IDS"
	if v := os.Getenv(env); v != "" {
		opts.AssigneeIDs = strings.Split(v, ",")
	}

	if opts.OutputFile == "" {
		opts.OutputFile = g.PullRequestOutputFile
	}

	return opts
}

func (g *Generator) prHeadFromEnv() string {
	env := strings.ToUpper(g.ToolName) + "_PULLREQUEST_HEAD"
	if v := os.Getenv(env); v != "" {
		return v
	}
	return ""
}
