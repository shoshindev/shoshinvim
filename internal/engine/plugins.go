package engine

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

//go:embed data/plugins.json
var pluginData []byte

// Plugin is one reviewed upstream plugin. Revision pins what is downloaded; the other
// fields are transparency metadata captured on the review date, not live data.
type Plugin struct {
	ID         string   `json:"-"`
	Repository string   `json:"repository"`
	Revision   string   `json:"revision"`
	Reviewed   string   `json:"reviewed"`
	Release    string   `json:"release,omitempty"`
	Stars      int      `json:"stars"`
	Updated    string   `json:"updated"`
	License    string   `json:"license"`
	UsedBy     []string `json:"used_by,omitempty"`
	Requires   []string `json:"requires,omitempty"`
}

var (
	registryOnce sync.Once
	registry     map[string]Plugin
)

func plugins() map[string]Plugin {
	registryOnce.Do(func() {
		if err := json.Unmarshal(pluginData, &registry); err != nil {
			panic(err)
		}
		for id, p := range registry {
			p.ID = id
			registry[id] = p
		}
	})
	return registry
}

// PluginInfo returns the registry entry for a plugin ID.
func PluginInfo(id string) (Plugin, bool) {
	p, ok := plugins()[id]
	return p, ok
}

// withRequirements adds every plugin that the selected plugins depend on.
func withRequirements(ids []string) []string {
	set := map[string]bool{}
	var add func(string)
	add = func(id string) {
		if set[id] {
			return
		}
		set[id] = true
		if p, ok := plugins()[id]; ok {
			for _, dep := range p.Requires {
				add(dep)
			}
		}
	}
	for _, id := range ids {
		add(id)
	}
	return sorted(set)
}

// Custom themes come from a repository the user names; they are pinned but not reviewed.
const customThemeID = "custom-theme"

var (
	repositoryPattern  = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	revisionPattern    = regexp.MustCompile(`^[0-9a-f]{40}$`)
	colorschemePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
)

// source returns what to download for a plugin ID and the folder it lives in.
func source(s State, id string) (repository, revision, dir string, err error) {
	if id == customThemeID {
		repo, _ := s.Values["appearance.custom_theme.repository"].(string)
		rev, _ := s.Values["appearance.custom_theme.revision"].(string)
		if !repositoryPattern.MatchString(repo) || !revisionPattern.MatchString(rev) {
			return "", "", "", fmt.Errorf("custom theme needs a repository and a pinned revision")
		}
		// The revision is part of the folder, so choosing another commit never collides.
		dir = "theme-" + strings.ReplaceAll(strings.ToLower(repo), "/", "-") + "-" + rev[:7]
		return repo, rev, dir, nil
	}
	p, ok := PluginInfo(id)
	if !ok {
		return "", "", "", fmt.Errorf("missing adapter %s", id)
	}
	return p.Repository, p.Revision, id, nil
}

// PluginsInUse describes each plugin a state installs, for review screens.
type PluginUse struct {
	Plugin
	Folder string
	Listed bool // In ShoshinVim's reviewed catalog; false for a custom theme.
}

func PluginsInUse(s State) ([]PluginUse, error) {
	var out []PluginUse
	ids := Plugins(s)
	if Value(s, "plugins.manager", "none") == "lazy" {
		ids = append([]string{"lazy"}, ids...)
	}
	for _, id := range ids {
		repo, rev, dir, err := source(s, id)
		if err != nil {
			return nil, err
		}
		p, known := PluginInfo(id)
		if !known {
			p = Plugin{ID: id, Repository: repo, Revision: rev}
		}
		out = append(out, PluginUse{Plugin: p, Folder: filepath.ToSlash(filepath.Join("plugins", dir)), Listed: known})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID == "lazy" && out[j].ID != "lazy" })
	return out, nil
}
