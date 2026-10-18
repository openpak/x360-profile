// Package xstoken is the token the Xbox 360 token service (x360-xsts) issues and the
// REST services (x360-profile, x360-progress) verify. The console never looks inside:
// it copies the bytes between <trust:RequestedSecurityToken> tags into
// "Authorization: XBL2.0 x=<token>". A token is base64url(json claims) "." base64url(hmac-sha256).
package xstoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Claims is what a token carries.
type Claims struct {
	Kind     string `json:"k"`   // "user" or "device"
	XUID     string `json:"x"`   // 16 hex digits: the user's online XUID, or the device id
	Account  string `json:"a"`   // OpenPak account id (users only)
	Audience string `json:"aud"` // the AppliesTo address the console asked for
	Expires  int64  `json:"exp"` // unix seconds
}

// Signer signs and verifies tokens with one HMAC-SHA256 key.
type Signer struct {
	Key []byte
	Now func() time.Time // default time.Now
}

func (s *Signer) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Signer) mac(b string) []byte {
	m := hmac.New(sha256.New, s.Key)
	m.Write([]byte(b))
	return m.Sum(nil)
}

// Sign encodes c.
func (s *Signer) Sign(c Claims) string {
	body, _ := json.Marshal(c)
	b := base64.RawURLEncoding.EncodeToString(body)
	return b + "." + base64.RawURLEncoding.EncodeToString(s.mac(b))
}

// Verify checks signature and expiry and returns the claims.
func (s *Signer) Verify(tok string) (Claims, error) {
	var c Claims
	b, sig, ok := strings.Cut(tok, ".")
	if !ok {
		return c, errors.New("malformed token")
	}
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(got, s.mac(b)) {
		return c, errors.New("bad signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(b)
	if err != nil || json.Unmarshal(raw, &c) != nil {
		return c, errors.New("malformed claims")
	}
	if s.now().Unix() >= c.Expires {
		return c, errors.New("expired")
	}
	return c, nil
}

// FromAuthorization extracts the token from an "XBL2.0 x=<token>" header value
// (an optional "uhs;" prefix is ignored).
func FromAuthorization(h string) (string, bool) {
	const p = "XBL2.0 x="
	if !strings.HasPrefix(h, p) {
		return "", false
	}
	t := h[len(p):]
	if i := strings.LastIndex(t, ";"); i >= 0 {
		t = t[i+1:]
	}
	return t, t != ""
}
