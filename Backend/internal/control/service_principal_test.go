package control

import "testing"

func TestIsBasaltServicePrincipal(t *testing.T) {
	if !hasBasaltServiceActor(map[string]any{"app_id": 77}) {
		t.Fatal("expected exchanged BasaltPass token to be a service principal")
	}

	if hasBasaltServiceActor(nil) {
		t.Fatal("did not expect a user token without act to be a service principal")
	}
	if hasBasaltServiceActor(map[string]any{}) {
		t.Fatal("did not expect an empty actor claim to be a service principal")
	}
}
