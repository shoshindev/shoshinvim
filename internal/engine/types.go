package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const SchemaVersion = 1
const CatalogVersion = "2026.09.28"

type State struct {
	Schema    int                       `json:"schema_version"`
	Catalog   string                    `json:"catalog_version"`
	Profile   string                    `json:"profile"`
	Values    map[string]any            `json:"values"`
	Explicit  map[string]bool           `json:"explicit"`
	Filetypes map[string]map[string]any `json:"filetypes,omitempty"`
	Keymaps   []Keymap                  `json:"keymaps,omitempty"`
}

type Keymap struct {
	Mode   string `json:"mode"`
	Key    string `json:"key"`
	Action string `json:"action"`
}

type File struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Before  string `json:"before_sha256"`
	After   string `json:"after_sha256"`
	Role    string `json:"role"`
	Delete  bool   `json:"delete,omitempty"`
}

type Manifest struct {
	Schema int               `json:"schema_version"`
	Files  map[string]string `json:"files"`
}

type Action struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	Source      string   `json:"source,omitempty"`
	Revision    string   `json:"revision,omitempty"`
	Package     string   `json:"package,omitempty"`
	Parsers     []string `json:"parsers,omitempty"`
	Destination string   `json:"destination"`
	Description string   `json:"description"`
}

type Plan struct {
	Schema        int      `json:"schema_version"`
	Catalog       string   `json:"catalog_version"`
	Target        string   `json:"target"`
	Identity      string   `json:"target_identity"`
	Neovim        string   `json:"neovim"`
	NeovimVersion string   `json:"neovim_version"`
	State         State    `json:"state"`
	Files         []File   `json:"files"`
	Actions       []Action `json:"external_actions"`
	RunExternal   bool     `json:"run_external"`
	Warnings      []string `json:"warnings,omitempty"`
	Digest        string   `json:"digest"`
}

type Dependency struct {
	ID          string `json:"id"`
	Purpose     string `json:"purpose"`
	Path        string `json:"path,omitempty"`
	Status      string `json:"status"`
	Instruction string `json:"instruction,omitempty"`
}

type Inspection struct {
	Target    string   `json:"target"`
	Ownership string   `json:"ownership"`
	State     *State   `json:"state,omitempty"`
	Problems  []string `json:"problems,omitempty"`
}

type ApplyResult struct {
	LocalCommitted bool           `json:"local_committed"`
	Transaction    string         `json:"transaction"`
	External       []ActionResult `json:"external,omitempty"`
}

type ActionResult struct {
	Action Action `json:"action"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func NewState() State {
	return State{Schema: SchemaVersion, Catalog: CatalogVersion, Profile: "custom", Values: map[string]any{}, Explicit: map[string]bool{}}
}
func Clone(s State) State {
	b, _ := json.Marshal(s)
	var out State
	_ = json.Unmarshal(b, &out)
	return out
}
func Equal(a, b State) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}
func Hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func JSON(v any) []byte    { b, _ := json.MarshalIndent(v, "", "  "); return append(b, '\n') }
func DecodeFile(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 16<<20))
	d.DisallowUnknownFields()
	if err = d.Decode(v); err != nil {
		return err
	}
	if err = d.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("unexpected trailing JSON")
	}
	return nil
}

func (p Plan) Fingerprint() string { p.Digest = ""; return Hash(JSON(p)) }
