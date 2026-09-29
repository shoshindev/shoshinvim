package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// ThemeLookup describes a theme repository the user named, pinned to its current commit.
type ThemeLookup struct {
	Repository   string
	Revision     string
	Colorschemes []string // Empty when GitHub could not list them; the user then types a name.
	Stars        int
	Updated      string
	License      string
}

// LookupTheme resolves a GitHub repository's current commit and the colorschemes it
// provides at that commit. It reads metadata only; nothing is downloaded or run.
func LookupTheme(ctx context.Context, repository string) (ThemeLookup, error) {
	repository = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(repository), "https://github.com/"), ".git")
	out := ThemeLookup{Repository: repository}
	if !repositoryPattern.MatchString(repository) {
		return out, fmt.Errorf("use the form owner/name, for example folke/tokyonight.nvim")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "https://github.com/"+repository+".git", "HEAD")
	boundProcess(cmd)
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0") // A missing repository must fail, not ask for a password.
	b, err := cmd.Output()
	fields := strings.Fields(string(b))
	if err != nil || len(fields) == 0 || !revisionPattern.MatchString(fields[0]) {
		return out, fmt.Errorf("could not find github.com/%s", repository)
	}
	out.Revision = fields[0]

	var meta struct {
		Stars   int    `json:"stargazers_count"`
		Pushed  string `json:"pushed_at"`
		License *struct {
			SPDX string `json:"spdx_id"`
		} `json:"license"`
	}
	if githubJSON(ctx, "https://api.github.com/repos/"+repository, &meta) == nil {
		out.Stars = meta.Stars
		if len(meta.Pushed) >= 7 {
			out.Updated = meta.Pushed[:7]
		}
		if meta.License != nil && meta.License.SPDX != "NOASSERTION" {
			out.License = meta.License.SPDX
		}
	}
	var files []struct {
		Name string `json:"name"`
	}
	if githubJSON(ctx, "https://api.github.com/repos/"+repository+"/contents/colors?ref="+out.Revision, &files) == nil {
		for _, f := range files {
			name := strings.TrimSuffix(strings.TrimSuffix(f.Name, ".lua"), ".vim")
			if name != f.Name && colorschemePattern.MatchString(name) {
				out.Colorschemes = append(out.Colorschemes, name)
			}
		}
		sort.Strings(out.Colorschemes)
	}
	return out, nil
}

func githubJSON(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ShoshinVim")
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub returned %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(v)
}
