package tui

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"image"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/armada877/syn-cli/internal/app"
	"github.com/armada877/syn-cli/internal/config"
	"github.com/armada877/syn-cli/internal/domain"
	"github.com/armada877/syn-cli/internal/images"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
)

type imageEntry struct {
	token                        int
	renderToken                  int
	cancel                       context.CancelFunc
	pixels                       image.Image
	lines                        []string
	width, height, columns, rows int
	id                           int
	ready                        bool
	rendering                    bool
	failed                       bool
}

type protocolItem struct {
	output   string
	uploaded *imageUploaded
}
type protocolDone struct{ uploaded *imageUploaded }

type imageState struct {
	queue      []protocolItem
	writing    bool
	quitting   bool
	config     config.Config
	service    *images.Service
	entries    map[string]*imageEntry
	failed     map[string]bool
	dimensions map[string][2]int
	article    int64
	generation uint64
	query      int
	pending    bool
	deadline   time.Time
	kitty      bool
	cellAspect float64
	status     string
}

type imageResult struct {
	token         int
	generation    uint64
	source        string
	pixels        image.Image
	width, height int
	err           error
}
type imageRendered struct {
	generation uint64
	source     string
	token      int
	lines      []string
	output     string
	err        error
}
type imageUploaded struct {
	generation uint64
	source     string
	id         int
}
type imageTimeout struct{ id int }

func graphicsID() int {
	var data [4]byte
	_, _ = rand.Read(data[:])
	return int(binary.LittleEndian.Uint32(data[:])&0x7fffffff) + 1
}

func (m *Model) configureImages(options Options) {
	cfg, status := config.Default(), ""
	if options.ImageConfig != nil {
		cfg = *options.ImageConfig
	} else {
		cfg, status = config.Load(options.ConfigPath)
	}
	cfg, warning := config.ApplyImageOverride(cfg, options.ImageOverride)
	if warning != "" {
		if status != "" {
			status += "; "
		}
		status += warning
	}
	m.imageState = imageState{config: cfg, service: options.ImageService, entries: map[string]*imageEntry{}, failed: map[string]bool{}, dimensions: map[string][2]int{}, query: graphicsID(), cellAspect: 0.5, status: status}
	if m.noColor {
		return
	}
	switch cfg.Images {
	case "kitty":
		m.imageState.kitty = true
	case "auto":
		if options.KittyCapability != nil {
			m.imageState.kitty = *options.KittyCapability
		} else {
			m.imageState.pending = options.ImageService != nil
		}
	}
}

func (m *Model) initImages() tea.Cmd {
	if !m.imageState.pending {
		if m.imageState.kitty && m.imageState.config.Images != "off" && !m.noColor {
			if m.imageState.config.Images == "kitty" {
				return tea.Batch(graphicsProfile(), tea.Raw("\x1b[16t"))
			}
			return graphicsProfile()
		}
		if m.imageState.config.Images == "blocks" && !m.noColor {
			return tea.Batch(graphicsProfile(), tea.Raw("\x1b[16t"))
		}
		return nil
	}
	m.imageState.deadline = time.Now().Add(500 * time.Millisecond)
	id := m.imageState.query
	return tea.Batch(tea.Raw(images.Query(id)+"\x1b[16t"), tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return imageTimeout{id} }))
}

func (m *Model) releaseImages() string {
	var out strings.Builder
	for _, entry := range m.imageState.entries {
		if entry.cancel != nil {
			entry.cancel()
		}
		if entry.id != 0 {
			if !m.discardUpload(entry.id) {
				out.WriteString(images.Delete(entry.id))
			}
		}
	}
	m.imageState.entries = map[string]*imageEntry{}
	m.imageState.failed = map[string]bool{}
	m.imageState.dimensions = map[string][2]int{}
	m.imageState.generation++
	return out.String()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var command tea.Cmd
	switch event := msg.(type) {
	case uv.KittyGraphicsEvent:
		if m.imageState.pending && event.Options.ID == m.imageState.query && (m.imageState.deadline.IsZero() || time.Now().Before(m.imageState.deadline)) {
			m.imageState.pending = false
			m.imageState.kitty = string(event.Payload) == "OK"
			if m.imageState.kitty {
				command = graphicsProfile()
			}
		}
	case imageTimeout:
		if event.id == m.imageState.query {
			m.imageState.pending = false
		}
	case uv.CellSizeEvent:
		if event.Width > 0 && event.Height > 0 {
			m.imageState.cellAspect = float64(event.Width) / float64(event.Height)
		}
	case protocolDone:
		m.imageState.writing = false
		if event.uploaded != nil {
			uploaded := event.uploaded
			if uploaded.generation == m.imageState.generation {
				if entry := m.imageState.entries[uploaded.source]; entry != nil && entry.id == uploaded.id {
					entry.ready = true
				}
			}
		}
	case imageRendered:
		if event.generation == m.imageState.generation {
			if entry := m.imageState.entries[event.source]; entry != nil && entry.renderToken == event.token {
				if event.err != nil {
					entry.failed = true
					m.imageState.failed[event.source] = true
				} else {
					entry.lines = event.lines
					entry.pixels = nil
					entry.rendering = false
					if event.output != "" {
						m.queueProtocol(event.output, &imageUploaded{event.generation, event.source, entry.id})
					} else {
						entry.ready = true
					}
				}
			}
		}
	case imageResult:
		if event.generation == m.imageState.generation {
			if entry := m.imageState.entries[event.source]; entry != nil && entry.token == event.token {
				entry.cancel = nil
				entry.failed = event.err != nil
				if entry.failed {
					m.imageState.failed[event.source] = true
				} else {
					m.imageState.dimensions[event.source] = [2]int{event.width, event.height}
				}
				entry.pixels = event.pixels
				entry.width, entry.height = event.width, event.height
			}
		}
	default:
		if key, ok := msg.(tea.KeyPressMsg); ok && (key.String() == "ctrl+c" || key.String() == "q" && !m.input) {
			output := m.releaseImages()
			m.queueProtocol(output, nil)
			m.imageState.quitting = true
			return m, m.pumpProtocol()
		}
		_, command = m.update(msg)
	}
	if m.imageState.quitting {
		return m, m.pumpProtocol()
	}
	work := m.syncImages()
	return m, tea.Batch(command, work, m.pumpProtocol())
}

func (m *Model) imageAllowed() bool {
	s := &m.imageState
	return s.service != nil && !m.noColor && !s.pending && (s.config.Images == "kitty" || s.config.Images == "blocks" || s.config.Images == "auto" && (s.kitty || s.config.ImageFallback == "blocks"))
}

func (m *Model) displayArticle() *domain.Article {
	if m.help || m.picker || m.input || m.confirm {
		return nil
	}
	if m.wide() {
		return m.previewArticle()
	}
	return m.reader
}

func (m *Model) imageLines(block app.ArticleBlock, columns int) []string {
	entry := m.imageState.entries[block.Source]
	if entry == nil || entry.failed {
		rows := 1
		if dimensions, ok := m.imageState.dimensions[block.Source]; ok {
			_, rows = images.Size(dimensions[0], dimensions[1], columns, m.imageState.cellAspect)
		}
		lines := make([]string, rows)
		lines[0] = block.Placeholder()
		return lines
	}
	columns, rows := images.Size(entry.width, entry.height, columns, m.imageState.cellAspect)
	if entry.columns == columns && entry.rows == rows && len(entry.lines) > 0 && entry.ready {
		return entry.lines
	}
	lines := make([]string, rows)
	lines[0] = block.Placeholder()
	return lines
}

type imageSpan struct {
	block        app.ArticleBlock
	start, count int
}

func appendImageLayout(lines *[]string, spans *[]imageSpan, next []string, images []imageSpan, indent string) {
	offset := len(*lines)
	for _, span := range images {
		span.start += offset
		*spans = append(*spans, span)
	}
	for _, line := range next {
		*lines = append(*lines, indent+line)
	}
}

func (m *Model) leadImageLayout(article *domain.Article, columns int) ([]string, []imageSpan) {
	for _, block := range app.ArticleBlocks(article.Content, article.URL) {
		if block.Image {
			lines := m.imageLines(block, columns)
			return append(lines, ""), []imageSpan{{block, 0, len(lines)}}
		}
	}
	return nil, nil
}

func (m *Model) articleBodyLayout(article *domain.Article, columns int) ([]string, []imageSpan) {
	var lines []string
	var spans []imageSpan
	lead := true
	for _, block := range app.ArticleBlocks(article.Content, article.URL) {
		if block.Image && lead {
			lead = false
			continue
		}
		var next []string
		if block.Image {
			next = m.imageLines(block, columns)
		} else {
			next = justifyText(block.Text, columns)
		}
		if len(lines) > 0 && len(next) > 0 {
			lines = append(lines, "")
		}
		if block.Image {
			spans = append(spans, imageSpan{block, len(lines), len(next)})
		}
		lines = append(lines, next...)
	}
	return lines, spans
}

func (m *Model) syncImages() tea.Cmd {
	if m.imageState.service == nil {
		return nil
	}
	article := m.displayArticle()
	id := int64(0)
	if article != nil {
		id = article.ID
	}
	var commands []tea.Cmd
	if id != m.imageState.article {
		output := m.releaseImages()
		m.imageState.article = id
		if output != "" {
			m.queueProtocol(output, nil)
		}
	}
	if article == nil || !m.imageAllowed() {
		return tea.Batch(commands...)
	}
	panel := m.width
	if m.wide() {
		panel = m.width - m.indexWidth() - 1
	}
	columns := max(1, min(NewspaperColumnWidth, panel-4))

	var spans []imageSpan
	if m.wide() {
		_, spans = m.previewLayout()
	} else {
		_, spans = m.readerLayout()
	}
	start, end := m.scroll, m.scroll+m.readerHeight()
	visible := map[string]bool{}
	for _, span := range spans {
		if span.start < end && span.start+span.count > start && span.block.Source != "" {
			visible[span.block.Source] = true
		}
	}
	for source, entry := range m.imageState.entries {
		if !visible[source] || entry.failed {
			if entry.cancel != nil {
				entry.cancel()
			}
			if entry.id != 0 {
				if !m.discardUpload(entry.id) {
					m.queueProtocol(images.Delete(entry.id), nil)
				}
			}
			delete(m.imageState.entries, source)
		}
	}
	active := 0
	for _, entry := range m.imageState.entries {
		if entry.cancel != nil || entry.rendering || entry.pixels != nil {
			active++
		}
	}
	for _, span := range spans {
		block := span.block
		if !visible[block.Source] || m.imageState.failed[block.Source] {
			continue
		}
		entry := m.imageState.entries[block.Source]
		needsLoad := entry == nil
		if entry != nil {
			desiredColumns, desiredRows := images.Size(entry.width, entry.height, columns, m.imageState.cellAspect)
			needsLoad = entry.cancel == nil && !entry.rendering && entry.pixels == nil && (entry.columns != desiredColumns || entry.rows != desiredRows)
		}
		if needsLoad {
			if active >= 8 {
				continue
			}
			active++

			ctx, cancel := context.WithCancel(m.ctx)
			if entry == nil {
				dimensions := m.imageState.dimensions[block.Source]
				entry = &imageEntry{width: dimensions[0], height: dimensions[1]}
			}
			entry.cancel, entry.token = cancel, graphicsID()
			m.imageState.entries[block.Source] = entry
			token := entry.token
			source, generation, service := block.Source, m.imageState.generation, m.imageState.service
			commands = append(commands, func() tea.Msg {
				decoded, err := service.Load(ctx, source)
				result := imageResult{token: token, generation: generation, source: source, err: err}
				if err == nil {
					bounds := decoded.Bounds()
					result.width, result.height = bounds.Dx(), bounds.Dy()
					scale := min(1.0, min(480.0/float64(bounds.Dx()), 1024.0/float64(bounds.Dy())))
					result.pixels = images.Scale(decoded, max(1, int(float64(bounds.Dx())*scale)), max(1, int(float64(bounds.Dy())*scale)))
				}
				return result
			})
		}
		imageColumns, imageRows := images.Size(entry.width, entry.height, columns, m.imageState.cellAspect)
		if entry.pixels != nil && (entry.columns != imageColumns || entry.rows != imageRows) {
			if entry.id != 0 {
				if !m.discardUpload(entry.id) {
					m.queueProtocol(images.Delete(entry.id), nil)
				}
			}

			entry.columns, entry.rows = imageColumns, imageRows
			entry.lines, entry.ready, entry.rendering = nil, false, true
			entry.renderToken = graphicsID()
			if m.imageState.kitty {
				entry.id = graphicsID()
			}
			source, generation, token := block.Source, m.imageState.generation, entry.renderToken
			pixels, imageID, kittyMode := entry.pixels, entry.id, m.imageState.kitty
			commands = append(commands, func() tea.Msg {
				result := imageRendered{generation: generation, source: source, token: token}
				if kittyMode {
					result.lines = images.Placeholders(imageID, 1, imageColumns, imageRows)
					result.output, result.err = images.Upload(pixels, imageID, 1, imageColumns, imageRows)
				} else {
					result.lines = images.Blocks(pixels, imageColumns, imageRows)
				}
				return result
			})
		}
	}
	return tea.Batch(commands...)
}

func (m *Model) queueProtocol(output string, uploaded *imageUploaded) {
	if output != "" {
		m.imageState.queue = append(m.imageState.queue, protocolItem{output, uploaded})
	}
}

func (m *Model) pumpProtocol() tea.Cmd {
	if m.imageState.writing {
		return nil
	}
	if len(m.imageState.queue) == 0 {
		if m.imageState.quitting {
			return tea.Quit
		}
		return nil
	}
	item := m.imageState.queue[0]
	m.imageState.queue = m.imageState.queue[1:]
	m.imageState.writing = true
	return tea.Sequence(tea.Raw(item.output), func() tea.Msg { return protocolDone{item.uploaded} })
}

func (m *Model) discardUpload(id int) bool {
	for index, item := range m.imageState.queue {
		if item.uploaded != nil && item.uploaded.id == id {
			m.imageState.queue = append(m.imageState.queue[:index], m.imageState.queue[index+1:]...)
			return true
		}
	}
	return false
}

func graphicsProfile() tea.Cmd {
	return func() tea.Msg { return tea.ColorProfileMsg{Profile: colorprofile.TrueColor} }
}
