package bridge

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ipanalytics/mmdbbridge/internal/schema"
	"github.com/oschwald/maxminddb-golang"
)

func TestBuildAndExportRoundTrip(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.yaml")
	csvPath := filepath.Join(dir, "input.csv")
	mmdbPath := filepath.Join(dir, "out.mmdb")
	exportPath := filepath.Join(dir, "export.csv")

	writeFile(t, schemaPath, schema.Example)
	writeFile(t, csvPath, "cidr,asn,is_vpn,country,tags\n203.0.113.0/24,64500,true,US,vpn|hosting\n2001:db8:42::/48,64501,false,DE,enterprise|office\n")

	s, err := schema.Load(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Build(s, csvPath, mmdbPath); err != nil {
		t.Fatal(err)
	}

	db, err := maxminddb.Open(mmdbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var got map[string]any
	if err := db.Lookup(mustIP("203.0.113.7"), &got); err != nil {
		t.Fatal(err)
	}
	if toFloat64(got["asn"]) != 64500 {
		t.Fatalf("asn = %#v", got["asn"])
	}
	if got["is_vpn"] != true {
		t.Fatalf("is_vpn = %#v", got["is_vpn"])
	}

	if err := Export(s, mmdbPath, exportPath); err != nil {
		t.Fatal(err)
	}
	exported, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(exported)
	for _, want := range []string{
		"cidr,asn,country,is_vpn,tags",
		"203.0.113.0/24,64500,US,true,vpn|hosting",
		"2001:db8:42::/48,64501,DE,false,enterprise|office",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("export missing %q:\n%s", want, text)
		}
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustIP(s string) []byte {
	ip := net.ParseIP(s)
	if ip == nil {
		panic(s)
	}
	return ip
}
