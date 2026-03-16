package pages

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type fieldItem struct {
	name      string
	selected  bool
	fieldType string
}

func (i fieldItem) Title() string       { return i.name }
func (i fieldItem) Description() string { return i.fieldType }
func (i fieldItem) FilterValue() string { return i.name }

type ExplorerModel struct {
	list   list.Model
	search textinput.Model
	fields []fieldItem
	width  int
	height int
}

func NewExplorer(fields []string) ExplorerModel {
	items := make([]list.Item, len(fields))
	for i, f := range fields {
		items[i] = fieldItem{name: f, fieldType: "keyword"}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select Fields for Aggregation"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	return ExplorerModel{
		list: l,
	}
}

func (m ExplorerModel) Init() tea.Cmd {
	return nil
}

func (m ExplorerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-6)

	case tea.KeyMsg:
		switch msg.String() {
		case " ":
			if i, ok := m.list.SelectedItem().(fieldItem); ok {
				i.selected = !i.selected
				m.list.SetItem(m.list.Index(), i)
			}
		case "enter":
			// Process selection and move to next state
			return m, nil
		}
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ExplorerModel) GetSelectedFields() []string {
	var results []string
	for _, item := range m.list.Items() {
		if fi, ok := item.(fieldItem); ok && fi.selected {
			results = append(results, fi.name)
		}
	}
	return results
}

func (m ExplorerModel) View() string {
	return lipgloss.NewStyle().Padding(1, 2).Render(m.list.View())
}
