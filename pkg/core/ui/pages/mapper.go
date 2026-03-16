package pages

import (
	"es-spectre/pkg/core/model"
	"es-spectre/pkg/core/ui"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MappingLabModel struct {
	list          list.Model
	mappings      []model.FieldMapping
	width, height int
	selectedIdx   int
	showModal     bool
}

type mappingItem struct {
	mapping model.FieldMapping
}

func (i mappingItem) Title() string {
	status := "[ ]"
	style := ui.Muted
	label := i.mapping.FieldName
	
	if i.mapping.SampleText != "" {
		label = fmt.Sprintf("%s (%s)", i.mapping.FieldName, i.mapping.SampleText)
	}

	switch i.mapping.Status {
	case model.StatusAutoMatched:
		status = "[âœ”]"
		style = ui.Match
	case model.StatusUnmapped:
		status = "[!]"
		style = ui.Missing
	case model.StatusManualMapped:
		status = "[M]"
		style = ui.Action
	}
	
	return lipgloss.NewStyle().Foreground(style).Render(fmt.Sprintf("%s %s", status, label))
}

func (i mappingItem) Description() string {
	if i.mapping.DictCode != "" {
		return fmt.Sprintf("Mapped to: %s", i.mapping.DictCode)
	}
	return "No dictionary association yet"
}

func (i mappingItem) FilterValue() string { return i.mapping.FieldName }

func NewMappingLab(mappings []model.FieldMapping) MappingLabModel {
	items := make([]list.Item, len(mappings))
	for i, m := range mappings {
		items[i] = mappingItem{mapping: m}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Mapping Lab: Verify Field Meanings"
	l.SetShowStatusBar(false)

	return MappingLabModel{
		list:     l,
		mappings: mappings,
	}
}

func (m MappingLabModel) Init() tea.Cmd {
	return nil
}

func (m MappingLabModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-6)

	case tea.KeyMsg:
		if m.showModal {
			switch msg.String() {
			case "esc":
				m.showModal = false
				return m, nil
			case "enter":
				// Logic to confirm manual mapping would go here
				m.showModal = false
				return m, nil
			}
		} else {
			switch msg.String() {
			case "enter":
				m.showModal = true
				return m, nil
			}
		}
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m MappingLabModel) View() string {
	listView := m.list.View()

	if m.showModal {
		// Simple overlay modal
		modal := lipgloss.NewStyle().
			Width(40).
			Height(10).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.Accent).
			Padding(1, 2).
			Render("Manual Mapping\n\nEnter Dict Code:\n\n[ __________ ]\n\n(Esc to cancel)")
		
		// Center the modal
		return ui.MainStyle.Render(
			lipgloss.Place(m.width, m.height,
				lipgloss.Center, lipgloss.Center,
				modal,
				lipgloss.WithWhitespaceChars("â–?),
				lipgloss.WithWhitespaceForeground(ui.Muted),
			),
		)
	}

	return ui.MainStyle.Render(listView)
}
