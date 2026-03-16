package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"es-spectre/internal/config"
	"es-spectre/internal/model"
	"es-spectre/internal/repository/adapters"
	"es-spectre/internal/service"
	"es-spectre/internal/ui"
	"es-spectre/internal/ui/pages"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateSplash state = iota
	stateExplore
	stateMap
	stateDashboard
	stateError
)

type appModel struct {
	state    state
	config   *config.Config
	esClient *service.ESClient
	dictRepo *adapters.GenericAdapter

	// Page Models
	explorer  pages.ExplorerModel
	mapper    pages.MappingLabModel
	dashboard pages.DashboardModel

	err           error
	loading       float64
	logLines      []string
	width, height int
}

func initialModel() appModel {
	return appModel{
		state:    stateSplash,
		loading:  0,
		logLines: []string{"[WAIT] Initializing ES-Spectre..."},
	}
}

type tickMsg time.Time
type initFinishedMsg struct {
	cfg      *config.Config
	esClient *service.ESClient
	dictRepo *adapters.GenericAdapter
	fields   []string
	err      error
}

func (m appModel) Init() tea.Cmd {
	return tea.Batch(
		tick(),
		runInitCmd(),
	)
}

func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func runInitCmd() tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.LoadConfig("configs/config.yaml")
		if err != nil {
			return initFinishedMsg{err: fmt.Errorf("Config Error: %w", err)}
		}

		esClient, err := service.NewESClient(cfg.Elasticsearch.Address)
		if err != nil {
			return initFinishedMsg{err: fmt.Errorf("ES Connection Error: %w", err)}
		}

		connStr := cfg.Database.ConnUrl
		if connStr == "" {
			switch cfg.Database.Type {
			case "pg", "kingbase", "highgo", "vastbase":
				connStr = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.DBName)
				if cfg.Database.Schema != "" {
					connStr += fmt.Sprintf(" search_path=%s", cfg.Database.Schema)
				}
			default:
				// 兼容处理：MySQL/MariaDB 可能将库名填在 schema 或 dbname 中
				dbIdent := cfg.Database.DBName
				if (cfg.Database.Type == "mariadb" || cfg.Database.Type == "mysql") && dbIdent == "" {
					dbIdent = cfg.Database.Schema
				}
				connStr = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, dbIdent)
			}
		}
		dictRepo, err := adapters.NewGenericAdapter(cfg.Database.Type, connStr, cfg.Database.Schema, "", cfg.Database.DriverClass)
		if err != nil {
			// Even if DB fails, we allow continuation in DEMO MODE for ES exploration
			dictRepo = nil
		}

		// 4. Fetch Fields
		fields, _ := esClient.GetFields(context.Background(), cfg.Elasticsearch.Index)
		if len(fields) == 0 {
			fields = []string{"No Fields Found", "Check Config"}
		}

		return initFinishedMsg{cfg: cfg, esClient: esClient, dictRepo: dictRepo, fields: fields, err: err}
	}
}

type nextStateMsg struct{}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case tickMsg:
		if m.state == stateSplash && m.loading < 1.0 {
			m.loading += 0.05
			return m, tick()
		}

	case initFinishedMsg:
		m.config = msg.cfg
		m.esClient = msg.esClient
		m.dictRepo = msg.dictRepo

		if msg.err != nil {
			m.err = msg.err
			m.logLines = append(m.logLines, fmt.Sprintf("[WARN] Init Issues: %v", msg.err))
			m.logLines = append(m.logLines, "Entering DEMO MODE (Press Enter to continue)...")
		} else {
			m.logLines = append(m.logLines, fmt.Sprintf("[OK] Connected to ES (%d fields found).", len(msg.fields)))
		}

		m.explorer = pages.NewExplorer(msg.fields)
		return m, nil
	}

	// Route updates to sub-models
	switch m.state {
	case stateSplash:
		if m.loading >= 1.0 {
			// In splash, wait for Enter if there was a warning/error,
			// or auto-transition if everything is OK.
			if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
				m.state = stateExplore
			} else if m.err == nil && m.config != nil {
				// Auto transition only if absolutely successful
				m.state = stateExplore
			}
		}

	case stateExplore:
		var newExplorer tea.Model
		newExplorer, cmd = m.explorer.Update(msg)
		m.explorer = newExplorer.(pages.ExplorerModel)

		// Check for transition (Enter key in explorer)
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
			selected := m.explorer.GetSelectedFields()
			mappings := []model.FieldMapping{}
			for _, f := range selected {
				mappings = append(mappings, model.FieldMapping{
					FieldName: f,
					Status:    model.StatusUnmapped,
				})
			}
			m.mapper = pages.NewMappingLab(mappings)
			m.state = stateMap
		}

	case stateMap:
		// Check if we should go back
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
			m.state = stateExplore
			return m, nil
		}
		var newMapper tea.Model
		newMapper, cmd = m.mapper.Update(msg)
		m.mapper = newMapper.(pages.MappingLabModel)

		// Reserved for future real-time save or transition

	case stateDashboard:
		var newDash tea.Model
		newDash, cmd = m.dashboard.Update(msg)
		m.dashboard = newDash.(pages.DashboardModel)
	}

	return m, cmd
}

func (m appModel) View() string {
	switch m.state {
	case stateSplash:
		logo := ui.TitleStyle.Render("ES-SPECTRE")
		progressBar := fmt.Sprintf("[%s%s] %.0f%%",
			strings.Repeat("█", int(m.loading*20)),
			strings.Repeat(" ", 20-int(m.loading*20)),
			m.loading*100)

		logs := ""
		for _, line := range m.logLines {
			logs += line + "\n"
		}

		content := lipgloss.JoinVertical(lipgloss.Center,
			logo,
			"\n",
			progressBar,
			"\n",
			ui.MainStyle.Render(logs),
		)
		return ui.MainStyle.Render(content)

	case stateExplore:
		return lipgloss.JoinVertical(lipgloss.Top,
			m.explorer.View(),
			ui.MutedStyle.Render("\n [Space] Select  [Enter] Map Fields  [q] Quit"),
		)

	case stateMap:
		return lipgloss.JoinVertical(lipgloss.Top,
			m.mapper.View(),
			ui.MutedStyle.Render("\n [Enter] Manual Map  [Esc] Back  [q] Quit"),
		)

	case stateDashboard:
		return m.dashboard.View()

	case stateError:
		return ui.MainStyle.Render(fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err))
	}
	return "Unknown state"
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
