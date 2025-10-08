package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/papaganelli/tailnginx/pkg/parser"
)

// tickMsg is sent on every timer tick
type tickMsg time.Time

// logLineMsg wraps an incoming log line
type logLineMsg string

// Model holds the Bubble Tea application state
type Model struct {
	lines       <-chan string
	visitors    []parser.Visitor
	uniqueIPs   map[string]int
	statusCodes map[int]int
	topPaths    map[string]int
	userAgents  map[string]int
	width       int
	height      int
	startTime   time.Time
}

// NewApp creates a new Bubble Tea model
func NewApp(lines <-chan string) *Model {
	return &Model{
		lines:       lines,
		visitors:    []parser.Visitor{},
		uniqueIPs:   make(map[string]int),
		statusCodes: make(map[int]int),
		topPaths:    make(map[string]int),
		userAgents:  make(map[string]int),
		startTime:   time.Now(),
	}
}

// Init initializes the Bubble Tea program
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		waitForLine(m.lines),
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		return m, tickCmd()

	case logLineMsg:
		// Parse the log line
		if visitor := parser.Parse(string(msg)); visitor != nil {
			// Add to visitors list (keep last 100)
			m.visitors = append([]parser.Visitor{*visitor}, m.visitors...)
			if len(m.visitors) > 100 {
				m.visitors = m.visitors[:100]
			}

			// Update stats
			m.uniqueIPs[visitor.IP]++
			m.statusCodes[visitor.Status]++
			m.topPaths[visitor.Path]++

			// Extract browser from user agent
			agent := extractBrowser(visitor.Agent)
			m.userAgents[agent]++
		}
		return m, waitForLine(m.lines)
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginTop(1)

	statStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46"))

	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220"))

	// Build the UI
	var s strings.Builder

	// Title
	s.WriteString(titleStyle.Render("NGINX MONITOR"))
	s.WriteString("\n")
	s.WriteString(labelStyle.Render(fmt.Sprintf("Running since: %s | Press 'q' to quit", m.startTime.Format("15:04:05"))))
	s.WriteString("\n")

	// Overview stats
	s.WriteString(headerStyle.Render("📊 OVERVIEW"))
	s.WriteString("\n")
	s.WriteString(statStyle.Render(fmt.Sprintf("Total Requests: %d | Unique IPs: %d | Uptime: %s",
		len(m.visitors),
		len(m.uniqueIPs),
		time.Since(m.startTime).Round(time.Second),
	)))
	s.WriteString("\n")

	// Status codes
	s.WriteString(headerStyle.Render("📡 STATUS CODES"))
	s.WriteString("\n")
	statuses := sortMapByValue(m.statusCodes)
	for i, kv := range statuses {
		if i >= 5 {
			break
		}
		statusStr := fmt.Sprintf("%d", kv.key)
		var styled string
		switch {
		case kv.key >= 200 && kv.key < 300:
			styled = successStyle.Render(statusStr)
		case kv.key >= 300 && kv.key < 400:
			styled = warningStyle.Render(statusStr)
		case kv.key >= 400:
			styled = errorStyle.Render(statusStr)
		default:
			styled = statStyle.Render(statusStr)
		}
		s.WriteString(fmt.Sprintf("  %s: %d\n", styled, kv.value))
	}

	// Top paths
	s.WriteString(headerStyle.Render("🔥 TOP PATHS"))
	s.WriteString("\n")
	paths := sortMapByValue(m.topPaths)
	for i, kv := range paths {
		if i >= 5 {
			break
		}
		pathStr := kv.key
		if len(pathStr) > 50 {
			pathStr = pathStr[:47] + "..."
		}
		s.WriteString(fmt.Sprintf("  %s %s\n",
			statStyle.Render(pathStr),
			labelStyle.Render(fmt.Sprintf("(%d)", kv.value)),
		))
	}

	// Top IPs
	s.WriteString(headerStyle.Render("👥 TOP VISITORS"))
	s.WriteString("\n")
	ips := sortMapByValue(m.uniqueIPs)
	for i, kv := range ips {
		if i >= 5 {
			break
		}
		s.WriteString(fmt.Sprintf("  %s %s\n",
			statStyle.Render(kv.key),
			labelStyle.Render(fmt.Sprintf("(%d requests)", kv.value)),
		))
	}

	// User agents
	s.WriteString(headerStyle.Render("🌐 BROWSERS / CLIENTS"))
	s.WriteString("\n")
	agents := sortMapByValue(m.userAgents)
	for i, kv := range agents {
		if i >= 5 {
			break
		}
		s.WriteString(fmt.Sprintf("  %s %s\n",
			statStyle.Render(kv.key),
			labelStyle.Render(fmt.Sprintf("(%d)", kv.value)),
		))
	}

	// Recent requests
	s.WriteString(headerStyle.Render("📝 RECENT REQUESTS"))
	s.WriteString("\n")
	recentCount := 8
	if len(m.visitors) < recentCount {
		recentCount = len(m.visitors)
	}
	for i := 0; i < recentCount; i++ {
		v := m.visitors[i]
		timeStr := v.Time.Format("15:04:05")
		var statusStyled string
		switch {
		case v.Status >= 200 && v.Status < 300:
			statusStyled = successStyle.Render(fmt.Sprintf("%d", v.Status))
		case v.Status >= 300 && v.Status < 400:
			statusStyled = warningStyle.Render(fmt.Sprintf("%d", v.Status))
		case v.Status >= 400:
			statusStyled = errorStyle.Render(fmt.Sprintf("%d", v.Status))
		default:
			statusStyled = statStyle.Render(fmt.Sprintf("%d", v.Status))
		}

		pathStr := v.Path
		if len(pathStr) > 40 {
			pathStr = pathStr[:37] + "..."
		}

		s.WriteString(fmt.Sprintf("  %s %s %s %s %s\n",
			labelStyle.Render(timeStr),
			statStyle.Render(v.IP),
			statStyle.Render(v.Method),
			statStyle.Render(pathStr),
			statusStyled,
		))
	}

	return s.String()
}

// Run starts the Bubble Tea program
func (m *Model) Run() error {
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// tickCmd returns a command that ticks every second
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// waitForLine waits for the next log line from the channel
func waitForLine(lines <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-lines
		if !ok {
			return tea.Quit()
		}
		return logLineMsg(line)
	}
}

// extractBrowser extracts a browser name from a user agent string
func extractBrowser(agent string) string {
	agent = strings.ToLower(agent)

	switch {
	case strings.Contains(agent, "curl"):
		return "curl"
	case strings.Contains(agent, "wget"):
		return "wget"
	case strings.Contains(agent, "postman"):
		return "Postman"
	case strings.Contains(agent, "chrome") && !strings.Contains(agent, "edg"):
		return "Chrome"
	case strings.Contains(agent, "edg"):
		return "Edge"
	case strings.Contains(agent, "safari") && !strings.Contains(agent, "chrome"):
		return "Safari"
	case strings.Contains(agent, "firefox"):
		return "Firefox"
	case strings.Contains(agent, "bot"):
		return "Bot"
	case strings.Contains(agent, "spider"):
		return "Crawler"
	default:
		if len(agent) > 20 {
			return agent[:20] + "..."
		}
		return agent
	}
}

// keyValue is a helper for sorting maps
type keyValue[K comparable, V any] struct {
	key   K
	value V
}

// sortMapByValue sorts a map by its values in descending order
func sortMapByValue[K comparable](m map[K]int) []keyValue[K, int] {
	var kv []keyValue[K, int]
	for k, v := range m {
		kv = append(kv, keyValue[K, int]{k, v})
	}
	sort.Slice(kv, func(i, j int) bool {
		return kv[i].value > kv[j].value
	})
	return kv
}
