// Package profile serves the profile REST API of the Xbox 360 dashboard: GET /users/xuid(<xuid>)/<resource>
// with "Authorization: XBL2.0 x=<token>" from x360-xsts. New users receive a built-in
// default gamer picture; the console needs HTTP 200 and an accepted response body (no
// continuationToken: its presence means "more pages").
package profile

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/openpak/x360-common/xstoken"
)

// bodies maps a resource path (after /users/xuid(<xuid>)/) to its response body.
var bodies = map[string]string{
	"profile/settings": defaultSettingsBody(),
}

// New accounts use the first picture already packaged with the dashboard.
// Xenon recovery restores pictures from settings and skips the new-account
// default-picture step. The setting value is UTF-16LE including its NUL.
const defaultPictureKey = "fffe07d10002000100010001"

func defaultSettingsBody() string {
	value := make([]byte, (len(defaultPictureKey)+1)*2)
	for i := range defaultPictureKey {
		value[2*i] = defaultPictureKey[i]
	}
	return fmt.Sprintf(`{"pagingInfo":{"totalRecords":1},"version":"2013-01-01T00:00:00Z","settings":[{"titleId":4294838225,"id":1080295439,"value":"%s"}]}`, base64.StdEncoding.EncodeToString(value))
}

var rePath = regexp.MustCompile(`^/users/xuid\(([0-9a-fA-F]{1,20})\)/(.+)$`)

// Handler serves the profile API.
type Handler struct {
	Signer *xstoken.Signer
	Log    *slog.Logger
}

func (h *Handler) log() *slog.Logger {
	if h.Log != nil {
		return h.Log
	}
	return slog.Default()
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := rePath.FindStringSubmatch(r.URL.Path)
	if m == nil || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	body, ok := bodies[strings.ToLower(m[2])]
	if !ok {
		http.NotFound(w, r)
		return
	}
	auth := r.Header.Get("Authorization")
	tok, ok := xstoken.FromAuthorization(auth)
	if !ok {
		// Log the scheme, never the token: the console's header format is what matters here.
		h.log().Info("token required", "path", r.URL.Path, "host", r.Host, "auth_len", len(auth), "auth_head", auth[:min(len(auth), 12)])
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}
	c, err := h.Signer.Verify(tok)
	if err != nil {
		h.log().Info("token refused", "err", err.Error(), "path", r.URL.Path)
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	// A user token reads only its own data (the xuid in the path is the user's online XUID).
	if c.Kind != "user" || !c.MatchesXUID(m[1]) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	h.log().Info("request", "path", r.URL.Path, "query", r.URL.RawQuery, "xuid", c.XUID, "account", c.Account)
	if !json.Valid([]byte(body)) {
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(body))
}
