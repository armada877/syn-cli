package tui

import (
	"context"
	"fmt"
	"image"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/armada877/syn-cli/internal/config"
	"github.com/armada877/syn-cli/internal/domain"
	"github.com/armada877/syn-cli/internal/images"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi/kitty"
)

func imageModel(t *testing.T, content string) *Model {
	t.Helper()
	capability := false
	cfg := config.Default()
	m := NewWithOptions(context.Background(), nil, Options{ImageConfig: &cfg, KittyCapability: &capability, ImageService: images.New(nil, t.TempDir(), images.Limits{})})
	m.width, m.height, m.loading = 80, 20, false
	m.reader = &domain.Article{ID: 1, URL: "https://example.com/article", Title: "Title", Content: content}
	return m
}

func TestImagePositionsAreStructural(t *testing.T) {
	for _, wide := range []bool{false, true} {
		m := imageModel(t, `<p>[image: same]</p><img src="/lead" alt="same"><p>`+strings.Repeat("paragraph<br>", 40)+`</p><img src="/later" alt="same">`)
		if wide {
			m.width = 120
		}
		var lines []string
		var spans []imageSpan
		if wide {
			lines, spans = m.previewLayout()
		} else {
			lines, spans = m.readerLayout()
		}
		if len(spans) != 2 || spans[0].block.Source != "https://example.com/lead" || spans[1].block.Source != "https://example.com/later" || spans[1].start <= spans[0].start {
			t.Fatal(spans)
		}
		m.syncImages()
		if m.imageState.entries["https://example.com/lead"] == nil || m.imageState.entries["https://example.com/later"] != nil {
			t.Fatal("requested hidden image")
		}
		m.scroll = spans[1].start
		m.syncImages()
		if m.imageState.entries["https://example.com/later"] == nil {
			t.Fatal("missed later image")
		}
		if strings.Count(strings.Join(lines, "\n"), "[image: same]") != 3 {
			t.Fatal(lines)
		}
	}
}

func TestCanceledResultCannotReplaceNewRequest(t *testing.T) {
	m := imageModel(t, `<img src="/a"><p>`+strings.Repeat("paragraph<br>", 40)+`</p>`)
	m.syncImages()
	old := m.imageState.entries["https://example.com/a"].token
	generation := m.imageState.generation
	m.scroll = 50
	m.syncImages()
	m.scroll = 0
	m.syncImages()
	entry := m.imageState.entries["https://example.com/a"]
	if entry == nil || entry.token == old {
		t.Fatal("request token reused")
	}
	m.Update(imageResult{token: old, generation: generation, source: "https://example.com/a", err: fmt.Errorf("canceled")})
	if entry.failed || entry.pixels != nil || entry.cancel == nil {
		t.Fatal("old result changed request")
	}
	m.Update(imageResult{token: old, generation: generation, source: "https://example.com/a", pixels: image.NewRGBA(image.Rect(0, 0, 2, 2)), width: 2, height: 2})
	if entry.pixels != nil {
		t.Fatal("old success replaced request")
	}
	m.reader = &domain.Article{ID: 2}
	m.syncImages()
	m.Update(imageResult{token: entry.token, generation: generation, source: "https://example.com/a", pixels: image.NewRGBA(image.Rect(0, 0, 2, 2))})
	if len(m.imageState.entries) != 0 {
		t.Fatal("old article result retained")
	}
}

func TestFailuresDoNotBlockLaterImages(t *testing.T) {
	content := ""
	for i := range 12 {
		content += fmt.Sprintf(`<img src="/%d"><p>Text</p>`, i)
	}
	m := imageModel(t, content)
	m.height = 100
	for range 4 {
		m.syncImages()
		for source, entry := range m.imageState.entries {
			m.Update(imageResult{token: entry.token, generation: m.imageState.generation, source: source, err: fmt.Errorf("failed")})
		}
	}
	if len(m.imageState.failed) != 12 {
		t.Fatal(len(m.imageState.failed))
	}
	if len(m.imageState.entries) != 0 {
		t.Fatal("failed entries fill memory")
	}
	if cmd := m.syncImages(); cmd != nil {
		t.Fatal("failed requests repeated")
	}
}

func TestKittyDetectionAndFallback(t *testing.T) {
	for _, test := range []struct {
		name      string
		event     tea.Msg
		supported bool
		pending   bool
	}{
		{"match", uv.KittyGraphicsEvent{Options: kitty.Options{ID: 42}, Payload: []byte("OK")}, true, false},
		{"unrelated", uv.KittyGraphicsEvent{Options: kitty.Options{ID: 43}, Payload: []byte("OK")}, false, true},
		{"negative", uv.KittyGraphicsEvent{Options: kitty.Options{ID: 42}, Payload: []byte("EINVAL")}, false, false},
		{"malformed", uv.KittyGraphicsEvent{Options: kitty.Options{ID: 0}, Payload: []byte("OK")}, false, true},
		{"timeout", imageTimeout{42}, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := imageModel(t, "")
			m.imageState.pending = true
			m.imageState.query = 42
			m.Update(test.event)
			if m.imageState.kitty != test.supported || m.imageState.pending != test.pending {
				t.Fatal(m.imageState)
			}
		})
	}
	m := imageModel(t, `<img src="/a">`)
	m.imageState.pending = true
	m.imageState.query = 42
	m.Update(imageTimeout{42})
	m.Update(uv.KittyGraphicsEvent{Options: kitty.Options{ID: 42}, Payload: []byte("OK")})
	if m.imageState.kitty {
		t.Fatal("accepted late response")
	}
	for _, mode := range []string{"off", "placeholder", "no-color"} {
		m := imageModel(t, `<img src="/a">`)
		switch mode {
		case "off":
			m.imageState.config.Images = "off"
		case "placeholder":
			m.imageState.config.ImageFallback = "placeholder"
		case "no-color":
			m.noColor = true
		}
		if m.syncImages() != nil || len(m.imageState.entries) != 0 {
			t.Fatal(mode)
		}
	}
	m = imageModel(t, `<img src="/a">`)
	m.reader = nil
	if m.syncImages() != nil {
		t.Fatal("list started request")
	}
}

func TestUploadBeforePlaceholderAndResize(t *testing.T) {
	m := imageModel(t, `<p>Before</p><img src="/a"><p>After</p>`)
	m.imageState.kitty = true
	m.syncImages()
	entry := m.imageState.entries["https://example.com/a"]
	_, render := m.Update(imageResult{token: entry.token, generation: m.imageState.generation, source: "https://example.com/a", pixels: image.NewRGBA(image.Rect(0, 0, 10, 10)), width: 10, height: 10})
	applyImageRender(m, render)
	if strings.Contains(m.View().Content, string(kitty.Placeholder)) {
		t.Fatal("placeholder preceded upload")
	}
	if !m.imageState.writing {
		t.Fatal("upload not queued")
	}
	m.Update(protocolDone{&imageUploaded{m.imageState.generation, "https://example.com/a", entry.id}})
	if !strings.Contains(m.View().Content, string(kitty.Placeholder)) {
		t.Fatal("missing uploaded placeholder")
	}
	old := entry.id
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	if entry.cancel == nil {
		t.Fatal("resize did not reload verified pixels")
	}
	m.Update(imageResult{token: entry.token, generation: m.imageState.generation, source: "https://example.com/a", pixels: image.NewRGBA(image.Rect(0, 0, 10, 10)), width: 10, height: 10})
	if entry.id == old || entry.columns != 36 || entry.ready {
		t.Fatal("placement not resized")
	}
	m.Update(tea.KeyPressMsg{Code: 'q'})
	if len(m.imageState.entries) != 0 || !m.imageState.quitting {
		t.Fatal("quit retained resources")
	}
}

func applyImageRender(m *Model, command tea.Cmd) {
	if command == nil {
		return
	}
	switch message := command().(type) {
	case tea.BatchMsg:
		for _, next := range message {
			applyImageRender(m, next)
		}
	case imageRendered:
		m.Update(message)
	}
}

func TestObsoleteRenderAndQueuedUpload(t *testing.T) {
	m := imageModel(t, `<img src="/a">`)
	m.imageState.kitty = true
	m.syncImages()
	entry := m.imageState.entries["https://example.com/a"]
	_, render := m.Update(imageResult{token: entry.token, generation: m.imageState.generation, source: "https://example.com/a", pixels: image.NewRGBA(image.Rect(0, 0, 2, 2)), width: 2, height: 2})
	previousToken := entry.renderToken
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	applyImageRender(m, render)
	if len(entry.lines) != 0 || previousToken == entry.renderToken {
		t.Fatal("accepted obsolete render")
	}
	generation := m.imageState.generation
	m.releaseImages()
	m.Update(imageRendered{generation: generation, source: "https://example.com/a", token: entry.renderToken, output: "unsafe stale upload"})
	if len(m.imageState.queue) != 0 {
		t.Fatal("queued obsolete upload")
	}
	m.queueProtocol("upload", &imageUploaded{generation, "source", 42})
	if !m.discardUpload(42) || len(m.imageState.queue) != 0 {
		t.Fatal("kept obsolete queued upload")
	}
}

func TestMoreThanEightSuccessfulVisibleImages(t *testing.T) {
	content := ""
	for index := range 16 {
		content += fmt.Sprintf(`<img src="/%d" alt="same">`, index)
	}
	m := imageModel(t, content)
	m.height = 80
	m.syncImages()
	for range 4 {
		sources := []string{}
		for source, entry := range m.imageState.entries {
			if entry.cancel != nil {
				sources = append(sources, source)
			}
		}
		for _, source := range sources {
			entry := m.imageState.entries[source]
			_, render := m.Update(imageResult{token: entry.token, generation: m.imageState.generation, source: source, pixels: image.NewRGBA(image.Rect(0, 0, 100, 1)), width: 100, height: 1})
			applyImageRender(m, render)
		}
	}
	if len(m.imageState.entries) != 16 {
		t.Fatal("visible images remained unloaded", len(m.imageState.entries))
	}
	for source, entry := range m.imageState.entries {
		if !entry.ready || entry.cancel != nil || entry.rendering || entry.pixels != nil || len(entry.lines) != 1 {
			t.Fatal(source, entry)
		}
	}
	if m.syncImages() != nil {
		t.Fatal("completed images started new work")
	}
}

func TestKnownDimensionsSurviveViewportExit(t *testing.T) {
	m := imageModel(t, `<img src="/a"><p>`+strings.Repeat("paragraph<br>", 80)+`</p>`)
	m.syncImages()
	entry := m.imageState.entries["https://example.com/a"]
	_, render := m.Update(imageResult{token: entry.token, generation: m.imageState.generation, source: "https://example.com/a", pixels: image.NewRGBA(image.Rect(0, 0, 100, 100)), width: 100, height: 100})
	applyImageRender(m, render)
	_, spans := m.readerLayout()
	rows := spans[0].count
	m.scroll = 100
	m.syncImages()
	if len(m.imageState.entries) != 0 {
		t.Fatal("offscreen image retained")
	}
	m.scroll = 0
	m.syncImages()
	_, spans = m.readerLayout()
	if spans[0].count != rows {
		t.Fatal("reserved rows changed", rows, spans[0].count)
	}
}

func TestGraphicsKeepsIdentifierColors(t *testing.T) {
	m := imageModel(t, "")
	m.imageState.pending, m.imageState.query = true, 42
	_, command := m.Update(uv.KittyGraphicsEvent{Options: kitty.Options{ID: 42}, Payload: []byte("OK")})
	if _, ok := command().(tea.ColorProfileMsg); !ok {
		t.Fatal("missing true-color profile")
	}
}

func TestImageSyncWithoutService(t *testing.T) {
	for _, mode := range []string{"reader", "help", "picker", "input", "confirm"} {
		t.Run(mode, func(t *testing.T) {
			m := imageModel(t, `<img src="/a">`)
			m.imageState.service = nil
			m.imageState.article = 99
			m.imageState.generation = 7
			switch mode {
			case "help":
				m.help = true
			case "picker":
				m.picker = true
			case "input":
				m.input = true
			case "confirm":
				m.confirm = true
			}
			before := m.imageState
			if command := m.syncImages(); command != nil {
				t.Fatal("disabled service returned image work")
			}
			if !reflect.DeepEqual(before, m.imageState) {
				t.Fatal("disabled service changed image state")
			}
		})
	}
}

func TestImageModalCancelsActiveRequest(t *testing.T) {
	for _, mode := range []string{"help", "picker", "input", "confirm"} {
		t.Run(mode, func(t *testing.T) {
			m := imageModel(t, `<img src="/a">`)
			m.syncImages()
			entry := m.imageState.entries["https://example.com/a"]
			cancel := entry.cancel
			canceled := false
			entry.cancel = func() { canceled = true; cancel() }
			entry.id = 42
			generation := m.imageState.generation
			switch mode {
			case "help":
				m.help = true
			case "picker":
				m.picker = true
			case "input":
				m.input = true
			case "confirm":
				m.confirm = true
			}
			m.syncImages()
			if !canceled || len(m.imageState.entries) != 0 || m.imageState.generation != generation+1 {
				t.Fatal("modal retained active image request")
			}
			if len(m.imageState.queue) != 1 || m.imageState.queue[0].output != images.Delete(42) {
				t.Fatal("modal did not release terminal image")
			}
		})
	}
}
