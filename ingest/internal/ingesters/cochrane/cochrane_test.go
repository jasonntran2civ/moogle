package cochrane

import "testing"

func TestExtractDOIFromCochraneURL(t *testing.T) {
	cases := map[string]string{
		"https://www.cochranelibrary.com/cdsr/doi/10.1002/14651858.CD000234.pub5/full": "10.1002/14651858.CD000234.pub5",
		"https://www.cochranelibrary.com/cdsr/doi/10.1002/14651858.CD111111.pub2":      "10.1002/14651858.CD111111.pub2",
		"https://www.cochranelibrary.com/cdsr/doi/10.1002/X.pub3?utm=src":              "10.1002/X.pub3",
		"https://example.org/no/doi/here":                                              "here",
		"":                                                                             "",
	}
	for in, want := range cases {
		if got := extractDOIFromCochraneURL(in); got != want {
			t.Errorf("extractDOIFromCochraneURL(%q) = %q, want %q", in, got, want)
		}
	}
}
