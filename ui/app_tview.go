// Package ui provides the tview-based terminal user interface for tailnginx.
package ui

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/papaganelli/tailnginx/pkg/geoip"
	"github.com/papaganelli/tailnginx/pkg/parser"
	"github.com/rivo/tview"
)

// TviewApp represents the tview-based application.
type TviewApp struct {
	app      *tview.Application
	grid     *tview.Grid
	mu       sync.RWMutex

	// Panels
	overview       *tview.TextView
	statusTable    *tview.Table
	pathsTable     *tview.Table
	visitorsTable  *tview.Table
	clientsTable   *tview.Table
	methodsTable   *tview.Table
	countriesTable *tview.Table
	referersTable  *tview.Table
	logStream      *tview.TextView

	// Data
	lines           <-chan string
	visitors        []parser.Visitor
	allVisitors     []parser.Visitor
	statusCodes     map[int]int
	pathsData       map[string]int
	ips             map[string]int
	userAgents      map[string]int
	methodsData     map[string]int
	countriesData   map[string]int
	referersData    map[string]int
	logLines        []string
	startTime       time.Time
	paused          bool
	refreshRate     time.Duration
	statusFilter    int
	geoLocator      *geoip.Locator
}

// NewTviewApp creates a new tview-based application.
func NewTviewApp(lines <-chan string, refreshRate time.Duration, geoLocator *geoip.Locator) *TviewApp {
	ta := &TviewApp{
		app:           tview.NewApplication(),
		lines:         lines,
		visitors:      []parser.Visitor{},
		allVisitors:   []parser.Visitor{},
		statusCodes:   make(map[int]int),
		pathsData:     make(map[string]int),
		ips:           make(map[string]int),
		userAgents:    make(map[string]int),
		methodsData:   make(map[string]int),
		countriesData: make(map[string]int),
		referersData:  make(map[string]int),
		logLines:      make([]string, 0),
		startTime:     time.Now(),
		refreshRate:   refreshRate,
		geoLocator:    geoLocator,
	}

	ta.initUI()
	return ta
}

// initUI initializes the tview UI components.
func (ta *TviewApp) initUI() {
	// Define elegant color scheme
	borderColor := tcell.NewRGBColor(75, 85, 99)    // Gray 600
	titleColor := tcell.NewRGBColor(139, 92, 246)   // Purple
	headerBg := tcell.NewRGBColor(31, 41, 55)       // Gray 800

	// Create panels with borders and titles
	ta.overview = ta.createTextView("📊 Overview", borderColor, titleColor)
	ta.statusTable = ta.createTable("📡 HTTP Status", borderColor, titleColor)
	ta.pathsTable = ta.createTable("🔥 Top Paths", borderColor, titleColor)
	ta.visitorsTable = ta.createTable("👥 Visitors", borderColor, titleColor)
	ta.clientsTable = ta.createTable("🌐 Clients", borderColor, titleColor)
	ta.methodsTable = ta.createTable("🔧 Methods", borderColor, titleColor)
	ta.countriesTable = ta.createTable("🌍 Countries", borderColor, titleColor)
	ta.referersTable = ta.createTable("🔗 Sources", borderColor, titleColor)
	ta.logStream = ta.createTextView("📝 Live Stream", borderColor, titleColor)

	// Create header
	header := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[white::b] TAILNGINX [-::-] Nginx Log Monitor")
	header.SetBackgroundColor(headerBg)

	// Create footer with help text
	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[yellow]q[-::-]:quit  [yellow]space[-::-]:pause  [yellow]±[-::-]:speed  [yellow]2-5[-::-]:filter  [yellow]esc[-::-]:clear")
	footer.SetBackgroundColor(headerBg)

	// Create main grid layout
	ta.grid = tview.NewGrid().
		SetRows(1, 0, 1).    // header, content, footer
		SetColumns(0).       // full width
		SetBorders(false)

	// Create content grid - responsive 3-column layout
	content := tview.NewGrid().
		SetRows(4, 0, 0).        // overview (smaller), middle row, bottom row
		SetColumns(0, 0, 0).     // 3 equal columns
		SetBorders(true)

	// Row 1: Overview spans all columns
	content.AddItem(ta.overview, 0, 0, 1, 3, 0, 0, false)

	// Row 2: Status, Paths, Methods (3 columns)
	content.AddItem(ta.statusTable, 1, 0, 1, 1, 0, 0, false)
	content.AddItem(ta.pathsTable, 1, 1, 1, 1, 0, 0, false)
	content.AddItem(ta.methodsTable, 1, 2, 1, 1, 0, 0, false)

	// Row 3: Bottom section with 2 rows
	bottomGrid := tview.NewGrid().
		SetRows(0, 0).           // 2 equal rows
		SetColumns(0, 0, 0).     // 3 equal columns
		SetBorders(true)

	// Bottom row 1: Visitors, Clients, Countries
	bottomGrid.AddItem(ta.visitorsTable, 0, 0, 1, 1, 0, 0, false)
	bottomGrid.AddItem(ta.clientsTable, 0, 1, 1, 1, 0, 0, false)
	bottomGrid.AddItem(ta.countriesTable, 0, 2, 1, 1, 0, 0, false)

	// Bottom row 2: Referers (1 col) + Log Stream (2 cols)
	bottomGrid.AddItem(ta.referersTable, 1, 0, 1, 1, 0, 0, false)
	bottomGrid.AddItem(ta.logStream, 1, 1, 1, 2, 0, 0, false)

	content.AddItem(bottomGrid, 2, 0, 1, 3, 0, 0, false)

	// Add all to main grid
	ta.grid.AddItem(header, 0, 0, 1, 1, 0, 0, false)
	ta.grid.AddItem(content, 1, 0, 1, 1, 0, 0, false)
	ta.grid.AddItem(footer, 2, 0, 1, 1, 0, 0, false)

	// Set up key bindings
	ta.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			ta.app.Stop()
			return nil
		case ' ':
			ta.mu.Lock()
			ta.paused = !ta.paused
			ta.mu.Unlock()
		case '+', '=':
			ta.mu.Lock()
			if ta.refreshRate > 100*time.Millisecond {
				ta.refreshRate -= 100 * time.Millisecond
			}
			ta.mu.Unlock()
		case '-', '_':
			ta.mu.Lock()
			if ta.refreshRate < 5*time.Second {
				ta.refreshRate += 100 * time.Millisecond
			}
			ta.mu.Unlock()
		case '2':
			ta.mu.Lock()
			ta.statusFilter = 2
			ta.applyFilters()
			ta.mu.Unlock()
		case '3':
			ta.mu.Lock()
			ta.statusFilter = 3
			ta.applyFilters()
			ta.mu.Unlock()
		case '4':
			ta.mu.Lock()
			ta.statusFilter = 4
			ta.applyFilters()
			ta.mu.Unlock()
		case '5':
			ta.mu.Lock()
			ta.statusFilter = 5
			ta.applyFilters()
			ta.mu.Unlock()
		}
		if event.Key() == tcell.KeyEscape {
			ta.mu.Lock()
			ta.statusFilter = 0
			ta.applyFilters()
			ta.mu.Unlock()
		}
		return event
	})
}

// createTextView creates a bordered text view with title.
func (ta *TviewApp) createTextView(title string, borderColor, titleColor tcell.Color) *tview.TextView {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true).
		SetWordWrap(true)

	tv.SetBorder(true).
		SetTitle(title).
		SetBorderColor(borderColor).
		SetTitleColor(titleColor)

	return tv
}

// createTable creates a bordered table with title.
func (ta *TviewApp) createTable(title string, borderColor, titleColor tcell.Color) *tview.Table {
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(false, false)

	table.SetBorder(true).
		SetTitle(title).
		SetBorderColor(borderColor).
		SetTitleColor(titleColor)

	return table
}

// Run starts the tview application.
func (ta *TviewApp) Run() error {
	// Start log reader goroutine
	go ta.readLines()

	// Start update ticker
	go ta.updateLoop()

	// Set root and run
	return ta.app.SetRoot(ta.grid, true).Run()
}

// readLines reads log lines from the channel.
func (ta *TviewApp) readLines() {
	for line := range ta.lines {
		if v := parser.Parse(line); v != nil {
			// Add country information if available
			if ta.geoLocator != nil {
				if loc, err := ta.geoLocator.Lookup(v.IP); err == nil && loc != nil {
					v.Country = loc.Country
				}
			}

			ta.mu.Lock()
			ta.allVisitors = append(ta.allVisitors, *v)
			// Keep only last 1000 visitors in memory
			if len(ta.allVisitors) > 1000 {
				ta.allVisitors = ta.allVisitors[len(ta.allVisitors)-1000:]
			}
			ta.applyFilters()
			ta.mu.Unlock()
		}
	}
}

// applyFilters filters visitors based on current filters.
func (ta *TviewApp) applyFilters() {
	ta.visitors = nil
	for _, v := range ta.allVisitors {
		if ta.statusFilter > 0 && v.Status/100 != ta.statusFilter {
			continue
		}
		ta.visitors = append(ta.visitors, v)
	}
}

// updateLoop continuously updates the UI.
func (ta *TviewApp) updateLoop() {
	ticker := time.NewTicker(ta.refreshRate)
	defer ticker.Stop()

	for range ticker.C {
		ta.mu.Lock()
		paused := ta.paused
		ta.mu.Unlock()

		if !paused {
			ta.mu.Lock()
			ta.updateData()
			ta.mu.Unlock()

			ta.app.QueueUpdateDraw(func() {
				ta.mu.RLock()
				defer ta.mu.RUnlock()
				ta.renderAll()
			})
		}
	}
}

// updateData updates internal data structures from visitors.
func (ta *TviewApp) updateData() {
	// Reset maps
	ta.statusCodes = make(map[int]int)
	ta.pathsData = make(map[string]int)
	ta.ips = make(map[string]int)
	ta.userAgents = make(map[string]int)
	ta.methodsData = make(map[string]int)
	ta.countriesData = make(map[string]int)
	ta.referersData = make(map[string]int)
	ta.logLines = make([]string, 0)

	for _, v := range ta.visitors {
		ta.statusCodes[v.Status]++
		ta.pathsData[v.Path]++
		ta.ips[v.IP]++

		// Truncate long user agents
		agent := v.Agent
		if len(agent) > 50 {
			agent = agent[:47] + "..."
		}
		ta.userAgents[agent]++

		ta.methodsData[v.Method]++

		if v.Country != "" && v.Country != "Unknown" {
			ta.countriesData[v.Country]++
		}

		if v.Referer != "" && v.Referer != "-" {
			ta.referersData[v.Referer]++
		}

		// Add to log stream (last 15 lines)
		logLine := fmt.Sprintf("[::d]%s[-::-] [yellow]%s[-::-] %s [cyan]%d[-::-]",
			v.Time.Format("15:04:05"),
			v.Method,
			v.Path,
			v.Status)
		ta.logLines = append(ta.logLines, logLine)
	}

	// Keep only last 15 log lines
	if len(ta.logLines) > 15 {
		ta.logLines = ta.logLines[len(ta.logLines)-15:]
	}
}

// renderAll renders all UI components.
func (ta *TviewApp) renderAll() {
	ta.renderOverview()
	ta.renderStatus()
	ta.renderPaths()
	ta.renderVisitors()
	ta.renderClients()
	ta.renderMethods()
	ta.renderCountries()
	ta.renderReferers()
	ta.renderLogStream()
}

// renderOverview renders the overview panel.
func (ta *TviewApp) renderOverview() {
	uptime := time.Since(ta.startTime).Round(time.Second)
	totalRequests := len(ta.visitors)
	totalAll := len(ta.allVisitors)

	status := "[green::b]Running[-::-]"
	if ta.paused {
		status = "[yellow::b]Paused[-::-]"
	}

	filterText := "All"
	if ta.statusFilter > 0 {
		filterText = fmt.Sprintf("[cyan]%dxx[-::-]", ta.statusFilter)
	}

	text := fmt.Sprintf(
		"  [::b]Requests:[-::-] [white]%d[-::-] / [::d]%d[-::-]  •  [::b]Uptime:[-::-] [white]%s[-::-]  •  [::b]Status:[-::-] %s  •  [::b]Refresh:[-::-] [white]%dms[-::-]  •  [::b]Filter:[-::-] %s",
		totalRequests,
		totalAll,
		uptime,
		status,
		ta.refreshRate.Milliseconds(),
		filterText,
	)

	ta.overview.SetText(text)
}

// renderStatus renders the HTTP status codes table.
func (ta *TviewApp) renderStatus() {
	ta.statusTable.Clear()

	type kv struct {
		key   int
		value int
	}

	var sorted []kv
	total := 0
	for k, v := range ta.statusCodes {
		sorted = append(sorted, kv{k, v})
		total += v
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].value > sorted[j].value
	})

	row := 0
	for _, item := range sorted {
		if row >= 10 {
			break
		}

		percentage := float64(item.value) / float64(total) * 100

		// Color based on status code
		color := "green"
		symbol := "✓"
		if item.key >= 300 && item.key < 400 {
			color = "blue"
			symbol = "↻"
		} else if item.key >= 400 && item.key < 500 {
			color = "yellow"
			symbol = "⚠"
		} else if item.key >= 500 {
			color = "red"
			symbol = "✗"
		}

		// Create filled bar (12 characters wide)
		barWidth := 12
		filledWidth := int(percentage * float64(barWidth) / 100)
		if filledWidth > barWidth {
			filledWidth = barWidth
		}
		bar := fmt.Sprintf("[%s]%s[-::-][::d]%s[-::-]",
			color,
			strings.Repeat("█", filledWidth),
			strings.Repeat("░", barWidth-filledWidth))

		ta.statusTable.SetCell(row, 0,
			tview.NewTableCell(fmt.Sprintf("[%s]%s %d[-::-]", color, symbol, item.key)).
				SetAlign(tview.AlignLeft))
		ta.statusTable.SetCell(row, 1,
			tview.NewTableCell(bar).
				SetAlign(tview.AlignLeft))
		ta.statusTable.SetCell(row, 2,
			tview.NewTableCell(fmt.Sprintf("[cyan::b]%.0f%%[-::-]", percentage)).
				SetAlign(tview.AlignRight))

		row++
	}
}

// renderPaths renders the top paths table.
func (ta *TviewApp) renderPaths() {
	ta.pathsTable.Clear()
	ta.renderTopN(ta.pathsTable, ta.pathsData, 10)
}

// renderVisitors renders the top visitors table.
func (ta *TviewApp) renderVisitors() {
	ta.visitorsTable.Clear()
	ta.renderTopN(ta.visitorsTable, ta.ips, 10)
}

// renderClients renders the top clients table.
func (ta *TviewApp) renderClients() {
	ta.clientsTable.Clear()
	ta.renderTopN(ta.clientsTable, ta.userAgents, 10)
}

// renderMethods renders the HTTP methods table.
func (ta *TviewApp) renderMethods() {
	ta.methodsTable.Clear()
	ta.renderTopN(ta.methodsTable, ta.methodsData, 10)
}

// renderCountries renders the top countries table.
func (ta *TviewApp) renderCountries() {
	ta.countriesTable.Clear()

	type kv struct {
		key   string
		value int
	}

	var sorted []kv
	for k, v := range ta.countriesData {
		sorted = append(sorted, kv{k, v})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].value > sorted[j].value
	})

	row := 0
	for _, item := range sorted {
		if row >= 10 {
			break
		}

		// Get country name - display as "US - United States"
		countryName := getCountryName(item.key)
		displayText := fmt.Sprintf("[yellow::b]%s[-::-] %s", item.key, countryName)

		ta.countriesTable.SetCell(row, 0,
			tview.NewTableCell(displayText).
				SetAlign(tview.AlignLeft))
		ta.countriesTable.SetCell(row, 1,
			tview.NewTableCell(fmt.Sprintf("[cyan]%d[-::-]", item.value)).
				SetAlign(tview.AlignRight))

		row++
	}
}

// renderReferers renders the top referers table.
func (ta *TviewApp) renderReferers() {
	ta.referersTable.Clear()
	ta.renderTopN(ta.referersTable, ta.referersData, 10)
}

// renderTopN is a helper to render top N items from a map.
func (ta *TviewApp) renderTopN(table *tview.Table, data map[string]int, limit int) {
	type kv struct {
		key   string
		value int
	}

	var sorted []kv
	for k, v := range data {
		sorted = append(sorted, kv{k, v})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].value > sorted[j].value
	})

	row := 0
	for _, item := range sorted {
		if row >= limit {
			break
		}

		// Truncate long strings
		key := item.key
		if len(key) > 40 {
			key = key[:37] + "..."
		}

		table.SetCell(row, 0,
			tview.NewTableCell(fmt.Sprintf("[white]%s[-::-]", key)).
				SetAlign(tview.AlignLeft).
				SetMaxWidth(40))
		table.SetCell(row, 1,
			tview.NewTableCell(fmt.Sprintf("[cyan]%d[-::-]", item.value)).
				SetAlign(tview.AlignRight))

		row++
	}
}

// renderLogStream renders the live log stream.
func (ta *TviewApp) renderLogStream() {
	var b strings.Builder
	for _, line := range ta.logLines {
		b.WriteString(line)
		b.WriteString("\n")
	}
	ta.logStream.SetText(b.String())
	ta.logStream.ScrollToEnd()
}

// countryCodeToName maps 2-letter country codes to full names
var countryCodeToName = map[string]string{
	"US": "United States", "GB": "United Kingdom", "DE": "Germany",
	"FR": "France", "JP": "Japan", "CN": "China", "IN": "India",
	"BR": "Brazil", "RU": "Russia", "CA": "Canada", "AU": "Australia",
	"ES": "Spain", "IT": "Italy", "NL": "Netherlands", "SE": "Sweden",
	"NO": "Norway", "DK": "Denmark", "FI": "Finland", "PL": "Poland",
	"BE": "Belgium", "CH": "Switzerland", "AT": "Austria", "PT": "Portugal",
	"IE": "Ireland", "GR": "Greece", "CZ": "Czech Republic", "RO": "Romania",
	"HU": "Hungary", "MX": "Mexico", "AR": "Argentina", "CL": "Chile",
	"CO": "Colombia", "ZA": "South Africa", "KR": "South Korea", "SG": "Singapore",
	"HK": "Hong Kong", "TW": "Taiwan", "TH": "Thailand", "MY": "Malaysia",
	"ID": "Indonesia", "PH": "Philippines", "VN": "Vietnam", "NZ": "New Zealand",
	"TR": "Turkey", "IL": "Israel", "AE": "United Arab Emirates", "SA": "Saudi Arabia",
	"EG": "Egypt", "NG": "Nigeria", "KE": "Kenya", "UA": "Ukraine",
}

// countryCodeToFlag maps 2-letter country codes to flag emojis
var countryCodeToFlag = map[string]string{
	"US": "🇺🇸", "GB": "🇬🇧", "DE": "🇩🇪", "FR": "🇫🇷", "JP": "🇯🇵",
	"CN": "🇨🇳", "IN": "🇮🇳", "BR": "🇧🇷", "RU": "🇷🇺", "CA": "🇨🇦",
	"AU": "🇦🇺", "ES": "🇪🇸", "IT": "🇮🇹", "NL": "🇳🇱", "SE": "🇸🇪",
	"NO": "🇳🇴", "DK": "🇩🇰", "FI": "🇫🇮", "PL": "🇵🇱", "BE": "🇧🇪",
	"CH": "🇨🇭", "AT": "🇦🇹", "PT": "🇵🇹", "IE": "🇮🇪", "GR": "🇬🇷",
	"CZ": "🇨🇿", "RO": "🇷🇴", "HU": "🇭🇺", "MX": "🇲🇽", "AR": "🇦🇷",
	"CL": "🇨🇱", "CO": "🇨🇴", "ZA": "🇿🇦", "KR": "🇰🇷", "SG": "🇸🇬",
	"HK": "🇭🇰", "TW": "🇹🇼", "TH": "🇹🇭", "MY": "🇲🇾", "ID": "🇮🇩",
	"PH": "🇵🇭", "VN": "🇻🇳", "NZ": "🇳🇿", "TR": "🇹🇷", "IL": "🇮🇱",
	"AE": "🇦🇪", "SA": "🇸🇦", "EG": "🇪🇬", "NG": "🇳🇬", "KE": "🇰🇪",
	"UA": "🇺🇦",
}

// getCountryFlag returns the flag emoji for a country code.
func getCountryFlag(countryCode string) string {
	if flag, ok := countryCodeToFlag[countryCode]; ok {
		return flag
	}
	return "🌍" // Default globe emoji for unknown countries
}

// getCountryName returns the full country name for a country code.
func getCountryName(countryCode string) string {
	if name, ok := countryCodeToName[countryCode]; ok {
		return name
	}
	return countryCode // Return code if name not found
}
