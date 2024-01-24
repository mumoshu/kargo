package kargo

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeRepo(t *testing.T) {
	tests := []struct {
		repo  string
		token string
		want  string
		err   string
	}{
		{
			repo:  "github.com:foo/bar.git",
			token: "",
			want:  "github.com:foo/bar.git",
		},
		{
			repo:  "github.com:foo/bar.git",
			token: "abc",
			want:  "github.com:foo/bar.git",
		},
		{
			repo:  "git@github.com:foo/bar.git",
			token: "",
			want:  "git@github.com:foo/bar.git",
		},
		{
			repo:  "git@github.com:foo/bar.git",
			token: "abc",
			want:  "git@github.com:foo/bar.git",
		},
		{
			repo:  "github.com/foo/bar.git",
			token: "",
			err:   "either http(s):// or host:owner/repo.git format is required for repo, but got github.com/foo/bar.git",
		},
		{
			repo:  "github.com/foo/bar.git",
			token: "abc",
			err:   "either http(s):// or host:owner/repo.git format is required for repo, but got github.com/foo/bar.git",
		},
		{
			repo:  "http://github.com/foo/bar.git",
			token: "",
			want:  "http://github.com/foo/bar.git",
		},
		{
			repo:  "http://github.com/foo/bar.git",
			token: "abc",
			want:  "http://github.com/foo/bar.git",
		},
		{
			repo:  "https://github.com/foo/bar.git",
			token: "",
			want:  "https://github.com/foo/bar.git",
		},
		{
			repo:  "https://github.com/foo/bar.git",
			token: "abc",
			want:  "https://kargo:abc@github.com/foo/bar.git",
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got, err := normalizeRepo(tt.repo, tt.token)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
			if tt.err != "" {
				require.Error(t, err)
				require.Equal(t, tt.err, err.Error())
			}
			if tt.err == "" {
				require.NoError(t, err)
			}
		})
	}
}
