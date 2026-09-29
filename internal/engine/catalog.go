package engine

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

//go:embed catalog.json
var catalogData []byte

//go:embed data/options.json
var optionData []byte

type Setting struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Option      string   `json:"option,omitempty"`
	Description string   `json:"description"`
	Choices     []string `json:"choices,omitempty"`
}
type NativeOption struct {
	Name        string `json:"name"`
	Short       string `json:"shortname"`
	Type        string `json:"type"`
	Scope       string `json:"scope"`
	GlobalLocal bool   `json:"global_local"`
	Default     any    `json:"default"`
}
type Catalog struct {
	Settings []Setting                 `json:"settings"`
	Profiles map[string]map[string]any `json:"profiles"`
	Options  map[string]NativeOption   `json:"-"`
}

func LoadCatalog() Catalog {
	var c Catalog
	if err := json.Unmarshal(catalogData, &c); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(optionData, &c.Options); err != nil {
		panic(err)
	}
	for i := range c.Settings {
		if c.Settings[i].ID == "appearance.colorscheme" {
			c.Settings[i].Choices = themeChoices()
		}
	}
	for _, l := range Languages() {
		c.Settings = append(c.Settings, Setting{ID: "language." + l.ID + ".lsp", Label: l.Label + " language server", Type: "boolean",
			Description: "Use " + l.Server.Name + " (" + l.Server.Command + ") for " + l.Label + "."})
	}
	return c
}

// Terminal-derived, read-only and process identity values are documentation only.
var readonly = map[string]bool{"channel": true, "columns": true, "lines": true, "modified": true, "readonly": false, "window": true, "term": true, "ttytype": true, "ttymouse": true, "t_Co": true, "shelltemp": true, "encoding": true, "termencoding": true, "loadplugins": true}
var identifier = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
var filetypeID = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.-]*$`)

func (c Catalog) Find(id string) (Setting, error) {
	if strings.HasPrefix(id, "tools.path.") {
		name := strings.TrimPrefix(id, "tools.path.")
		if toolExecutables()[name] {
			return Setting{ID: id, Label: name + " executable", Type: "string", Description: "Absolute path to an existing system or project executable; empty uses PATH/Mason."}, nil
		}
	}
	for _, s := range c.Settings {
		if s.ID == id {
			return s, nil
		}
	}
	if strings.HasPrefix(id, "options.") {
		name := strings.TrimPrefix(id, "options.")
		if o, ok := c.Options[name]; ok && !readonly[name] {
			return Setting{ID: id, Label: name, Type: o.Type, Option: name, Description: "Neovim :help '" + name + "'; scope: " + o.Scope + "; metadata: Neovim 0.12.5"}, nil
		}
	}
	return Setting{}, fmt.Errorf("unsupported setting %q; use /search", id)
}
func canonical(s Setting) string {
	if s.Option != "" {
		return "options." + s.Option
	}
	return s.ID
}
func (c Catalog) Parse(id, text string) (any, error) {
	s, err := c.Find(id)
	if err != nil {
		return nil, err
	}
	switch s.Type {
	case "boolean":
		switch text {
		case "true", "on", "yes":
			return true, nil
		case "false", "off", "no":
			return false, nil
		}
		return nil, fmt.Errorf("%s expects on or off", id)
	case "number":
		v, err := strconv.ParseFloat(text, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v != math.Trunc(v) {
			return nil, fmt.Errorf("%s expects an integer", id)
		}
		return v, nil
	default:
		return text, nil
	}
}
func (c Catalog) Set(s *State, id string, value any, explicit bool) error {
	meta, err := c.Find(id)
	if err != nil {
		return err
	}
	if err = checkValue(meta, value); err != nil {
		return err
	}
	if s.Values == nil {
		s.Values = map[string]any{}
	}
	if s.Explicit == nil {
		s.Explicit = map[string]bool{}
	}
	s.Values[canonical(meta)] = value
	if explicit {
		s.Explicit[canonical(meta)] = true
	}
	return nil
}
func checkValue(s Setting, v any) error {
	valid := false
	switch s.Type {
	case "boolean":
		_, valid = v.(bool)
	case "number":
		n, ok := v.(float64)
		if !ok {
			if i, ok := v.(int); ok {
				n = float64(i)
				valid = true
			}
		} else {
			valid = true
		}
		valid = valid && !math.IsNaN(n) && !math.IsInf(n, 0) && n == math.Trunc(n) && math.Abs(n) <= 2147483647
	case "string":
		_, valid = v.(string)
	case "enum":
		if x, ok := v.(string); ok {
			for _, choice := range s.Choices {
				if choice == x {
					valid = true
				}
			}
		}
	}
	if !valid {
		return fmt.Errorf("invalid value for %s (%s; choices %v)", s.ID, s.Type, s.Choices)
	}
	if str, ok := v.(string); ok && (strings.ContainsRune(str, 0) || len(str) > 8192) {
		return fmt.Errorf("invalid string for %s", s.ID)
	}
	if str, ok := v.(string); ok && strings.HasPrefix(s.ID, "tools.path.") && str != "" && !filepath.IsAbs(str) {
		return fmt.Errorf("%s requires an absolute executable path", s.ID)
	}
	return nil
}
func (c Catalog) Profile(s State, name string) (State, error) {
	values, ok := c.Profiles[name]
	if !ok {
		return s, fmt.Errorf("profile %q is planned; available: rich-ide, custom, minimal, web, lua", name)
	}
	// A profile replaces the previous proposal instead of layering on it; only
	// explicit choices, filetype settings and keymaps carry over.
	next := NewState()
	next.Profile = name
	prev := Clone(s)
	next.Filetypes, next.Keymaps = prev.Filetypes, prev.Keymaps
	for id, v := range values {
		if err := c.Set(&next, id, v, false); err != nil {
			return s, err
		}
	}
	for id, explicit := range s.Explicit {
		if v, ok := s.Values[id]; ok && explicit {
			next.Values[id] = v
			next.Explicit[id] = true
		}
	}
	if len(Plugins(next)) > 0 {
		next.Values["plugins.manager"] = "lazy"
	}
	return next, nil
}
func (c Catalog) Validate(s State) error {
	if s.Schema != SchemaVersion || s.Catalog != CatalogVersion {
		return fmt.Errorf("unsupported state/catalog version; explicit migration required")
	}
	if _, ok := c.Profiles[s.Profile]; !ok {
		return fmt.Errorf("unsupported profile %q", s.Profile)
	}
	for id, v := range s.Values {
		meta, err := c.Find(id)
		if err != nil {
			return err
		}
		if canonical(meta) != id {
			return fmt.Errorf("state must use canonical ID %s", canonical(meta))
		}
		if err = checkValue(meta, v); err != nil {
			return err
		}
	}
	for ft, values := range s.Filetypes {
		if !filetypeID.MatchString(ft) {
			return fmt.Errorf("invalid filetype %q", ft)
		}
		for key, v := range values {
			o, ok := c.Options[key]
			if !ok || readonly[key] || (o.Scope != "buf" && o.Scope != "win") {
				return fmt.Errorf("%s is not an editable local option", key)
			}
			if err := checkValue(Setting{ID: key, Type: o.Type}, v); err != nil {
				return err
			}
		}
	}
	if err := validateKeymaps(s); err != nil {
		return err
	}
	if err := validateCustomTheme(s); err != nil {
		return err
	}
	if Value(s, "plugins.manager", "none") == "none" && len(Plugins(s)) > 0 {
		return fmt.Errorf("selected plugins require plugins.manager=lazy; review manager selection")
	}
	return nil
}
func Value(s State, id string, fallback any) any {
	if v, ok := s.Values[id]; ok {
		return v
	}
	return fallback
}
func Enabled(s State, id string) bool { return Value(s, id, false) == true }
func (c Catalog) Search(query string) []Setting {
	all := append([]Setting{}, c.Settings...)
	for name := range c.Options {
		if meta, err := c.Find("options." + name); err == nil {
			all = append(all, meta)
		}
	}
	var out []Setting
	query = strings.ToLower(query)
	for _, s := range all {
		if strings.Contains(strings.ToLower(s.ID+" "+s.Label+" "+s.Description), query) {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
