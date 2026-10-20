package auth

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// The browser part is simulated: the "browser" follows the authorize URL by
// calling the local callback with the state it was given.
func TestBrowserCallback(t *testing.T) {
	ElyClientID, ElyClientSecret = "test-client", "test-secret"
	defer func() { ElyClientID, ElyClientSecret = "", "" }()

	cases := []struct {
		name  string
		query func(state string) string
		want  string
	}{
		{"denied", func(s string) string { return "state=" + s + "&error=access_denied" }, "@oauthDenied"},
		{"bad state", func(string) string { return "state=forged&code=x" }, "invalid state"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var opened string
			_, err := LoginElyBrowser(context.Background(), func(u string) {
				opened = u
				pu, _ := url.Parse(u)
				q := pu.Query()
				if q.Get("redirect_uri") != OAuthRedirect || q.Get("client_id") != "test-client" {
					t.Errorf("bad authorize url %s", u)
				}
				go http.Get(OAuthRedirect + "?" + c.query(q.Get("state")))
			})
			if opened == "" {
				t.Fatal("browser was not opened")
			}
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(c.want)) {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
		})
	}
}

func TestOAuthNotConfigured(t *testing.T) {
	_, err := LoginElyBrowser(context.Background(), func(string) { t.Fatal("must not open") })
	if err == nil || !strings.Contains(err.Error(), "oauthNotConfigured") {
		t.Fatal(err)
	}
}
