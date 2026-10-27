package artifacts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

var thumbnailSlots = make(chan struct{}, 3)

// Thumbnail produces bounded 512px previews without exposing originals via
// unauthenticated public paths. DecodeConfig checks dimensions before decoding.
func Thumbnail(w http.ResponseWriter, r *http.Request, root *os.Root, rel, dataDir, projectID string) error {
	f, info, kind, _, err := inspected(root, rel)
	if err != nil {
		return err
	}
	defer f.Close()
	if kind != "image" || info.Size() > maxPreviewBytes {
		return ErrUnsupported
	}
	id := sha256.Sum256([]byte(projectID + "\x00" + rel + "\x00" + info.ModTime().UTC().String() + "\x00" + fmt.Sprint(info.Size())))
	key := hex.EncodeToString(id[:])
	cacheDir := filepath.Join(dataDir, "outputs", "thumbnails")
	cacheFile := filepath.Join(cacheDir, key+".jpg")
	if cached, err := os.Open(cacheFile); err == nil {
		defer cached.Close()
		st, e := cached.Stat()
		if e == nil {
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("Cache-Control", "private, max-age=60")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			http.ServeContent(w, r, key+".jpg", st.ModTime(), cached)
			return nil
		}
	}
	select {
	case thumbnailSlots <- struct{}{}:
		defer func() { <-thumbnailSlots }()
	case <-r.Context().Done():
		return r.Context().Err()
	}
	// Recheck cache after acquiring the decoding semaphore.
	if b, e := os.ReadFile(cacheFile); e == nil {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "private, max-age=60")
		_, _ = w.Write(b)
		return nil
	}
	cfg, format, err := image.DecodeConfig(f)
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > 24000000 {
		return ErrUnsupported
	}
	if format != "png" && format != "jpeg" && format != "gif" && format != "webp" {
		return ErrUnsupported
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	src, _, err := image.Decode(f)
	if err != nil {
		return ErrUnsupported
	}
	width, height := cfg.Width, cfg.Height
	scale := float64(512) / float64(width)
	if float64(height)*scale > 512 {
		scale = 512 / float64(height)
	}
	if scale > 1 {
		scale = 1
	}
	width = int(float64(width) * scale)
	height = int(float64(height) * scale)
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	var output bytes.Buffer
	if err := jpeg.Encode(&output, dst, &jpeg.Options{Quality: 76}); err != nil {
		return err
	}
	b := output.Bytes()
	if err := os.MkdirAll(cacheDir, 0700); err == nil {
		temp, e := os.CreateTemp(cacheDir, "thumb-*.tmp")
		if e == nil {
			_ = temp.Chmod(0600)
			if _, e = temp.Write(b); e == nil {
				_ = temp.Close()
				if e = os.Rename(temp.Name(), cacheFile); e != nil {
					_ = os.Remove(temp.Name())
				}
			} else {
				_ = temp.Close()
				_ = os.Remove(temp.Name())
			}
		}
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Security-Policy", "sandbox")
	w.Header().Set("Cache-Control", "private, max-age=60")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "inline; filename=\"thumbnail.jpg\"")
	if strings.EqualFold(r.Method, http.MethodHead) {
		return nil
	}
	_, err = w.Write(b)
	return err
}
