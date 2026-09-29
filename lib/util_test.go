package lib

import "testing"

func TestGenerateMD5(t *testing.T) {
	for in, want := range map[string]string{
		"":       "d41d8cd98f00b204e9800998ecf8427e",
		"secret": "5ebe2294ecd0e0f08eab7690d2a6ee69",
	} {
		if got := GenerateMD5(in); got != want {
			t.Errorf("GenerateMD5(%q) = %s, want %s", in, got, want)
		}
	}
}
