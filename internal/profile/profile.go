// Package profile serves the profile REST API of the Xbox 360 dashboard: GET /users/xuid(<xuid>)/<resource>
// with "Authorization: XBL2.0 x=<token>" from x360-xsts. Every user starts empty, so each resource answers its
// documented empty list; the console needs HTTP 200 and a body its parse template accepts (no
// continuationToken: its presence means "more pages").
package profile

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/openpak/x360-common/xstoken"
)

// bodies maps a resource path (after /users/xuid(<xuid>)/) to its empty reply.
var bodies = map[string]string{
	"profile/settings": `{"pagingInfo":{"totalRecords":0},"version":"2013-01-01T00:00:00Z","settings":[]}`,
}

var rePath = regexp.MustCompile(`^/users/xuid\(([0-9a-fA-F]{1,16})\)/(.+)$`)

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
	tok, ok := xstoken.FromAuthorization(r.Header.Get("Authorization"))
	if !ok {
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
	if c.Kind != "user" || !strings.EqualFold(strings.TrimLeft(m[1], "0"), strings.TrimLeft(c.XUID, "0")) {
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
