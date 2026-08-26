package main

import (
	"math"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"asciicity/shell"
)

// model drives the session from bubbletea's event loop. Window size and key
// messages arrive as they happen. A self-rescheduling tick paces the render
// loop at the configured frame rate.
type model struct {
	sess *session
	keys *keyboard

	fps        int
	cols, rows int // 0 means fill whatever bubbletea reports
	aspect     float64

	last  time.Time
	ready bool
}

func newModel(s *session, fps, cols, rows int, aspect float64) *model {
	return &model{sess: s, keys: newKeyboard(), fps: fps, cols: cols, rows: rows, aspect: aspect}
}

type tickMsg time.Time

func tick(fps int) tea.Cmd {
	d := time.Second / time.Duration(max(1, fps))
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *model) Init() tea.Cmd {
	m.last = time.Now()
	return tick(m.fps)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w, h := msg.Width, msg.Height
		if m.cols > 0 {
			w = m.cols
		}
		if m.rows > 0 {
			h = m.rows
		}
		m.sess.resize(w, h, m.aspect)
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		m.keys.handle(msg)
		if m.keys.quit {
			return m, tea.Quit
		}
		return m, nil

	case tickMsg:
		now := time.Time(msg)
		dt := math.Min(now.Sub(m.last).Seconds(), 0.1)
		m.last = now
		if m.ready {
			m.sess.step(m.keys, dt)
		}
		return m, tick(m.fps)
	}
	return m, nil
}

func (m *model) View() string {
	if !m.ready {
		return ""
	}
	screen := m.sess.render()
	if m.sess.hud {
		shell.Status(screen, m.sess.status())
	}
	return shell.Ansi(screen)
}
