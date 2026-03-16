package pages

import (
	"es-spectre/pkg/core/ui"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type RowData struct {
	Level      int
	Label      string
	Count      int
	Percentage float64
	IsLast     bool
}

type DashboardModel struct {
	rows          []RowData
	cursor        int
	width, height int
}

func NewDashboard(data []RowData) DashboardModel {
	return DashboardModel{
		rows: data,
	}
}

func (m DashboardModel) Init() tea.Cmd {
	return nil
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	}
	return m, nil
}

func (m DashboardModel) renderProgressBar(percent float64) string {
	width := 20
	filled := int(percent / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	
	bar := strings.Repeat("‚ñ?, filled) + strings.Repeat("‚ñ?, width-filled)
	
	// Gradient effect via colors based on percentage
	color := ui.Match
	if percent < 30 {
		color = ui.Accent
	} else if percent > 70 {
		color = ui.Missing
	}
	
	return lipgloss.NewStyle().Foreground(color).Render(bar)
}

func (m DashboardModel) View() string {
	header := ui.TitleStyle.Render("Û∞ö≠ Ghost Grid: Analysis Dashboard") + "\n"
	
	var tableBody strings.Builder
	for i, row := range m.rows {
		cursor := "  "
		if m.cursor == i {
			cursor = lipgloss.NewStyle().Foreground(ui.Accent).Render("¬ª ")
		}

		indent := strings.Repeat("  ", row.Level)
		prefix := ""
		if row.Level > 0 {
			prefix = "‚î£‚îÅ "
			if row.IsLast {
				prefix = "‚îó‚îÅ "
			}
		}

		labelStyle := lipgloss.NewStyle()
		if row.Level == 0 {
			labelStyle = labelStyle.Bold(true).Foreground(ui.Action)
		}

		rowText := fmt.Sprintf("%s%s%s%s", 
			cursor, 
			indent, 
			prefix, 
			labelStyle.Render(row.Label))
		
		stats := fmt.Sprintf(" [%d Ê¨? %.1f%%] ", row.Count, row.Percentage)
		bar := m.renderProgressBar(row.Percentage)

		// Padding label to align bars
		labelPadding := 40 - lipgloss.Width(rowText)
		if labelPadding < 0 { labelPadding = 0 }
		
		tableBody.WriteString(rowText + strings.Repeat(" ", labelPadding) + stats + bar + "\n")
	}

	footer := lipgloss.NewStyle().
		Foreground(ui.Muted).
		MarginTop(1).
		Render("‚Ü?‚Ü? Scroll ‚Ä?Ctrl+E: Export ‚Ä?q: Quit")

	content := header + "\n" + tableBody.String() + "\n" + footer
	return ui.MainStyle.Render(content)
}
