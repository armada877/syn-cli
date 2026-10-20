package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/armada877/syn-cli/internal/domain"
)

func wheel(t *testing.T, m *Model, button tea.MouseButton, x, y int) {
	t.Helper()
	_, cmd := m.Update(tea.MouseWheelMsg{Button: button, X: x, Y: y})
	if cmd != nil {
		t.Fatal("wheel returned a command")
	}
}

func wheelFixture(t *testing.T) *Model {
	t.Helper()
	m, _, _ := fixture(t)
	m.articles = nil
	for i := 0; i < 8; i++ {
		m.articles = append(m.articles, domain.Article{ID: int64(i + 1), Read: i%2 == 0, Content: strings.Repeat("word ", 1000)})
	}
	m.selected = 0
	m.reconcile()
	return m
}

func TestWheelArticleSelection(t *testing.T) {
	for _, width := range []int{80, 120} {
		for _, focused := range []bool{false, true} {
			if width == 80 && focused {
				continue
			}
			m := wheelFixture(t)
			resize(m, width)
			if focused {
				a := m.articles[0]
				m.reader = &a
			}
			before := append([]domain.Article(nil), m.articles...)
			y := len(m.headerRows())
			for _, want := range []int{3, 6, 7, 7} {
				wheel(t, m, tea.MouseWheelDown, 0, y)
				if m.articleIndex != want || m.selected != m.articles[want].ID {
					t.Fatal("selection", m.articleIndex, want)
				}
				if focused && (m.reader == nil || m.reader.ID != m.selected) {
					t.Fatal("focused preview")
				}
				if !focused && m.reader != nil {
					t.Fatal("focus changed")
				}
			}
			for _, want := range []int{4, 1, 0, 0} {
				wheel(t, m, tea.MouseWheelUp, 0, y)
				if m.articleIndex != want {
					t.Fatal("selection", m.articleIndex, want)
				}
			}
			if !reflect.DeepEqual(before, m.articles) {
				t.Fatal("articles changed")
			}
		}
	}
}

func TestWheelContent(t *testing.T) {
	for _, width := range []int{80, 120} {
		for _, focused := range []bool{false, true} {
			if width == 80 && !focused {
				continue
			}
			m := wheelFixture(t)
			resize(m, width)
			if focused {
				a := m.articles[0]
				m.reader = &a
			}
			x, y := m.width-1, len(m.headerRows())
			selected, reader := m.selected, m.reader
			wheel(t, m, tea.MouseWheelDown, x, y)
			if m.scroll != 3 {
				t.Fatal("scroll step", m.scroll)
			}
			wheel(t, m, tea.MouseWheelUp, x, y)
			wheel(t, m, tea.MouseWheelUp, x, y)
			if m.scroll != 0 {
				t.Fatal("upper limit")
			}
			m.scroll = m.maxScroll() - 1
			wheel(t, m, tea.MouseWheelDown, x, y)
			wheel(t, m, tea.MouseWheelDown, x, y)
			if m.scroll != m.maxScroll() || m.selected != selected || m.reader != reader {
				t.Fatal("lower limit or focus")
			}
		}
	}
}

func TestWheelPickerAndEmptyLists(t *testing.T) {
	m := wheelFixture(t)
	m.picker = true
	m.feeds = make([]domain.Feed, 7)
	for _, want := range []int{3, 6, 7, 7} {
		wheel(t, m, tea.MouseWheelDown, 0, 0)
		if m.feedIndex != want {
			t.Fatal(m.feedIndex, want)
		}
	}
	for _, want := range []int{4, 1, 0, 0} {
		wheel(t, m, tea.MouseWheelUp, 0, 0)
		if m.feedIndex != want {
			t.Fatal(m.feedIndex, want)
		}
	}
	m.feeds = nil
	wheel(t, m, tea.MouseWheelDown, 0, 0)
	if m.feedIndex != 0 {
		t.Fatal("empty feeds")
	}
	m.picker = false
	m.articles = nil
	m.selected = 0
	m.articleIndex = 0
	for _, width := range []int{80, 120} {
		resize(m, width)
		wheel(t, m, tea.MouseWheelDown, 0, len(m.headerRows()))
		wheel(t, m, tea.MouseWheelUp, 0, len(m.headerRows()))
		if m.selected != 0 || m.articleIndex != 0 {
			t.Fatal("empty articles")
		}
	}
}

func TestWheelGuards(t *testing.T) {
	for _, focused := range []bool{false, true} {
		m := wheelFixture(t)
		resize(m, 120)
		if focused {
			a := m.articles[0]
			m.reader = &a
		}
		y := len(m.headerRows())
		for _, point := range [][2]int{{-1, y}, {m.width, y}, {0, -1}, {0, y - 1}, {0, m.height - 1}, {0, m.height}, {m.indexWidth(), y}} {
			before := wheelState(m)
			wheel(t, m, tea.MouseWheelDown, point[0], point[1])
			if !reflect.DeepEqual(before, wheelState(m)) {
				t.Fatal("outside panel", point)
			}
		}
	}
	for _, mode := range []string{"help", "input", "confirm", "dragging"} {
		m := wheelFixture(t)
		resize(m, 120)
		switch mode {
		case "help":
			m.help = true
		case "input":
			m.input = true
		case "confirm":
			m.confirm = true
		case "dragging":
			m.dragging = true
		}
		before := wheelState(m)
		wheel(t, m, tea.MouseWheelDown, 0, len(m.headerRows()))
		if !reflect.DeepEqual(before, wheelState(m)) {
			t.Fatal("modal changed", mode)
		}
	}
	m := wheelFixture(t)
	before := wheelState(m)
	wheel(t, m, tea.MouseWheelLeft, 0, 0)
	wheel(t, m, tea.MouseWheelRight, 0, 0)
	if !reflect.DeepEqual(before, wheelState(m)) {
		t.Fatal("horizontal wheel changed state")
	}
}

func TestWheelMouseModes(t *testing.T) {
	m := wheelFixture(t)
	for _, width := range []int{80, 120} {
		resize(m, width)
		for _, mode := range []string{"stream", "reader", "picker", "help", "input", "confirm"} {
			m.reader = nil
			m.picker = false
			m.help = false
			m.input = false
			m.confirm = false
			want := tea.MouseModeCellMotion
			switch mode {
			case "reader":
				a := m.articles[0]
				m.reader = &a
			case "picker":
				m.picker = true
			case "help":
				m.help = true
				want = tea.MouseModeNone
			case "input":
				m.input = true
				want = tea.MouseModeNone
			case "confirm":
				m.confirm = true
				want = tea.MouseModeNone
			}
			if m.View().MouseMode != want {
				t.Fatal(width, mode, m.View().MouseMode)
			}
		}
	}
}

func wheelState(m *Model) Model {
	state := *m
	state.now = nil
	return state
}

func TestWheelPreservesStoredReadState(t *testing.T) {
	m, r, _ := fixture(t)
	resize(m, 120)
	finish(m, press(m, "enter"))
	before := append([]domain.Article(nil), r.articles...)
	calls, readCalls := r.calls, r.readCalls
	wheel(t, m, tea.MouseWheelDown, 0, len(m.headerRows()))
	if m.reader == nil || m.reader.ID != 10 || m.reader.Read {
		t.Fatal("stable reader did not follow selection")
	}
	wheel(t, m, tea.MouseWheelDown, m.width-1, len(m.headerRows()))
	wheel(t, m, tea.MouseWheelUp, 0, len(m.headerRows()))
	if r.calls != calls || r.readCalls != readCalls || !reflect.DeepEqual(before, r.articles) {
		t.Fatal("wheel changed storage")
	}
}

func TestWheelStableReaderOutsideUnreadIndex(t *testing.T) {
	m := wheelFixture(t)
	resize(m, 120)
	m.filter.UnreadOnly = true
	a := m.articles[0]
	m.reader = &a
	m.reconcile()
	wheel(t, m, tea.MouseWheelDown, 0, len(m.headerRows()))
	if m.reader == nil || m.reader.ID != 8 || m.selected != 8 || m.reader.Read {
		t.Fatal("unread selection lost reader")
	}
}
