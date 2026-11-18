package dest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sgnd/kongkow/internal/store"
)

// Image limits per network: count per post and bytes per file.
const (
	maxImages         = 4
	mastodonImageSize = 16 << 20
	blueskyImageSize  = 1_000_000
)

var imageTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true}

type image struct {
	Path string
	Type string
	Data []byte
}

// uploadsMedia reports whether d's network gets real image uploads; the
// others list the media paths in the text.
func uploadsMedia(d *store.Dest) bool {
	return d.Kind == store.DestMastodon || d.Kind == store.DestBluesky
}

// CheckMedia says whether every media file of c can go to d's network,
// so nothing is uploaded or sent when one of them cannot.
func CheckMedia(d *store.Dest, c Content) error {
	_, err := loadImages(d, c)
	return err
}

func loadImages(d *store.Dest, c Content) ([]image, error) {
	if !uploadsMedia(d) || len(c.Media) == 0 {
		return nil, nil
	}
	if len(c.Media) > maxImages {
		return nil, fmt.Errorf("%s takes at most %d images; this post has %d", d.Kind, maxImages, len(c.Media))
	}
	limit := mastodonImageSize
	if d.Kind == store.DestBluesky {
		limit = blueskyImageSize
	}
	var imgs []image
	for _, p := range c.Media {
		fi, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("media %s: %w", p, err)
		}
		if fi.Size() > int64(limit) {
			return nil, fmt.Errorf("media %s is %d bytes; %s takes at most %d", p, fi.Size(), d.Kind, limit)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("media %s: %w", p, err)
		}
		typ := http.DetectContentType(data)
		if !imageTypes[typ] {
			return nil, fmt.Errorf("media %s is %s; only JPEG, PNG, GIF and WebP images can be uploaded", p, typ)
		}
		imgs = append(imgs, image{Path: p, Type: typ, Data: data})
	}
	return imgs, nil
}

// mastodonUpload sends one image and returns its media id.
func mastodonUpload(ctx context.Context, base, tok string, img image) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", filepath.Base(img.Path))
	if err != nil {
		return "", err
	}
	if _, err := part.Write(img.Data); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	var res struct {
		ID string `json:"id"`
	}
	if err := rawCall(ctx, http.MethodPost, base+"/api/v2/media", tok, w.FormDataContentType(), &buf, &res); err != nil {
		return "", fmt.Errorf("upload %s: %w", img.Path, err)
	}
	return res.ID, nil
}

// bskyUpload sends one image and returns the blob reference to embed.
func bskyUpload(ctx context.Context, jwt string, img image) (json.RawMessage, error) {
	var res struct {
		Blob json.RawMessage `json:"blob"`
	}
	if err := rawCall(ctx, http.MethodPost, bskyService+"/xrpc/com.atproto.repo.uploadBlob", jwt, img.Type, bytes.NewReader(img.Data), &res); err != nil {
		return nil, fmt.Errorf("upload %s: %w", img.Path, err)
	}
	return res.Blob, nil
}
