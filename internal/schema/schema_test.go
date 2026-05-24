package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExample(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schema.yaml")
	if err := os.WriteFile(path, []byte(Example), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Network.Column != "cidr" {
		t.Fatalf("network column = %q", s.Network.Column)
	}
	if s.Metadata.BuildEpoch != 1 {
		t.Fatalf("build epoch = %d", s.Metadata.BuildEpoch)
	}
}
