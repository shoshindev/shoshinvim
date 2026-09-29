package engine

import (
	"github.com/aymanbagabas/go-udiff"
	"strings"
)

func Diff(p Plan) string {
	var b strings.Builder
	for _, f := range p.Files {
		if f.Before == f.After {
			continue
		}
		before, _ := readPath(p.Target, f.Path)
		after := f.Content
		if f.Delete {
			after = ""
		}
		diff := udiff.Unified("a/"+f.Path, "b/"+f.Path, string(before), after)
		b.WriteString(diff)
	}
	return b.String()
}
