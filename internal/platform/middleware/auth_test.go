package middleware

import "testing"

func TestAuthorizedParty(t *testing.T) {
	allowed := map[string]struct{}{"https://openaiq.org": {}}
	cases := []struct {
		name   string
		native bool
		party  string
		want   bool
	}{
		{"listed origin", false, "https://openaiq.org", true},
		{"unlisted origin", true, "https://evil.example", false},
		{"native token rejected by default", false, "", false},
		{"native token accepted when enabled", true, "", true},
	}
	for _, tc := range cases {
		if got := authorizedParty(allowed, tc.native, tc.party); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
