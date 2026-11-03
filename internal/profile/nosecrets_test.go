package profile

import (
	"bytes"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/openpak/x360-common/xstoken"
)

func TestLogCarriesNoSecrets(t *testing.T) {
	var logs bytes.Buffer
	s := &xstoken.Signer{Key: []byte("k")}
	h := &Handler{Signer: s, Log: slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))}
	tok := s.Sign(xstoken.Claims{Kind: "user", XUID: "0009005f5139dadb", Account: "a", Expires: time.Now().Add(time.Hour).Unix()})
	r := httptest.NewRequest("GET", "/users/xuid(0009005f5139dadb)/profile/settings?token=QUERYSECRET", strings.NewReader("BODYSECRET"))
	r.Header.Set("Authorization", "XBL2.0 x="+tok)
	r.Header.Set("Cookie", "COOKIESECRET")
	r.Header.Set("User-Agent", "UASECRET")
	h.ServeHTTP(httptest.NewRecorder(), r)
	out := logs.String()
	if !strings.Contains(out, "request") {
		t.Fatalf("request line missing: %q", out)
	}
	for _, v := range []string{"QUERYSECRET", "BODYSECRET", tok, "COOKIESECRET", "UASECRET", "token="} {
		if strings.Contains(out, v) {
			t.Fatalf("%q leaked: %s", v, out)
		}
	}
}
