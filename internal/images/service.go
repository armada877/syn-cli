package images

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/armada877/syn-cli/internal/app"
	_ "golang.org/x/image/webp"
)

const MaxBytes int64 = 10 << 20

type Limits struct {
	Bytes   int64
	Edge    int
	Pixels  int64
	Timeout time.Duration
}

type Service struct {
	client http.Client
	cache  string
	limits Limits
	slots  chan struct{}
}

func CachePath(xdg, home string) string {
	if xdg == "" {
		xdg = filepath.Join(home, ".cache")
	}
	return filepath.Join(xdg, "syn", "images")
}

func New(client *http.Client, cache string, limits Limits) *Service {
	if limits.Bytes <= 0 || limits.Bytes > MaxBytes {
		limits.Bytes = MaxBytes
	}
	if limits.Edge <= 0 || limits.Edge > 8192 {
		limits.Edge = 8192
	}
	if limits.Pixels <= 0 || limits.Pixels > 16000000 {
		limits.Pixels = 16000000
	}
	if limits.Timeout <= 0 || limits.Timeout > 10*time.Second {
		limits.Timeout = 10 * time.Second
	}
	c := http.Client{}
	if client != nil {
		c = *client
	}
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || app.ImageURL(req.URL.String(), "") == "" {
			return fmt.Errorf("unsafe image redirect")
		}
		return nil
	}
	return &Service{client: c, cache: cache, limits: limits, slots: make(chan struct{}, 2)}
}

func (s *Service) decode(data []byte, contentType string) (image.Image, error) {
	if int64(len(data)) > s.limits.Bytes {
		return nil, fmt.Errorf("image exceeds byte limit")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("invalid image dimensions: %w", err)
	}
	types := map[string]string{"png": "image/png", "jpeg": "image/jpeg", "gif": "image/gif", "webp": "image/webp"}
	if types[format] == "" || contentType != "" && types[format] != contentType {
		return nil, fmt.Errorf("image format does not match content type")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > s.limits.Edge || cfg.Height > s.limits.Edge || int64(cfg.Width)*int64(cfg.Height) > s.limits.Pixels {
		return nil, fmt.Errorf("image exceeds dimension limit")
	}
	decoded, actual, err := image.Decode(bytes.NewReader(data))
	if err != nil || actual != format {
		return nil, fmt.Errorf("cannot decode image")
	}
	return decoded, nil
}

func (s *Service) cacheFile(source string) string {
	return filepath.Join(s.cache, fmt.Sprintf("%x", sha256.Sum256([]byte(source))))
}

func (s *Service) cached(source string) image.Image {
	if s.cache == "" {
		return nil
	}
	file, err := os.Open(s.cacheFile(source))
	if err != nil {
		return nil
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, s.limits.Bytes+1))
	if err != nil {
		return nil
	}
	decoded, _ := s.decode(data, "")
	return decoded
}

func (s *Service) save(source string, data []byte) {
	if s.cache == "" || os.MkdirAll(s.cache, 0700) != nil {
		return
	}
	file, err := os.CreateTemp(s.cache, ".image-*")
	if err != nil {
		return
	}
	defer os.Remove(file.Name())
	_, err = file.Write(data)
	closeErr := file.Close()
	if err == nil && closeErr == nil {
		_ = os.Rename(file.Name(), s.cacheFile(source))
	}
}

func (s *Service) Load(ctx context.Context, source string) (image.Image, error) {
	if app.ImageURL(source, "") == "" {
		return nil, fmt.Errorf("unsafe image URL")
	}
	ctx, cancel := context.WithTimeout(ctx, s.limits.Timeout)
	defer cancel()
	select {
	case s.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-s.slots }()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cached := s.cached(source); cached != nil {
		return cached, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "image/png, image/jpeg, image/gif, image/webp")
	response, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image request failed")
	}
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || contentType != "image/png" && contentType != "image/jpeg" && contentType != "image/gif" && contentType != "image/webp" {
		return nil, fmt.Errorf("unsupported image content type")
	}
	if response.ContentLength > s.limits.Bytes {
		return nil, fmt.Errorf("image exceeds byte limit")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, s.limits.Bytes+1))
	if err != nil {
		return nil, err
	}
	decoded, err := s.decode(data, contentType)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.save(source, data)
	return decoded, nil
}
