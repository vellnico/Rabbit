package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vellnico/rabbit/internal/download"
	"github.com/vellnico/rabbit/internal/msg"
)

// Credentials of the Ely.by OAuth application. They are injected at build
// time, see README ("Ely.by OAuth"):
//
//	-ldflags "-X github.com/vellnico/rabbit/internal/auth.ElyClientID=... -X ...ElyClientSecret=..."
var (
	ElyClientID     string
	ElyClientSecret string
)

const (
	elyAuthorize = "https://account.ely.by/oauth2/v1"
	elyToken     = "https://account.ely.by/api/oauth2/v1/token"
	elyInfo      = "https://account.ely.by/api/account/v1/info"
	elyScope     = "account_info minecraft_server_session offline_access"

	// OAuthRedirect must be registered as the Redirect URI of the app.
	OAuthRedirect = "http://127.0.0.1:25585/callback"
	oauthListen   = "127.0.0.1:25585"
)

// OAuthConfigured reports whether browser login is available in this build.
func OAuthConfigured() bool { return ElyClientID != "" && ElyClientSecret != "" }

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	Message      string `json:"message"`
}

func tokenRequest(ctx context.Context, form url.Values) (*tokenResponse, error) {
	form.Set("client_id", ElyClientID)
	form.Set("client_secret", ElyClientSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, elyToken, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := download.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var t tokenResponse
	json.NewDecoder(resp.Body).Decode(&t)
	if resp.StatusCode != http.StatusOK || t.AccessToken == "" {
		if t.Message != "" {
			return nil, errors.New(t.Message)
		}
		return nil, fmt.Errorf("ely.by: %s %s", resp.Status, t.Error)
	}
	return &t, nil
}

func accountInfo(ctx context.Context, token string) (uuid, name string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, elyInfo, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := download.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	var info struct {
		UUID     string `json:"uuid"`
		Username string `json:"username"`
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("ely.by: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", "", err
	}
	return strings.ReplaceAll(info.UUID, "-", ""), info.Username, nil
}

const callbackPage = `<!doctype html><meta charset="utf-8"><title>Rabbit</title>
<style>body{margin:0;height:100vh;display:grid;place-items:center;background:#111113;color:#ececee;font:16px "Segoe UI",system-ui,sans-serif}
div{text-align:center}b{font-size:22px;display:block;margin-bottom:8px;color:%s}</style>
<div><b>%s</b>Rabbit · Ely.by</div>`

// LoginElyBrowser opens the Ely.by consent page via open and waits until
// the browser is redirected back to the local callback server.
func LoginElyBrowser(ctx context.Context, open func(string)) (*Account, error) {
	if !OAuthConfigured() {
		return nil, msg.E("oauthNotConfigured")
	}
	ln, err := net.Listen("tcp", oauthListen)
	if err != nil {
		return nil, msg.Wrap("oauthPortBusy", err)
	}
	state := randomHex(16)
	type result struct {
		code string
		err  error
	}
	done := make(chan result, 1)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/callback" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var res result
		switch {
		case q.Get("state") != state:
			res.err = errors.New("invalid state")
		case q.Get("error") != "":
			res.err = msg.E("oauthDenied")
		default:
			res.code = q.Get("code")
		}
		if res.err != nil {
			fmt.Fprintf(w, callbackPage, "#f87171", "✕")
		} else {
			fmt.Fprintf(w, callbackPage, "#4ade80", "✓")
		}
		select {
		case done <- res:
		default:
		}
	})}
	go srv.Serve(ln)
	defer srv.Close()

	q := url.Values{
		"client_id":     {ElyClientID},
		"redirect_uri":  {OAuthRedirect},
		"response_type": {"code"},
		"scope":         {elyScope},
		"state":         {state},
		"prompt":        {"select_account"},
	}
	open(elyAuthorize + "?" + q.Encode())

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	var res result
	select {
	case res = <-done:
	case <-ctx.Done():
		return nil, msg.E("oauthCancelled")
	}
	if res.err != nil {
		return nil, res.err
	}

	t, err := tokenRequest(ctx, url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {res.code},
		"redirect_uri": {OAuthRedirect},
	})
	if err != nil {
		return nil, err
	}
	uuid, name, err := accountInfo(ctx, t.AccessToken)
	if err != nil {
		return nil, err
	}
	return &Account{
		ID:           "elyby-" + uuid,
		Kind:         ElyBy,
		Name:         name,
		UUID:         uuid,
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(t.ExpiresIn) * time.Second),
		OAuth:        true,
	}, nil
}

// refreshOAuth renews an expiring OAuth token. ok=false means the user has
// to sign in again.
func refreshOAuth(ctx context.Context, acc *Account) (bool, error) {
	if time.Until(acc.ExpiresAt) > 30*time.Minute {
		return true, nil
	}
	if acc.RefreshToken == "" || !OAuthConfigured() {
		return false, nil
	}
	t, err := tokenRequest(ctx, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {acc.RefreshToken},
		"scope":         {elyScope},
	})
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) {
			return false, err
		}
		return false, nil
	}
	acc.AccessToken = t.AccessToken
	acc.ExpiresAt = time.Now().Add(time.Duration(t.ExpiresIn) * time.Second)
	if _, name, err := accountInfo(ctx, t.AccessToken); err == nil && name != "" {
		acc.Name = name
	}
	return true, nil
}
