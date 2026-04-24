package guidelines

import (
	"net/url"
	"testing"
)

func TestStripHTML(t *testing.T) {
	in := `<html><head><style>body{}</style></head><body><script>alert(1)</script><p>Hello <b>world</b></p></body></html>`
	out := stripHTML(in)
	if out != "Hello world" {
		t.Errorf("stripHTML: %q", out)
	}
}

func TestResolveURL(t *testing.T) {
	base, _ := url.Parse("https://www.nice.org.uk/guidance/published?type=ng")
	got := resolveURL(base, "/guidance/NG123")
	want := "https://www.nice.org.uk/guidance/NG123"
	if got != want {
		t.Errorf("resolveURL: %q != %q", got, want)
	}
}

func TestSha1HexStable(t *testing.T) {
	a := sha1Hex("abc")
	b := sha1Hex("abc")
	if a != b {
		t.Error("sha1Hex not deterministic")
	}
	if len(a) != 40 {
		t.Errorf("sha1Hex hex length: %d", len(a))
	}
}
