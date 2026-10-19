package profile

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
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
		rec := get("/users/xuid(2533684175035099)/"+res, "XBL2.0 x="+tok)
		if rec.Code != http.StatusOK {
			t.Fatalf("decimal XUID %s: %d", res, rec.Code)
		}
		rec = get("/users/xuid(0009005f5139dadb)/"+res, "XBL2.0 x="+tok)
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

// The firmware decodes base64, byte-swaps UTF-16LE, and parses three hex words.
func TestDefaultPictureSetting(t *testing.T) {
	var body struct {
		Paging struct {
			Total int `json:"totalRecords"`
		} `json:"pagingInfo"`
		Settings []struct {
			TitleID uint32 `json:"titleId"`
			ID      uint32 `json:"id"`
			Value   string `json:"value"`
		} `json:"settings"`
	}
	if err := json.Unmarshal([]byte(defaultSettingsBody()), &body); err != nil {
		t.Fatal(err)
	}
	if body.Paging.Total != 1 || len(body.Settings) != 1 {
		t.Fatalf("unexpected settings: %+v", body)
	}
	setting := body.Settings[0]
	if setting.TitleID != 0xFFFE07D1 || setting.ID != 0x4064000F {
		t.Fatalf("wrong setting: %+v", setting)
	}
	value, err := base64.StdEncoding.DecodeString(setting.Value)
	if err != nil {
		t.Fatal(err)
	}
	if len(value) != 50 || binary.LittleEndian.Uint16(value[48:]) != 0 {
		t.Fatalf("invalid terminated UTF-16: %x", value)
	}
	var key string
	for i := 0; i < 48; i += 2 {
		key += string(rune(binary.LittleEndian.Uint16(value[i:])))
	}
	if key != "fffe07d10002000100010001" {
		t.Fatalf("wrong picture key: %q", key)
	}
}
