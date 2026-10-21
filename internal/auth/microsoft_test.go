package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeMicrosoft answers every endpoint of the sign-in chain in-process.
type fakeMicrosoft struct {
	pending int   // authorization_pending answers before the token
	xErr    int64 // XSTS error, 0 for success
	noGame  bool
}

func (f *fakeMicrosoft) RoundTrip(r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	reply := func(code int, v any) {
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(v)
	}
	xbox := map[string]any{"Token": "xtoken", "DisplayClaims": map[string]any{"xui": []any{map[string]string{"uhs": "hash"}}}}
	switch r.URL.String() {
	case msDeviceCode:
		reply(200, map[string]any{"device_code": "dev", "user_code": "ABCD-1234", "verification_uri": "https://www.microsoft.com/link", "expires_in": 60, "interval": 0})
	case msToken:
		r.ParseForm()
		if r.Form.Get("grant_type") == "refresh_token" {
			reply(200, map[string]string{"access_token": "ms2", "refresh_token": "refresh2"})
		} else if f.pending > 0 {
			f.pending--
			reply(400, map[string]string{"error": "authorization_pending"})
		} else {
			reply(200, map[string]string{"access_token": "ms", "refresh_token": "refresh"})
		}
	case xblAuth:
		reply(200, xbox)
	case xstsAuth:
		if f.xErr != 0 {
			reply(401, map[string]int64{"XErr": f.xErr})
		} else {
			reply(200, xbox)
		}
	case mcLogin:
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["identityToken"] != "XBL3.0 x=hash;xtoken" {
			reply(400, nil)
		} else {
			reply(200, map[string]any{"access_token": "mc", "expires_in": 86400})
		}
	case mcProfile:
		if f.noGame || r.Header.Get("Authorization") != "Bearer mc" {
			reply(404, nil)
		} else {
			reply(200, map[string]any{"id": "0123456789abcdef0123456789abcdef", "name": "Steve",
				"skins": []any{map[string]string{"state": "ACTIVE", "url": "http://textures.minecraft.net/texture/x"}}})
		}
	default:
		reply(404, nil)
	}
	return w.Result(), nil
}

func withFake(t *testing.T, f *fakeMicrosoft) {
	old := http.DefaultClient.Transport
	http.DefaultClient.Transport = f
	MSClientID = "test-client"
	t.Cleanup(func() { http.DefaultClient.Transport, MSClientID = old, "" })
}

func TestMicrosoftLogin(t *testing.T) {
	withFake(t, &fakeMicrosoft{pending: 1})
	var shown DeviceCode
	acc, err := LoginMicrosoft(context.Background(), func(dc DeviceCode) { shown = dc })
	if err != nil {
		t.Fatal(err)
	}
	if shown.Code != "ABCD-1234" {
		t.Errorf("code = %q", shown.Code)
	}
	if acc.Kind != Microsoft || acc.Name != "Steve" || acc.AccessToken != "mc" || acc.RefreshToken != "refresh" ||
		acc.ID != "microsoft-0123456789abcdef0123456789abcdef" || !strings.HasPrefix(acc.Skin, "https://") {
		t.Fatalf("account = %+v", acc)
	}

	// A fresh token is kept, an expiring one is renewed.
	if ok, err := Refresh(context.Background(), acc); !ok || err != nil || acc.RefreshToken != "refresh" {
		t.Fatalf("refresh of a fresh token: %v %v", ok, err)
	}
	acc.ExpiresAt = acc.ExpiresAt.AddDate(0, 0, -1)
	if ok, err := Refresh(context.Background(), acc); !ok || err != nil || acc.RefreshToken != "refresh2" {
		t.Fatalf("refresh of an expired token: %v %v %+v", ok, err, acc)
	}
}

func TestMicrosoftErrors(t *testing.T) {
	cases := []struct {
		fake fakeMicrosoft
		want string
	}{
		{fakeMicrosoft{xErr: 2148916233}, "@xboxNoAccount"},
		{fakeMicrosoft{xErr: 2148916238}, "@xboxChild"},
		{fakeMicrosoft{noGame: true}, "@msNoGame"},
	}
	for _, c := range cases {
		withFake(t, &c.fake)
		_, err := LoginMicrosoft(context.Background(), func(DeviceCode) {})
		if err == nil || err.Error() != c.want {
			t.Errorf("err = %v, want %s", err, c.want)
		}
	}
}

func TestMicrosoftNotConfigured(t *testing.T) {
	_, err := LoginMicrosoft(context.Background(), func(DeviceCode) { t.Fatal("must not show a code") })
	if err == nil || err.Error() != "@msNotConfigured" {
		t.Fatal(err)
	}
}
