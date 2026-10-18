package profile

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openpak/x360-common/xstoken"
)

func TestEmptyReplies(t *testing.T) {
	s := &xstoken.Signer{Key: []byte("k")}
	h := &Handler{Signer: s}
	tok := s.Sign(xstoken.Claims{Kind: "user", XUID: "0009005f5139dadb", Account: "a", Expires: time.Now().Add(time.Hour).Unix()})
	get := func(path, auth string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		if auth != "" {
			r.Header.Set("Authorization", auth)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec
	}
	for res := range bodies {
		rec := get("/users/xuid(0009005f5139dadb)/"+res, "XBL2.0 x="+tok)
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("%s: %d", res, rec.Code)
		}
	}
	if rec := get("/users/xuid(0009005f5139dadb)/profile/settings", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: %d", rec.Code)
	}
	if rec := get("/users/xuid(0009005f5139dadc)/profile/settings", "XBL2.0 x="+tok); rec.Code != http.StatusForbidden {
		t.Fatalf("other user: %d", rec.Code)
	}
	if rec := get("/users/xuid(0009005f5139dadb)/nothing", "XBL2.0 x="+tok); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown resource: %d", rec.Code)
	}
}
