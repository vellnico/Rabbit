package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vellnico/rabbit/internal/msg"
)

// MSClientID is the Azure application (client) id used for Microsoft
// sign-in. It is injected at build time, see README ("Microsoft"):
//
//	-ldflags "-X github.com/vellnico/rabbit/internal/auth.MSClientID=..."
var MSClientID string

const (
	msDeviceCode = "https://login.microsoftonline.com/consumers/oauth2/v2.0/devicecode"
	msToken      = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"
	msScope      = "XboxLive.signin offline_access"
	xblAuth      = "https://user.auth.xboxlive.com/user/authenticate"
	xstsAuth     = "https://xsts.auth.xboxlive.com/xsts/authorize"
	mcLogin      = "https://api.minecraftservices.com/authentication/login_with_xbox"
	mcProfile    = "https://api.minecraftservices.com/minecraft/profile"
)

// MSConfigured reports whether Microsoft sign-in is available in this build.
func MSConfigured() bool { return MSClientID != "" }

// DeviceCode is shown to the user while waiting for Microsoft sign-in.
type DeviceCode struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

type msTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
	Description  string `json:"error_description"`
}

func postForm(ctx context.Context, endpoint string, form url.Values, v any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return doJSON(req, v)
}

func postJSON(ctx context.Context, endpoint string, body, v any) (int, error) {
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return doJSON(req, v)
}

func doJSON(req *http.Request, v any) (int, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if len(data) > 0 {
		json.Unmarshal(data, v)
	}
	return resp.StatusCode, nil
}

// LoginMicrosoft signs in with the device code flow: show is called with the
// code the user enters on the Microsoft page, then it waits for the sign-in.
func LoginMicrosoft(ctx context.Context, show func(DeviceCode)) (*Account, error) {
	if !MSConfigured() {
		return nil, msg.E("msNotConfigured")
	}
	var dc struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
		Error           string `json:"error"`
		Description     string `json:"error_description"`
	}
	code, err := postForm(ctx, msDeviceCode, url.Values{"client_id": {MSClientID}, "scope": {msScope}}, &dc)
	if err != nil {
		return nil, msg.Wrap("msUnavailable", err)
	}
	if code != http.StatusOK || dc.DeviceCode == "" {
		return nil, fmt.Errorf("Microsoft: %s %s", dc.Error, firstLine(dc.Description))
	}
	show(DeviceCode{Code: dc.UserCode, URL: dc.VerificationURI})

	interval := time.Duration(max(dc.Interval, 1)) * time.Second
	ctx, cancel := context.WithTimeout(ctx, time.Duration(max(dc.ExpiresIn, 60))*time.Second)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, msg.E("msCodeExpired")
			}
			return nil, msg.E("oauthCancelled")
		case <-time.After(interval):
		}
		var t msTokenResponse
		if _, err := postForm(ctx, msToken, url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"client_id":   {MSClientID},
			"device_code": {dc.DeviceCode},
		}, &t); err != nil {
			if ctx.Err() != nil {
				continue // reported by the select above
			}
			return nil, msg.Wrap("msUnavailable", err)
		}
		switch t.Error {
		case "":
			acc := &Account{Kind: Microsoft, RefreshToken: t.RefreshToken}
			if err := minecraftLogin(ctx, acc, t.AccessToken); err != nil {
				return nil, err
			}
			return acc, nil
		case "authorization_pending":
		case "slow_down":
			interval += 5 * time.Second
		case "authorization_declined":
			return nil, msg.E("oauthDenied")
		case "expired_token":
			return nil, msg.E("msCodeExpired")
		default:
			return nil, fmt.Errorf("Microsoft: %s %s", t.Error, firstLine(t.Description))
		}
	}
}

// refreshMicrosoft renews the Minecraft token before it expires. ok=false
// means the user has to sign in again.
func refreshMicrosoft(ctx context.Context, acc *Account) (bool, error) {
	if time.Until(acc.ExpiresAt) > 30*time.Minute {
		return true, nil
	}
	if acc.RefreshToken == "" || !MSConfigured() {
		return false, nil
	}
	var t msTokenResponse
	_, err := postForm(ctx, msToken, url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {MSClientID},
		"refresh_token": {acc.RefreshToken},
		"scope":         {msScope},
	}, &t)
	if err != nil {
		return false, err
	}
	if t.AccessToken == "" {
		return false, nil
	}
	if t.RefreshToken != "" {
		acc.RefreshToken = t.RefreshToken
	}
	if err := minecraftLogin(ctx, acc, t.AccessToken); err != nil {
		return false, err
	}
	return true, nil
}

// minecraftLogin trades a Microsoft token for a Minecraft one through Xbox
// Live and fills the account's profile.
func minecraftLogin(ctx context.Context, acc *Account, msToken string) error {
	type xboxResponse struct {
		Token         string `json:"Token"`
		DisplayClaims struct {
			Xui []struct {
				Uhs string `json:"uhs"`
			} `json:"xui"`
		} `json:"DisplayClaims"`
		XErr int64 `json:"XErr"`
	}
	var xbl xboxResponse
	code, err := postJSON(ctx, xblAuth, map[string]any{
		"Properties": map[string]any{
			"AuthMethod": "RPS",
			"SiteName":   "user.auth.xboxlive.com",
			"RpsTicket":  "d=" + msToken,
		},
		"RelyingParty": "http://auth.xboxlive.com",
		"TokenType":    "JWT",
	}, &xbl)
	if err != nil {
		return msg.Wrap("msUnavailable", err)
	}
	if code != http.StatusOK || xbl.Token == "" {
		return fmt.Errorf("Xbox Live: HTTP %d", code)
	}

	var xsts xboxResponse
	code, err = postJSON(ctx, xstsAuth, map[string]any{
		"Properties": map[string]any{
			"SandboxId":  "RETAIL",
			"UserTokens": []string{xbl.Token},
		},
		"RelyingParty": "rp://api.minecraftservices.com/",
		"TokenType":    "JWT",
	}, &xsts)
	if err != nil {
		return msg.Wrap("msUnavailable", err)
	}
	if code != http.StatusOK || xsts.Token == "" || len(xsts.DisplayClaims.Xui) == 0 {
		switch xsts.XErr {
		case 2148916233:
			return msg.E("xboxNoAccount")
		case 2148916235:
			return msg.E("xboxRegion")
		case 2148916236, 2148916237:
			return msg.E("xboxAdult")
		case 2148916238:
			return msg.E("xboxChild")
		}
		return fmt.Errorf("Xbox Live (XSTS): HTTP %d %d", code, xsts.XErr)
	}

	var mc struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	code, err = postJSON(ctx, mcLogin, map[string]string{
		"identityToken": "XBL3.0 x=" + xsts.DisplayClaims.Xui[0].Uhs + ";" + xsts.Token,
	}, &mc)
	if err != nil {
		return msg.Wrap("msUnavailable", err)
	}
	if code == http.StatusForbidden {
		// New Azure apps must be approved by Mojang to use this API.
		return msg.E("msAppNotApproved")
	}
	if code != http.StatusOK || mc.AccessToken == "" {
		return fmt.Errorf("Minecraft: HTTP %d", code)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mcProfile, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+mc.AccessToken)
	var p struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Skins []struct {
			State string `json:"state"`
			URL   string `json:"url"`
		} `json:"skins"`
	}
	code, err = doJSON(req, &p)
	if err != nil {
		return msg.Wrap("msUnavailable", err)
	}
	if code == http.StatusNotFound || (code == http.StatusOK && p.ID == "") {
		return msg.E("msNoGame")
	}
	if code != http.StatusOK {
		return fmt.Errorf("Minecraft profile: HTTP %d", code)
	}

	acc.ID = "microsoft-" + p.ID
	acc.Name = p.Name
	acc.UUID = p.ID
	acc.AccessToken = mc.AccessToken
	acc.ExpiresAt = time.Now().Add(time.Duration(mc.ExpiresIn) * time.Second)
	acc.Skin = ""
	for _, s := range p.Skins {
		if s.State == "ACTIVE" {
			acc.Skin = strings.Replace(s.URL, "http://", "https://", 1)
		}
	}
	return nil
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
