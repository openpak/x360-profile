package profile

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/openpak/x360-common/accountcore"
	"github.com/openpak/x360-common/xstoken"
)

type avatarFixture struct {
	avatar  *accountcore.Avatar
	err     error
	account string
}

func (f *avatarFixture) GetAvatar(_ context.Context, account string, size uint32, version string) (*accountcore.Avatar, error) {
	f.account = account
	if size != 256 {
		return nil, fmt.Errorf("wrong requested size")
	}
	return f.avatar, f.err
}

func TestAccountAvatarRecoveryAndPublicPictures(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			if x < 128 {
				src.SetRGBA(x, y, color.RGBA{255, 0, 0, 255})
			} else {
				src.SetRGBA(x, y, color.RGBA{0, 0, 255, 255})
			}
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, src)
	core := &avatarFixture{avatar: &accountcore.Avatar{PNG: buf.Bytes(), Version: "v1", UpdatedAtUnix: 1700000000}}
	signer := &xstoken.Signer{Key: []byte("test-token-signing-key")}
	store := &PictureStore{Dir: t.TempDir(), Key: signer.Key}
	h := &Handler{Signer: signer, Core: core, Pictures: store}
	request := func(path, auth string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Authorization", auth)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	token := signer.Sign(xstoken.Claims{Kind: "user", XUID: "0009005f5139dadb", Account: "account-A", Expires: time.Now().Add(time.Hour).Unix()})
	w := request("/users/xuid(2533684175035099)/profile/settings", "XBL2.0 x="+token)
	if w.Code != 200 || core.account != "account-A" {
		t.Fatalf("settings: %d %s", w.Code, w.Body.String())
	}
	var body struct {
		Settings []struct {
			Value string `json:"value"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(body.Settings[0].Value)
	if err != nil {
		t.Fatal(err)
	}
	var key strings.Builder
	for i := 0; i < len(raw)-2; i += 2 {
		key.WriteRune(rune(binary.LittleEndian.Uint16(raw[i:])))
	}
	k := key.String()
	title, _ := strconv.ParseUint(k[:8], 16, 32)
	// Recreate the store to prove handles survive a service restart.
	h.Pictures = &PictureStore{Dir: store.Dir, Key: store.Key}
	for i, size := range []int{64, 32} {
		tile, _ := strconv.ParseUint(k[8+i*8:16+i*8], 16, 32)
		path := fmt.Sprintf("/global/t.%x/tile/0/%x", title, tile)
		picture := request(path, "")
		if picture.Code != 200 {
			t.Fatalf("picture: %d %s", picture.Code, picture.Body.String())
		}
		img, err := png.Decode(picture.Body)
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Dx() != size || img.Bounds().Dy() != size {
			t.Fatalf("wrong dimensions %v", img.Bounds())
		}
		r, g, b, _ := img.At(0, 0).RGBA()
		if r != 65535 || g != 0 || b != 0 {
			t.Fatal("account avatar pixels lost")
		}
		r, g, b, _ = img.At(size-1, 0).RGBA()
		if r != 0 || g != 0 || b != 65535 {
			t.Fatal("wrong avatar region")
		}
		core.err = &accountcore.Error{Code: accountcore.CodeNotFound}
		if got := request(path, "").Code; got != 404 {
			t.Fatalf("deleted avatar: %d", got)
		}
		core.err = nil
	}
	if got := request("/users/xuid(2533684175035100)/profile/settings", "XBL2.0 x="+token).Code; got != 403 {
		t.Fatalf("cross-account settings: %d", got)
	}
	core.err = errors.New("core unreachable")
	if got := request("/users/xuid(2533684175035099)/profile/settings", "XBL2.0 x="+token).Code; got != 503 {
		t.Fatalf("temporary error silently replaced avatar: %d", got)
	}
}

func TestAvatarVersionChangesPictureKey(t *testing.T) {
	s := &PictureStore{Dir: t.TempDir(), Key: []byte("key")}
	first, err := s.handles("account", "version1")
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.handles("account", "version1")
	if err != nil {
		t.Fatal(err)
	}
	next, err := s.handles("account", "version2")
	if err != nil {
		t.Fatal(err)
	}
	if first != again || first == next {
		t.Fatal("unstable or stale picture handle")
	}
}
