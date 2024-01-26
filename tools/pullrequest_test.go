package tools

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRepository(t *testing.T) {
	tests := []struct {
		gitRemoteGetURLOut string
		want               *Repository
		err                string
	}{
		{
			gitRemoteGetURLOut: `git@github.com:mumoshu/kargo.git
`,
			want: &Repository{
				Owner: "mumoshu",
				Name:  "kargo",
			},
		},
		{
			gitRemoteGetURLOut: `https://github.com/mumoshu/kargo.git
`,
			want: &Repository{
				Owner: "mumoshu",
				Name:  "kargo",
			},
		},
		{
			gitRemoteGetURLOut: `https://user:pass@github.com/mumoshu/kargo.git
`,
			want: &Repository{
				Owner: "mumoshu",
				Name:  "kargo",
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got, err := parseRepository([]byte(tt.gitRemoteGetURLOut))

			if tt.err != "" {
				require.Error(t, err)
				require.Equal(t, tt.err, err.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.want, got)
		})
	}
}
