// Package ui provides the terminal user interface for tailnginx using Bubble Tea.
package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/papaganelli/tailnginx/pkg/geoip"
	"github.com/papaganelli/tailnginx/pkg/parser"
)

// tickMsg is sent on every timer tick
type tickMsg time.Time

// logLineMsg wraps an incoming log line
type logLineMsg string

// Model holds the Bubble Tea application state
type Model struct {
	lines           <-chan string
	visitors        []parser.Visitor
	allVisitors     []parser.Visitor // Unfiltered list
	uniqueIPs       map[string]int
	statusCodes     map[int]int
	topPaths        map[string]int
	userAgents      map[string]int
	methods         map[string]int
	countries       map[string]int // Country name -> count
	width           int
	height          int
	startTime       time.Time
	refreshRate     time.Duration
	paused          bool
	filterIP        string
	filterStatus    int // 0=all, 2=2xx, 3=3xx, 4=4xx, 5=5xx
	filterPath      string
	filterInputMode bool   // true when typing a filter
	filterInput     string // current filter input
	geoLocator      *geoip.Locator
}

// NewApp creates a new Bubble Tea model
func NewApp(lines <-chan string, refreshRate time.Duration, geoLocator *geoip.Locator) *Model {
	return &Model{
		lines:        lines,
		visitors:     []parser.Visitor{},
		allVisitors:  []parser.Visitor{},
		uniqueIPs:    make(map[string]int),
		statusCodes:  make(map[int]int),
		topPaths:     make(map[string]int),
		userAgents:   make(map[string]int),
		methods:      make(map[string]int),
		countries:    make(map[string]int),
		startTime:    time.Now(),
		refreshRate:  refreshRate,
		filterStatus: 0, // Show all by default
		geoLocator:   geoLocator,
	}
}

// Init initializes the Bubble Tea program
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(m.refreshRate),
		waitForLine(m.lines),
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle filter input mode separately
		if m.filterInputMode {
			switch msg.String() {
			case "enter":
				// Apply filter
				m.filterIP = m.filterInput
				m.filterInputMode = false
				m.filterInput = ""
				m.applyFilters()
			case "esc":
				// Cancel filter input
				m.filterInputMode = false
				m.filterInput = ""
			case "backspace":
				if len(m.filterInput) > 0 {
					m.filterInput = m.filterInput[:len(m.filterInput)-1]
				}
			default:
				// Add character to filter input
				if len(msg.String()) == 1 {
					m.filterInput += msg.String()
				}
			}
			return m, nil
		}

		// Normal keyboard handling
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Clear all filters
			m.filterIP = ""
			m.filterStatus = 0
			m.filterPath = ""
			m.applyFilters()
		case " ":
			// Toggle pause
			wasPaused := m.paused
			m.paused = !m.paused
			// If unpausing, restart the tick
			if wasPaused && !m.paused {
				return m, tickCmd(m.refreshRate)
			}
		case "i":
			// Enter IP filter input mode
			m.filterInputMode = true
			m.filterInput = ""
		case "2":
			// Toggle 2xx filter
			if m.filterStatus == 2 {
				m.filterStatus = 0
			} else {
				m.filterStatus = 2
			}
			m.applyFilters()
		case "3":
			// Toggle 3xx filter
			if m.filterStatus == 3 {
				m.filterStatus = 0
			} else {
				m.filterStatus = 3
			}
			m.applyFilters()
		case "4":
			// Toggle 4xx filter
			if m.filterStatus == 4 {
				m.filterStatus = 0
			} else {
				m.filterStatus = 4
			}
			m.applyFilters()
		case "5":
			// Toggle 5xx filter
			if m.filterStatus == 5 {
				m.filterStatus = 0
			} else {
				m.filterStatus = 5
			}
			m.applyFilters()
		case "+", "=":
			// Decrease refresh interval (faster updates)
			m.refreshRate = m.refreshRate / 2
			if m.refreshRate < 100*time.Millisecond {
				m.refreshRate = 100 * time.Millisecond
			}
		case "-", "_":
			// Increase refresh interval (slower updates)
			m.refreshRate = m.refreshRate * 2
			if m.refreshRate > 10*time.Second {
				m.refreshRate = 10 * time.Second
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		// Only schedule next tick if not paused
		if !m.paused {
			return m, tickCmd(m.refreshRate)
		}
		return m, nil

	case logLineMsg:
		// Only process new log lines if not paused
		if !m.paused {
			// Parse the log line
			if visitor := parser.Parse(string(msg)); visitor != nil {
				// Add to ALL visitors list (keep last 100)
				m.allVisitors = append([]parser.Visitor{*visitor}, m.allVisitors...)
				if len(m.allVisitors) > 100 {
					m.allVisitors = m.allVisitors[:100]
				}

				// Apply filters to update the displayed visitors
				m.applyFilters()

				// Update stats (on unfiltered data)
				m.uniqueIPs[visitor.IP]++
				m.statusCodes[visitor.Status]++
				m.topPaths[visitor.Path]++
				m.methods[visitor.Method]++

				// Extract browser from user agent
				agent := extractBrowser(visitor.Agent)
				m.userAgents[agent]++

				// Lookup geolocation
				if m.geoLocator != nil {
					if loc, err := m.geoLocator.Lookup(visitor.IP); err == nil && loc != nil {
						if loc.Country != "Unknown" && loc.Country != "??" {
							m.countries[loc.Country]++
						}
					}
				}
			}
		}
		// Continue reading lines even when paused (just don't process them)
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
		primaryColor   = lipgloss.Color("86")  // Cyan
		secondaryColor = lipgloss.Color("213") // Pink
		successColor   = lipgloss.Color("46")  // Green
		warningColor   = lipgloss.Color("220") // Yellow
		errorColor     = lipgloss.Color("196") // Red
		textColor      = lipgloss.Color("252") // Light gray
		dimColor       = lipgloss.Color("241") // Dark gray
		borderColor    = lipgloss.Color("240") // Border gray
		accentColor    = lipgloss.Color("117") // Light blue
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
	// For 3-panel row, account for borders (2 chars per panel) and gaps (2 chars between panels)
	smallPanelWidth := (m.width - 10) / 3
	if smallPanelWidth < 25 {
		smallPanelWidth = 25
	}

	// Header
	uptime := time.Since(m.startTime).Round(time.Second)
	refreshMs := m.refreshRate.Milliseconds()

	statusText := ""
	if m.paused {
		statusText = lipgloss.NewStyle().Foreground(warningColor).Bold(true).Render(" ⏸ PAUSED")
	}

	// Build filter status text
	var filterTexts []string
	if m.filterIP != "" {
		filterTexts = append(filterTexts, fmt.Sprintf("IP:%s", m.filterIP))
	}
	if m.filterStatus != 0 {
		filterTexts = append(filterTexts, fmt.Sprintf("Status:%dxx", m.filterStatus))
	}
	if m.filterPath != "" {
		filterTexts = append(filterTexts, fmt.Sprintf("Path:%s", m.filterPath))
	}

	filterStatus := ""
	if len(filterTexts) > 0 {
		filterStatus = lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render(
			fmt.Sprintf(" 🔍 Filters: %s", strings.Join(filterTexts, ", ")),
		)
	}

	helpText := "Press 'q' to quit, 'space' to pause, 'i' for IP filter, '2-5' for status filter, 'esc' to clear filters"
	if m.filterInputMode {
		helpText = fmt.Sprintf("Filter by IP: %s_ (Enter to apply, Esc to cancel)", m.filterInput)
	}

	header := titleStyle.Render("🚀 TAILNGINX DASHBOARD") + statusText + filterStatus + "\n" +
		lipgloss.NewStyle().Foreground(dimColor).Render(
			fmt.Sprintf("Running: %s  •  Refresh: %dms  •  %s", uptime, refreshMs, helpText),
		)

	// Overview metrics panel
	totalText := fmt.Sprintf("%d", len(m.allVisitors))
	if len(filterTexts) > 0 {
		totalText = fmt.Sprintf("%d/%d", len(m.visitors), len(m.allVisitors))
	}

	overviewContent := headerStyle.Render("📊 Overview") + "\n" +
		metricLabelStyle.Render("Total Requests") + metricValueStyle.Render(totalText) + "\n" +
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

	browsersPanel := panelStyle.Width(smallPanelWidth).Render(browsersContent)

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

	// Countries panel (full width)
	countriesContent := headerStyle.Render("🌍 Countries") + "\n"
	if m.geoLocator == nil {
		countriesContent += lipgloss.NewStyle().Foreground(dimColor).Render("  Geolocation disabled")
	} else if len(m.countries) == 0 {
		countriesContent += lipgloss.NewStyle().Foreground(dimColor).Render("  Waiting for visitor data...")
	} else {
		countries := sortMapByValue(m.countries)
		for i, kv := range countries {
			if i >= 6 {
				break
			}
			count := lipgloss.NewStyle().Foreground(secondaryColor).Render(fmt.Sprintf("%3d", kv.value))
			countriesContent += fmt.Sprintf("%s  %s\n", count, statLabelStyle.Render(kv.key))
		}
	}

	countriesPanel := panelStyle.Width(m.width - 4).Render(countriesContent)

	// Recent requests panel (full width)
	recentContent := headerStyle.Render("📝 Recent Activity") + "\n"

	if len(m.visitors) == 0 {
		recentContent += lipgloss.NewStyle().Foreground(dimColor).Render("  Waiting for log entries...")
	} else {
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
	}

	recentPanel := panelStyle.Width(m.width - 4).Render(recentContent)

	// Assemble the dashboard
	dashboard := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		topRow,
		middleRow,
		lowerMiddleRow,
		countriesPanel,
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

// tickCmd returns a command that ticks at the specified interval
func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
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

// applyFilters filters the allVisitors list based on active filters
func (m *Model) applyFilters() {
	m.visitors = []parser.Visitor{}

	for _, v := range m.allVisitors {
		// Apply IP filter
		if m.filterIP != "" && !strings.Contains(v.IP, m.filterIP) {
			continue
		}

		// Apply status code filter
		if m.filterStatus != 0 {
			statusClass := v.Status / 100
			if statusClass != m.filterStatus {
				continue
			}
		}

		// Apply path filter
		if m.filterPath != "" && !strings.Contains(v.Path, m.filterPath) {
			continue
		}

		// Passed all filters
		m.visitors = append(m.visitors, v)
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
