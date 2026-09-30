package cli

import (
	"fmt"
	"strings"

	"github.com/shoshindev/shoshinvim/internal/engine"
)

// Text forms of command results. --output json prints the structures themselves.

var ownershipText = map[string]string{
	"NEW":       "new: nothing here yet; ShoshinVim can create a configuration",
	"SAVED":     "made by ShoshinVim; it can be changed",
	"READ ONLY": "a configuration ShoshinVim did not create; it will not be changed",
	"RECOVERY":  "needs recovery: an earlier save was interrupted, or generated files were edited by hand",
}

func inspectionText(ins engine.Inspection) string {
	status, ok := ownershipText[ins.Ownership]
	if !ok {
		status = strings.ToLower(ins.Ownership)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Target  %s\nStatus  %s", ins.Target, status)
	for _, p := range ins.Problems {
		fmt.Fprintf(&b, "\n        · %s", p)
	}
	return b.String()
}

func doctorText(ins engine.Inspection, deps []engine.Dependency, probe string, probed bool) string {
	var b strings.Builder
	b.WriteString(inspectionText(ins) + "\n\nPrograms\n")
	width := 0
	for _, d := range deps {
		width = max(width, len(d.ID))
	}
	for _, d := range deps {
		mark := "–"
		switch d.Status {
		case "present":
			mark = "✓"
		case "missing":
			mark = "✗"
		}
		fmt.Fprintf(&b, "  %s %-*s  %s", mark, width, d.ID, d.Purpose)
		if d.Status == "present" && d.Path != "" {
			fmt.Fprintf(&b, " (%s)", d.Path)
		} else if d.Status != "present" && d.Instruction != "" {
			fmt.Fprintf(&b, "\n  %*s  %s", width+2, "", d.Instruction)
		}
		b.WriteString("\n")
	}
	mark := "✗"
	if probed {
		mark = "✓"
	}
	fmt.Fprintf(&b, "\nValidation\n  %s %s", mark, probe)
	return b.String()
}

func applyText(r engine.ApplyResult) string {
	text := "Local configuration saved and verified. Restart Neovim.\nTransaction " + r.Transaction +
		" (undo with: shoshinvim rollback --transaction " + r.Transaction + ")"
	if len(r.External) > 0 {
		text += "\n\nDownloads\n" + actionsText(r.External)
	}
	return text
}

func actionsText(rs []engine.ActionResult) string {
	if len(rs) == 0 {
		return "Nothing to retry."
	}
	lines := make([]string, len(rs))
	for i, r := range rs {
		mark := "–"
		switch r.Status {
		case "installed":
			mark = "✓"
		case "failed":
			mark = "✗"
		}
		lines[i] = fmt.Sprintf("  %s %s: %s", mark, r.Action.Description, r.Status)
		if r.Detail != "" {
			lines[i] += " (" + r.Detail + ")"
		}
	}
	return strings.Join(lines, "\n")
}
