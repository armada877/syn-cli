package images

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func fixture(t *testing.T, format string) []byte {
	t.Helper()
	pixels := image.NewRGBA(image.Rect(0, 0, 2, 2))
	pixels.Set(0, 0, color.RGBA{255, 0, 0, 255})
	var out bytes.Buffer
	var err error
	switch format {
	case "png":
		err = png.Encode(&out, pixels)
	case "jpeg":
		err = jpeg.Encode(&out, pixels, nil)
	case "gif":
		first := image.NewPaletted(pixels.Bounds(), color.Palette{color.Black, color.White})
		second := image.NewPaletted(pixels.Bounds(), first.Palette)
		second.SetColorIndex(0, 0, 1)
		err = gif.EncodeAll(&out, &gif.GIF{Image: []*image.Paletted{first, second}, Delay: []int{1, 1}})
	case "webp":
		data, e := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
		if e != nil {
			t.Fatal(e)
		}
		return data
	}
	if err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func response(data []byte, mime string) *http.Response {
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {mime}}, Body: io.NopCloser(bytes.NewReader(data)), ContentLength: -1}
}

func TestFormatsAndCache(t *testing.T) {
	for _, format := range []string{"png", "jpeg", "gif", "webp"} {
		t.Run(format, func(t *testing.T) {
			data := fixture(t, format)
			var calls int
			cache := t.TempDir()
			service := New(&http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) { calls++; return response(data, "image/"+format), nil })}, cache, Limits{})
			decoded, err := service.Load(context.Background(), "https://example.com/a")
			if err != nil {
				t.Fatal(err)
			}
			if format == "gif" && color.GrayModel.Convert(decoded.At(0, 0)).(color.Gray).Y != 0 {
				t.Fatal("GIF did not use first frame")
			}
			if _, err = service.Load(context.Background(), "https://example.com/a"); err != nil || calls != 1 {
				t.Fatal(calls, err)
			}
			if err := os.WriteFile(service.cacheFile("https://example.com/a"), []byte("broken"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err = service.Load(context.Background(), "https://example.com/a"); err != nil || calls != 2 {
				t.Fatal(calls, err)
			}
			files, _ := os.ReadDir(cache)
			if len(files) != 1 || strings.HasPrefix(files[0].Name(), ".") {
				t.Fatal(files)
			}
		})
	}
}

func TestLimitsAndFailures(t *testing.T) {
	pngData := fixture(t, "png")
	for _, test := range []struct {
		name   string
		data   []byte
		mime   string
		limits Limits
		length int64
	}{
		{"mismatch", pngData, "image/jpeg", Limits{}, -1},
		{"mime", pngData, "text/plain", Limits{}, -1},
		{"decode", []byte("bad"), "image/png", Limits{}, -1},
		{"edge", pngData, "image/png", Limits{Edge: 1}, -1},
		{"pixels", pngData, "image/png", Limits{Pixels: 3}, -1},
		{"bytes", pngData, "image/png", Limits{Bytes: 8}, -1},
		{"length", pngData, "image/png", Limits{Bytes: 8}, 100},
		{"false-length", pngData, "image/png", Limits{Bytes: 8}, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			cache := t.TempDir()
			service := New(&http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
				r := response(test.data, test.mime)
				r.ContentLength = test.length
				return r, nil
			})}, cache, test.limits)
			if _, err := service.Load(context.Background(), "https://example.com/a"); err == nil {
				t.Fatal("accepted invalid image")
			}
			files, _ := os.ReadDir(cache)
			if len(files) != 0 {
				t.Fatal(files)
			}
		})
	}
	service := New(nil, t.TempDir(), Limits{Bytes: 8})
	if err := os.WriteFile(service.cacheFile("https://example.com/a"), make([]byte, 9), 0600); err != nil {
		t.Fatal(err)
	}
	if service.cached("https://example.com/a") != nil {
		t.Fatal("accepted oversized cache")
	}
	for _, source := range []string{"file:///tmp/a", "https://user:pass@example.com/a"} {
		if _, err := service.Load(context.Background(), source); err == nil {
			t.Fatal(source)
		}
	}
	if CachePath("", "/test") != filepath.Join("/test", ".cache", "syn", "images") {
		t.Fatal("cache path")
	}
}

func TestRedirects(t *testing.T) {
	for _, target := range []string{"file:///tmp/a", "https://user:pass@example.com/a", "https://example.com/loop"} {
		service := New(&http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
			r := response(nil, "image/png")
			r.StatusCode = 302
			r.Header.Set("Location", target)
			return r, nil
		})}, t.TempDir(), Limits{})
		if _, err := service.Load(context.Background(), "https://example.com/start"); err == nil {
			t.Fatal(target)
		}
	}
}

func TestConcurrencyCancellationAndTimeout(t *testing.T) {
	var active, peak atomic.Int32
	client := &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
		count := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); count > old; old = peak.Load() {
			if peak.CompareAndSwap(old, count) {
				break
			}
		}
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	service := New(client, t.TempDir(), Limits{Timeout: 30 * time.Millisecond})
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			if _, err := service.Load(context.Background(), "https://example.com/a"); err == nil {
				t.Error("missing timeout")
			}
		})
	}
	group.Wait()
	if peak.Load() > 2 || peak.Load() == 0 {
		t.Fatal(peak.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Load(ctx, "https://example.com/a"); err == nil {
		t.Fatal("missing cancellation")
	}
}

func TestHugeDimensionsAndCorruptPixels(t *testing.T) {
	data := fixture(t, "png")
	for _, size := range [][2]uint32{{9000, 1}, {8192, 8192}, {0, 1}, {1, 0}} {
		header := append([]byte(nil), data...)
		binary.BigEndian.PutUint32(header[16:20], size[0])
		binary.BigEndian.PutUint32(header[20:24], size[1])
		binary.BigEndian.PutUint32(header[29:33], crc32.ChecksumIEEE(header[12:29]))
		service := New(nil, t.TempDir(), Limits{})
		if _, err := service.decode(header, "image/png"); err == nil {
			t.Fatal(size)
		}
	}
	corrupt := append([]byte(nil), data...)
	corrupt[len(corrupt)-8] ^= 1
	if _, err := New(nil, t.TempDir(), Limits{}).decode(corrupt, "image/png"); err == nil {
		t.Fatal("accepted corrupt pixels")
	}
}

func TestCacheFailureAndRedirectSuccess(t *testing.T) {
	data := fixture(t, "png")
	root := t.TempDir()
	badCache := filepath.Join(root, "file")
	if err := os.WriteFile(badCache, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	var calls int
	service := New(&http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Path == "/start" {
			r := response(nil, "image/png")
			r.StatusCode = 302
			r.Header.Set("Location", "/image")
			return r, nil
		}
		return response(data, "image/png"), nil
	})}, badCache, Limits{})
	if _, err := service.Load(context.Background(), "https://example.com/start"); err != nil || calls != 2 {
		t.Fatal(calls, err)
	}
}
