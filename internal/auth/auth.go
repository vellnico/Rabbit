// Package auth handles offline and Ely.by accounts.
package auth

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/vellnico/rabbit/internal/download"
	"github.com/vellnico/rabbit/internal/msg"
	"github.com/vellnico/rabbit/internal/paths"
)

const (
	elyAuthServer = "https://authserver.ely.by"
	// InjectorAPI is passed to authlib-injector so the game uses Ely.by for
	// sessions and skins.
	InjectorAPI  = "https://authserver.ely.by/api/authlib-injector"
	injectorMeta = "https://authlib-injector.yushi.moe/artifact/latest.json"
)

type Kind string

const (
	Offline Kind = "offline"
	ElyBy   Kind = "elyby"
)

type Account struct {
	ID          string `json:"id"` // stable local id
	Kind        Kind   `json:"kind"`
	Name        string `json:"name"`
	UUID        string `json:"uuid"` // without dashes
	AccessToken string `json:"accessToken,omitempty"`
	ClientToken string `json:"clientToken,omitempty"`
	// ElySkin means an offline nick that also exists on Ely.by, so its skin
	// can be shown through authlib-injector.
	ElySkin bool `json:"elySkin,omitempty"`

	// Set for accounts added through the browser (OAuth2).
	OAuth        bool      `json:"oauth,omitempty"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	ExpiresAt    time.Time `json:"expiresAt,omitempty"`
}

var nickRe = regexp.MustCompile(`^[A-Za-z0-9_]{3,16}$`)

// OfflineUUID mirrors Java's UUID.nameUUIDFromBytes("OfflinePlayer:"+name).
func OfflineUUID(name string) string {
	h := md5.Sum([]byte("OfflinePlayer:" + name))
	h[6] = h[6]&0x0f | 0x30
	h[8] = h[8]&0x3f | 0x80
	return hex.EncodeToString(h[:])
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// NewOffline creates an offline account. If the nick is registered on
// Ely.by its UUID is used so that the skin from Ely.by shows up in game.
func NewOffline(ctx context.Context, name string) (*Account, error) {
	name = strings.TrimSpace(name)
	if !nickRe.MatchString(name) {
		return nil, msg.E("badNick")
	}
	acc := &Account{ID: "offline-" + strings.ToLower(name), Kind: Offline, Name: name, UUID: OfflineUUID(name), AccessToken: randomHex(16)}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var p struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := download.JSON(ctx, elyAuthServer+"/api/users/profiles/minecraft/"+url.PathEscape(name), &p); err == nil && p.ID != "" {
		acc.UUID, acc.Name, acc.ElySkin = p.ID, p.Name, true
	}
	return acc, nil
}

// ErrTwoFactor is returned when the Ely.by account needs a TOTP code.
var ErrTwoFactor = errors.New("two-factor")

type yggProfile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type yggResponse struct {
	AccessToken     string      `json:"accessToken"`
	ClientToken     string      `json:"clientToken"`
	SelectedProfile *yggProfile `json:"selectedProfile"`
	Error           string      `json:"error"`
	ErrorMessage    string      `json:"errorMessage"`
}

func post(ctx context.Context, path string, body any) (*yggResponse, int, error) {
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, elyAuthServer+path, bytes.NewReader(data))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := download.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	var r yggResponse
	if resp.ContentLength != 0 {
		json.NewDecoder(resp.Body).Decode(&r)
	}
	return &r, resp.StatusCode, nil
}

// LoginEly signs in to Ely.by. totp may be empty unless ErrTwoFactor was
// returned before.
func LoginEly(ctx context.Context, login, password, totp string) (*Account, error) {
	if totp != "" {
		password += ":" + strings.TrimSpace(totp)
	}
	clientToken := randomHex(16)
	r, code, err := post(ctx, "/auth/authenticate", map[string]any{
		"username":    strings.TrimSpace(login),
		"password":    password,
		"clientToken": clientToken,
		"requestUser": true,
	})
	if err != nil {
		return nil, msg.Wrap("elyUnavailable", err)
	}
	if code != http.StatusOK {
		if strings.Contains(strings.ToLower(r.ErrorMessage), "two factor") {
			return nil, ErrTwoFactor
		}
		if r.ErrorMessage != "" {
			return nil, translate(r.ErrorMessage)
		}
		return nil, fmt.Errorf("Ely.by: HTTP %d", code)
	}
	if r.SelectedProfile == nil {
		return nil, msg.E("elyNoProfile")
	}
	return &Account{
		ID:          "elyby-" + r.SelectedProfile.ID,
		Kind:        ElyBy,
		Name:        r.SelectedProfile.Name,
		UUID:        r.SelectedProfile.ID,
		AccessToken: r.AccessToken,
		ClientToken: clientToken,
	}, nil
}

func translate(text string) error {
	switch {
	case strings.Contains(text, "Invalid credentials"):
		return msg.E("badCredentials")
	case strings.Contains(text, "banned"):
		return msg.E("banned")
	}
	return errors.New(text)
}

// Refresh makes sure the Ely.by token is valid, refreshing it if needed.
// It returns ok=false when the user has to sign in again.
func Refresh(ctx context.Context, acc *Account) (bool, error) {
	if acc.Kind != ElyBy {
		return true, nil
	}
	if acc.OAuth {
		return refreshOAuth(ctx, acc)
	}
	_, code, err := post(ctx, "/auth/validate", map[string]string{"accessToken": acc.AccessToken})
	if err != nil {
		return false, err
	}
	if code == http.StatusOK || code == http.StatusNoContent {
		return true, nil
	}
	r, code, err := post(ctx, "/auth/refresh", map[string]any{
		"accessToken": acc.AccessToken,
		"clientToken": acc.ClientToken,
		"requestUser": true,
	})
	if err != nil {
		return false, err
	}
	if code != http.StatusOK || r.AccessToken == "" {
		return false, nil
	}
	acc.AccessToken = r.AccessToken
	if r.SelectedProfile != nil {
		acc.Name = r.SelectedProfile.Name
	}
	return true, nil
}

// Invalidate signs the token out on Ely.by (best effort).
func Invalidate(ctx context.Context, acc *Account) {
	if acc.Kind == ElyBy && !acc.OAuth {
		post(ctx, "/auth/invalidate", map[string]string{"accessToken": acc.AccessToken, "clientToken": acc.ClientToken})
	}
}

// EnsureInjector downloads the latest authlib-injector and returns its path.
func EnsureInjector(ctx context.Context) (string, error) {
	path := paths.AuthlibInjector()
	var meta struct {
		DownloadURL string `json:"download_url"`
		Checksums   struct {
			SHA256 string `json:"sha256"`
		} `json:"checksums"`
	}
	if err := download.JSON(ctx, injectorMeta, &meta); err != nil {
		if download.Valid(download.File{Path: path}) {
			return path, nil
		}
		return "", fmt.Errorf("authlib-injector: %w", err)
	}
	err := download.One(ctx, download.File{URL: meta.DownloadURL, Path: path, Hash: meta.Checksums.SHA256})
	return path, err
}
