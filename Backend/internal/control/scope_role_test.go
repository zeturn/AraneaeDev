package control

import "testing"

func TestNormalizeScopes(t *testing.T) {
	got := normalizeScopes(" araneae.read  ARANEAE.WRITE  araneae.read ")
	want := "araneae.read araneae.write"
	if got != want {
		t.Fatalf("normalizeScopes mismatch: got=%q want=%q", got, want)
	}
}

func TestRoleFromScope(t *testing.T) {
	cases := []struct {
		name   string
		scopes string
		want   string
	}{
		{name: "admin", scopes: "openid araneae.admin", want: "admin"},
		{name: "operator", scopes: "araneae.write", want: "operator"},
		{name: "viewer", scopes: "araneae.read", want: "viewer"},
		{name: "default viewer", scopes: "openid profile", want: "viewer"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := roleFromScope(tc.scopes); got != tc.want {
				t.Fatalf("roleFromScope mismatch: got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestHasScopeSupportsWildcards(t *testing.T) {
	if !hasScope("araneae.*", "araneae.write") {
		t.Fatal("expected araneae.* to satisfy araneae.write")
	}
	if !hasScope("*", "araneae.read") {
		t.Fatal("expected * to satisfy araneae.read")
	}
	if hasScope("araneae.read", "araneae.write") {
		t.Fatal("did not expect araneae.read to satisfy araneae.write")
	}
}
