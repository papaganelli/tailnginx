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
	methods     map[string]int
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
		methods:     make(map[string]int),
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
			m.methods[visitor.Method]++

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

	// Color palette
	var (
		primaryColor   = lipgloss.Color("86")   // Cyan
		secondaryColor = lipgloss.Color("213")  // Pink
		successColor   = lipgloss.Color("46")   // Green
		warningColor   = lipgloss.Color("220")  // Yellow
		errorColor     = lipgloss.Color("196")  // Red
		textColor      = lipgloss.Color("252")  // Light gray
		dimColor       = lipgloss.Color("241")  // Dark gray
		borderColor    = lipgloss.Color("240")  // Border gray
		accentColor    = lipgloss.Color("117")  // Light blue
	)

	// Base styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Background(lipgloss.Color("235")).
		Padding(0, 2).
		MarginBottom(1)

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		MarginRight(1).
		MarginBottom(1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentColor).
		MarginBottom(1)

	metricLabelStyle := lipgloss.NewStyle().
		Foreground(dimColor).
		Width(18)

	metricValueStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true)

	statLabelStyle := lipgloss.NewStyle().
		Foreground(textColor)

	// Calculate panel widths
	panelWidth := (m.width / 2) - 4
	if panelWidth < 30 {
		panelWidth = 30
	}

	// Header
	uptime := time.Since(m.startTime).Round(time.Second)
	header := titleStyle.Render("🚀 TAILNGINX DASHBOARD") + "\n" +
		lipgloss.NewStyle().Foreground(dimColor).Render(
			fmt.Sprintf("Running: %s  •  Press 'q' to quit", uptime),
		)

	// Overview metrics panel
	overviewContent := headerStyle.Render("📊 Overview") + "\n" +
		metricLabelStyle.Render("Total Requests") + metricValueStyle.Render(fmt.Sprintf("%d", len(m.visitors))) + "\n" +
		metricLabelStyle.Render("Unique Visitors") + metricValueStyle.Render(fmt.Sprintf("%d", len(m.uniqueIPs))) + "\n" +
		metricLabelStyle.Render("Avg Bytes/Req") + metricValueStyle.Render(m.calculateAvgBytes())

	overviewPanel := panelStyle.Width(panelWidth).Render(overviewContent)

	// Status codes panel with bars
	statusContent := headerStyle.Render("📡 HTTP Status Codes") + "\n"
	totalReqs := len(m.visitors)
	if totalReqs == 0 {
		totalReqs = 1
	}

	statuses := sortMapByValue(m.statusCodes)
	for i, kv := range statuses {
		if i >= 5 {
			break
		}
		percentage := float64(kv.value) / float64(totalReqs) * 100
		bar := createBar(int(percentage), 20)

		var statusColor lipgloss.Color
		switch {
		case kv.key >= 200 && kv.key < 300:
			statusColor = successColor
		case kv.key >= 300 && kv.key < 400:
			statusColor = warningColor
		default:
			statusColor = errorColor
		}

		statusContent += fmt.Sprintf("%s %s %s\n",
			lipgloss.NewStyle().Foreground(statusColor).Render(fmt.Sprintf("%3d", kv.key)),
			bar,
			lipgloss.NewStyle().Foreground(dimColor).Render(fmt.Sprintf("%3d (%.1f%%)", kv.value, percentage)),
		)
	}

	statusPanel := panelStyle.Width(panelWidth).Render(statusContent)

	// Top row: Overview + Status codes
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, overviewPanel, statusPanel)

	// Top paths panel
	pathsContent := headerStyle.Render("🔥 Top Paths") + "\n"
	paths := sortMapByValue(m.topPaths)
	for i, kv := range paths {
		if i >= 6 {
			break
		}
		pathStr := kv.key
		if len(pathStr) > panelWidth-15 {
			pathStr = pathStr[:panelWidth-18] + "..."
		}

		count := lipgloss.NewStyle().Foreground(secondaryColor).Render(fmt.Sprintf("%3d", kv.value))
		pathsContent += fmt.Sprintf("%s  %s\n", count, statLabelStyle.Render(pathStr))
	}

	pathsPanel := panelStyle.Width(panelWidth).Render(pathsContent)

	// Top visitors panel
	visitorsContent := headerStyle.Render("👥 Top Visitors") + "\n"
	ips := sortMapByValue(m.uniqueIPs)
	for i, kv := range ips {
		if i >= 6 {
			break
		}
		count := lipgloss.NewStyle().Foreground(secondaryColor).Render(fmt.Sprintf("%3d", kv.value))
		visitorsContent += fmt.Sprintf("%s  %s\n",
			count,
			statLabelStyle.Render(kv.key),
		)
	}

	visitorsPanel := panelStyle.Width(panelWidth).Render(visitorsContent)

	// Middle row: Paths + Visitors
	middleRow := lipgloss.JoinHorizontal(lipgloss.Top, pathsPanel, visitorsPanel)

	// Browsers panel
	browsersContent := headerStyle.Render("🌐 Clients & Browsers") + "\n"
	agents := sortMapByValue(m.userAgents)
	for i, kv := range agents {
		if i >= 6 {
			break
		}
		count := lipgloss.NewStyle().Foreground(secondaryColor).Render(fmt.Sprintf("%3d", kv.value))
		browsersContent += fmt.Sprintf("%s  %s\n", count, statLabelStyle.Render(kv.key))
	}

	browsersPanel := panelStyle.Width(panelWidth).Render(browsersContent)

	// Methods panel
	methodsContent := headerStyle.Render("🔧 HTTP Methods") + "\n"
	methods := sortMapByValue(m.methods)
	for i, kv := range methods {
		if i >= 6 {
			break
		}
		var methodColor lipgloss.Color
		switch kv.key {
		case "GET":
			methodColor = successColor
		case "POST":
			methodColor = accentColor
		case "PUT", "PATCH":
			methodColor = warningColor
		case "DELETE":
			methodColor = errorColor
		default:
			methodColor = textColor
		}

		count := lipgloss.NewStyle().Foreground(secondaryColor).Render(fmt.Sprintf("%3d", kv.value))
		method := lipgloss.NewStyle().Foreground(methodColor).Bold(true).Render(fmt.Sprintf("%-6s", kv.key))
		methodsContent += fmt.Sprintf("%s  %s\n", count, method)
	}

	methodsPanel := panelStyle.Width(panelWidth).Render(methodsContent)

	// Lower middle row: Browsers + Methods
	lowerMiddleRow := lipgloss.JoinHorizontal(lipgloss.Top, browsersPanel, methodsPanel)

	// Recent requests panel (full width)
	recentContent := headerStyle.Render("📝 Recent Activity") + "\n"
	recentCount := 7
	if len(m.visitors) < recentCount {
		recentCount = len(m.visitors)
	}

	for i := 0; i < recentCount; i++ {
		v := m.visitors[i]
		timeStr := lipgloss.NewStyle().Foreground(dimColor).Render(v.Time.Format("15:04:05"))

		var statusStyle lipgloss.Style
		switch {
		case v.Status >= 200 && v.Status < 300:
			statusStyle = lipgloss.NewStyle().Foreground(successColor).Bold(true)
		case v.Status >= 300 && v.Status < 400:
			statusStyle = lipgloss.NewStyle().Foreground(warningColor).Bold(true)
		default:
			statusStyle = lipgloss.NewStyle().Foreground(errorColor).Bold(true)
		}

		var methodColor lipgloss.Color
		switch v.Method {
		case "GET":
			methodColor = successColor
		case "POST":
			methodColor = accentColor
		case "PUT", "PATCH":
			methodColor = warningColor
		case "DELETE":
			methodColor = errorColor
		default:
			methodColor = textColor
		}

		pathStr := v.Path
		maxPathLen := m.width - 60
		if maxPathLen < 20 {
			maxPathLen = 20
		}
		if len(pathStr) > maxPathLen {
			pathStr = pathStr[:maxPathLen-3] + "..."
		}

		recentContent += fmt.Sprintf("%s  %-15s  %s  %s  %s\n",
			timeStr,
			lipgloss.NewStyle().Foreground(textColor).Render(v.IP),
			lipgloss.NewStyle().Foreground(methodColor).Bold(true).Render(fmt.Sprintf("%-6s", v.Method)),
			statusStyle.Render(fmt.Sprintf("%3d", v.Status)),
			lipgloss.NewStyle().Foreground(textColor).Render(pathStr),
		)
	}

	recentPanel := panelStyle.Width(m.width - 4).Render(recentContent)

	// Assemble the dashboard
	dashboard := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		topRow,
		middleRow,
		lowerMiddleRow,
		recentPanel,
	)

	return dashboard
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

// calculateAvgBytes calculates average bytes per request
func (m *Model) calculateAvgBytes() string {
	if len(m.visitors) == 0 {
		return "0 B"
	}

	total := 0
	for _, v := range m.visitors {
		total += v.Bytes
	}

	avg := total / len(m.visitors)

	if avg < 1024 {
		return fmt.Sprintf("%d B", avg)
	} else if avg < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(avg)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(avg)/(1024*1024))
}

// createBar creates a simple progress bar
func createBar(percentage, width int) string {
	if percentage > 100 {
		percentage = 100
	}
	if percentage < 0 {
		percentage = 0
	}

	filled := int(float64(width) * float64(percentage) / 100)
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(bar)
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
