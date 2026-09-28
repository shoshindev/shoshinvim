package distribution

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"github.com/shoshindev/shoshinvim/internal/engine"
	"testing"
)

func TestChecksumFailsClosed(t *testing.T) {
	b := []byte("binary")
	m := []byte(engine.Hash(b) + "  file.tar.gz\n")
	if VerifyChecksum(m, "file.tar.gz", b) != nil {
		t.Fatal("valid rejected")
	}
	if VerifyChecksum(m, "file.tar.gz", []byte("modified")) == nil {
		t.Fatal("modified accepted")
	}
	if VerifyChecksum(append(m, m...), "file.tar.gz", b) == nil {
		t.Fatal("ambiguous accepted")
	}
}
func TestArchiveOnlyReadsExpectedBinary(t *testing.T) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tr := tar.NewWriter(gz)
	for _, name := range []string{"../../escape", "shoshinvim"} {
		data := []byte(name)
		tr.WriteHeader(&tar.Header{Name: name, Mode: 0755, Size: int64(len(data))})
		tr.Write(data)
	}
	tr.Close()
	gz.Close()
	out, err := Extract(b.Bytes(), false)
	if err != nil || string(out) != "shoshinvim" {
		t.Fatalf("%s %v", out, err)
	}
}
func TestAssets(t *testing.T) {
	for _, target := range [][2]string{{"darwin", "arm64"}, {"darwin", "amd64"}, {"linux", "arm64"}, {"linux", "amd64"}, {"windows", "amd64"}} {
		if _, err := Asset("v0.1.0", target[0], target[1]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Asset("../../bad", "linux", "amd64"); err == nil {
		t.Fatal("unsafe tag")
	}
}
