package profile

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/openpak/x360-common/accountcore"
)

// AvatarSource reads the existing account avatar without modifying it.
type AvatarSource interface {
	GetAvatar(context.Context, string, uint32, string) (*accountcore.Avatar, error)
}

type pictureRef struct {
	Account string
	Version string
	Size    int
}

// PictureStore keeps opaque native picture handles across service restarts.
// Avatar pixels remain in the account core and are fetched when requested.
type PictureStore struct {
	Dir string
	Key []byte
	mu  sync.Mutex
}

func (s *PictureStore) handles(account, version string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.Dir, 0700); err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, s.Key)
	fmt.Fprintf(mac, "x360-picture\x00%s\x00%s", account, version)
	sum := mac.Sum(nil)
	title := uint32(0x4f500000) | uint32(binary.BigEndian.Uint16(sum))
	big, small := binary.BigEndian.Uint32(sum[2:6]), binary.BigEndian.Uint32(sum[6:10])
	if big == small {
		small ^= 1
	}
	for _, entry := range []struct {
		id   uint32
		size int
	}{{big, 64}, {small, 32}} {
		name := fmt.Sprintf("%x-%x.json", title, entry.id)
		ref := pictureRef{account, version, entry.size}
		path := filepath.Join(s.Dir, name)
		old, err := os.ReadFile(path)
		if err == nil {
			var existing pictureRef
			if json.Unmarshal(old, &existing) != nil || existing != ref {
				return "", fmt.Errorf("picture handle collision")
			}
			continue
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		data, _ := json.Marshal(ref)
		f, err := os.CreateTemp(s.Dir, "handle-")
		if err != nil {
			return "", err
		}
		temp := f.Name()
		_, err = f.Write(data)
		if err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Rename(temp, path)
		}
		os.Remove(temp)
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%08x%08x%08x", title, big, small), nil
}

var picturePath = regexp.MustCompile(`^/global/t\.([0-9a-f]{1,8})/tile/0/([0-9a-f]{1,8})$`)

func (h *Handler) accountSettings(ctx context.Context, account string) (string, error) {
	if h.Core == nil {
		return defaultSettingsBody(), nil
	}
	avatar, err := h.Core.GetAvatar(ctx, account, 256, "")
	if accountcore.IsNotFound(err) {
		return defaultSettingsBody(), nil
	}
	if err != nil {
		return "", err
	}
	if h.Pictures == nil || avatar.Version == "" {
		return "", fmt.Errorf("picture store or avatar version unavailable")
	}
	key, err := h.Pictures.handles(account, avatar.Version)
	if err != nil {
		return "", err
	}
	version := time.Unix(avatar.UpdatedAtUnix, 0).UTC().Format(time.RFC3339)
	return settingsBody(key, version), nil
}

func (h *Handler) servePicture(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", 405)
		return
	}
	match := picturePath.FindStringSubmatch(r.URL.Path)
	if match == nil || h.Core == nil || h.Pictures == nil {
		http.NotFound(w, r)
		return
	}
	title, _ := strconv.ParseUint(match[1], 16, 32)
	tile, _ := strconv.ParseUint(match[2], 16, 32)
	data, err := os.ReadFile(filepath.Join(h.Pictures.Dir, fmt.Sprintf("%x-%x.json", title, tile)))
	var ref pictureRef
	if err != nil || json.Unmarshal(data, &ref) != nil || (ref.Size != 32 && ref.Size != 64) {
		http.NotFound(w, r)
		return
	}
	avatar, err := h.Core.GetAvatar(r.Context(), ref.Account, 256, "")
	if accountcore.IsNotFound(err) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.log().Error("avatar unavailable", "err", err)
		http.Error(w, "avatar unavailable", 503)
		return
	}
	// Old handles must not cache a new avatar under an earlier content version.
	if avatar.Version != ref.Version {
		http.NotFound(w, r)
		return
	}
	body, err := resizeAvatar(avatar.PNG, ref.Size)
	if err != nil {
		h.log().Error("avatar decode", "err", err)
		http.Error(w, "avatar unavailable", 502)
		return
	}
	h.log().Info("picture", "path", r.URL.Path, "size", ref.Size, "account", ref.Account)
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.Header().Set("Cache-Control", "public, max-age=300")
	if r.Method == http.MethodGet {
		w.Write(body)
	}
}

// Area averaging preserves the uploaded picture at the console's two sizes.
func resizeAvatar(data []byte, size int) ([]byte, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if cfg.Width != 256 || cfg.Height != 256 || (size != 32 && size != 64) {
		return nil, fmt.Errorf("invalid avatar dimensions")
	}
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	step := 256 / size
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var red, green, blue, alpha uint32
			for j := 0; j < step; j++ {
				for i := 0; i < step; i++ {
					r, g, b, a := src.At(x*step+i, y*step+j).RGBA()
					red += r
					green += g
					blue += b
					alpha += a
				}
			}
			n := uint32(step * step)
			dst.SetRGBA(x, y, color.RGBA{uint8((red / n) >> 8), uint8((green / n) >> 8), uint8((blue / n) >> 8), uint8((alpha / n) >> 8)})
		}
	}
	var out bytes.Buffer
	err = png.Encode(&out, dst)
	return out.Bytes(), err
}
