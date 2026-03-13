package ui

import (
	"context"
	"fmt"

	"github.com/tardanoir/seshat/internal/config"
	"github.com/tardanoir/seshat/internal/db"
	"github.com/tardanoir/seshat/internal/editor"
	"github.com/tardanoir/seshat/internal/query"
	"github.com/tardanoir/seshat/internal/ui/modal"
	"github.com/tardanoir/seshat/internal/ui/queryeditor"
	"github.com/tardanoir/seshat/internal/ui/resultstable"
	"github.com/tardanoir/seshat/internal/ui/sidebar"
	"github.com/tardanoir/seshat/internal/ui/statusbar"
	"github.com/tardanoir/seshat/internal/ui/style"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Focus int

const (
	FocusSidebar Focus = iota
	FocusPreview
	FocusResults
)

type ModalState int

const (
	ModalNone ModalState = iota
	ModalConnection
	ModalSave
	ModalTemplatePicker
	ModalTemplateVars
	ModalConfirm
)

// Messages
type QueryResultMsg struct{ Result *db.QueryResult }
type QueryErrorMsg struct{ Err error }
type QueriesLoadedMsg struct{ Queries []query.SavedQuery }
type TemplatesLoadedMsg struct{ Templates []query.Template }
type ConnectedMsg struct {
	DB   *db.DB
	Name string
	Conn config.Connection
}
type ConnectErrorMsg struct{ Err error }
type TablesLoadedMsg struct{ Tables []sidebar.TableEntry }
type ColumnsLoadedMsg struct {
	Schema    string
	TableName string
	Columns   []sidebar.ColumnDef
}

type App struct {
	cfg    *config.Config
	db     *db.DB
	cancel context.CancelFunc

	connName string

	sidebar sidebar.Model
	preview queryeditor.Model
	results resultstable.Model
	status  statusbar.Model

	modalState     ModalState
	connModal      modal.ConnectionModel
	saveModal      modal.SaveModel
	templatePicker modal.TemplatePickerModel
	templateVars   modal.TemplateVarsModel
	confirmModal   modal.ConfirmModel
	deleteTarget   string

	focus          Focus
	sidebarVisible bool
	width          int
	height         int
	ready          bool
}

func NewApp(cfg *config.Config) App {
	s := sidebar.New()
	p := queryeditor.New()
	r := resultstable.New()
	st := statusbar.New()

	return App{
		cfg:            cfg,
		sidebar:        s,
		preview:        p,
		results:        r,
		status:         st,
		focus:          FocusPreview,
		sidebarVisible: true,
		connName:       cfg.DefaultConnection,
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.connectCmd(a.cfg.DefaultConnection),
		a.loadQueriesCmd(),
		a.loadTemplatesCmd(),
	)
}

func (a *App) connectCmd(name string) tea.Cmd {
	conn, ok := a.cfg.Connections[name]
	if !ok {
		return func() tea.Msg {
			return ConnectErrorMsg{Err: fmt.Errorf("connection %q not found", name)}
		}
	}
	connStr := conn.ConnString()
	return func() tea.Msg {
		ctx := context.Background()
		d, err := db.Connect(ctx, connStr, name)
		if err != nil {
			return ConnectErrorMsg{Err: err}
		}
		return ConnectedMsg{DB: d, Name: name, Conn: conn}
	}
}

func (a *App) loadQueriesCmd() tea.Cmd {
	return func() tea.Msg {
		q, _ := query.List()
		return QueriesLoadedMsg{Queries: q}
	}
}

func (a *App) loadTemplatesCmd() tea.Cmd {
	return func() tea.Msg {
		t, _ := query.ListTemplates()
		return TemplatesLoadedMsg{Templates: t}
	}
}

func (a *App) loadTablesCmd() tea.Cmd {
	d := a.db
	if d == nil {
		return nil
	}
	return func() tea.Msg {
		tables, err := d.ListTables(context.Background())
		if err != nil {
			return nil
		}
		entries := make([]sidebar.TableEntry, len(tables))
		for i, t := range tables {
			entries[i] = sidebar.TableEntry{Schema: t.Schema, Name: t.Name}
		}
		return TablesLoadedMsg{Tables: entries}
	}
}

func (a *App) loadColumnsCmd(schema, tableName string) tea.Cmd {
	d := a.db
	if d == nil {
		return nil
	}
	return func() tea.Msg {
		cols, err := d.ListColumns(context.Background(), schema, tableName)
		if err != nil {
			return nil
		}
		defs := make([]sidebar.ColumnDef, len(cols))
		for i, c := range cols {
			defs[i] = sidebar.ColumnDef{
				Name:     c.Name,
				DataType: c.DataType,
				Nullable: c.Nullable,
			}
		}
		return ColumnsLoadedMsg{Schema: schema, TableName: tableName, Columns: defs}
	}
}

func (a *App) executeSelectedCmd() tea.Cmd {
	sql := a.preview.SelectedStatement()
	d := a.db
	if d == nil {
		return func() tea.Msg { return QueryErrorMsg{Err: fmt.Errorf("not connected")} }
	}
	if sql == "" {
		return func() tea.Msg { return QueryErrorMsg{Err: fmt.Errorf("no statement selected")} }
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	return func() tea.Msg {
		result, err := d.Execute(ctx, sql)
		if err != nil {
			return QueryErrorMsg{Err: err}
		}
		return QueryResultMsg{Result: result}
	}
}

func (a *App) setFocus(f Focus) {
	a.sidebar.SetFocused(f == FocusSidebar)
	a.preview.SetFocused(f == FocusPreview)
	a.results.SetFocused(f == FocusResults)
	a.focus = f
}

func (a *App) layout() {
	if a.width == 0 || a.height == 0 {
		return
	}

	mainH := a.height - 1 // status bar = 1 row
	previewH := 8
	if previewH > mainH/3 {
		previewH = mainH / 3
	}
	if previewH < 3 {
		previewH = 3
	}
	resultsH := mainH - previewH

	if a.sidebarVisible {
		sidebarW := a.width / 4
		if sidebarW < 25 {
			sidebarW = 25
		}
		if sidebarW > 50 {
			sidebarW = 50
		}
		mainW := a.width - sidebarW
		a.sidebar.SetSize(sidebarW, mainH)
		a.preview.SetSize(mainW, previewH)
		a.results.SetSize(mainW, resultsH)
	} else {
		a.preview.SetSize(a.width, previewH)
		a.results.SetSize(a.width, resultsH)
	}
	a.status.SetWidth(a.width)

	a.connModal.SetSize(a.width, a.height)
	a.saveModal.SetSize(a.width, a.height)
	a.templatePicker.SetSize(a.width, a.height)
	a.templateVars.SetSize(a.width, a.height)
	a.confirmModal.SetSize(a.width, a.height)
}

func (a *App) toggleMainFocus() {
	switch a.focus {
	case FocusPreview:
		a.setFocus(FocusResults)
	default:
		a.setFocus(FocusPreview)
	}
}

func (a *App) toggleSidebarFocus() {
	if !a.sidebarVisible {
		a.sidebarVisible = true
		a.setFocus(FocusSidebar)
		a.layout()
	} else if a.focus != FocusSidebar {
		a.setFocus(FocusSidebar)
	} else {
		a.sidebarVisible = false
		a.setFocus(FocusPreview)
		a.layout()
	}
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.layout()
		a.ready = true
		return a, nil

	case tea.KeyMsg:
		if key.Matches(msg, style.Keys.Quit) {
			if a.db != nil {
				a.db.Close(context.Background())
			}
			return a, tea.Quit
		}

		if key.Matches(msg, style.Keys.Suspend) {
			return a, tea.Suspend
		}

		if key.Matches(msg, style.Keys.Escape) {
			if a.modalState != ModalNone {
				a.modalState = ModalNone
				return a, nil
			}
		}

		if a.modalState != ModalNone {
			return a.updateModal(msg)
		}

		// Global keybindings
		switch {
		case key.Matches(msg, style.Keys.Tab):
			a.toggleMainFocus()
			return a, nil
		case key.Matches(msg, style.Keys.Execute):
			stmtInfo := ""
			if a.preview.StmtCount() > 1 {
				stmtInfo = fmt.Sprintf(" (stmt %d/%d)", a.preview.StmtIndex()+1, a.preview.StmtCount())
			}
			a.status.SetMessage("Executing..." + stmtInfo)
			return a, a.executeSelectedCmd()
		case key.Matches(msg, style.Keys.Editor):
			return a, editor.Open(a.cfg.Editor, a.preview.Value())
		case key.Matches(msg, style.Keys.Save):
			a.modalState = ModalSave
			a.saveModal = modal.NewSave()
			a.saveModal.SetSize(a.width, a.height)
			return a, nil
		case key.Matches(msg, style.Keys.Template):
			templates, _ := query.ListTemplates()
			a.modalState = ModalTemplatePicker
			a.templatePicker = modal.NewTemplatePicker(templates)
			a.templatePicker.SetSize(a.width, a.height)
			return a, nil
		case key.Matches(msg, style.Keys.ConnPick):
			a.modalState = ModalConnection
			a.connModal = modal.NewConnection(a.cfg.Connections, a.connName)
			a.connModal.SetSize(a.width, a.height)
			return a, nil
		case key.Matches(msg, style.Keys.ToggleSidebar):
			a.toggleSidebarFocus()
			return a, nil
		}

	case ConnectedMsg:
		if a.db != nil {
			a.db.Close(context.Background())
		}
		a.db = msg.DB
		a.connName = msg.Name
		a.sidebar.SetConnection(msg.Name, msg.Conn.Database)
		a.status.SetMessage("Connected to " + msg.Name)
		a.status.SetConnection(msg.Name, msg.Conn.Database)
		return a, a.loadTablesCmd()

	case ConnectErrorMsg:
		a.status.SetError("Connection failed: " + msg.Err.Error())
		return a, nil

	case QueryResultMsg:
		a.cancel = nil
		a.results.SetResult(msg.Result)
		a.status.SetResult(
			msg.Result.Duration.String(),
			fmt.Sprintf("%d", len(msg.Result.Rows)),
		)
		return a, nil

	case QueryErrorMsg:
		a.cancel = nil
		a.results.SetError(msg.Err.Error())
		a.status.SetError("Query error: " + msg.Err.Error())
		return a, nil

	case QueriesLoadedMsg:
		a.sidebar.SetQueries(msg.Queries)
		return a, nil

	case TemplatesLoadedMsg:
		a.sidebar.SetTemplates(msg.Templates)
		return a, nil

	case editor.ContentMsg:
		if msg.Err != nil {
			a.status.SetError("Editor error: " + msg.Err.Error())
		} else {
			a.preview.SetValue(msg.Content)
			a.status.SetMessage("Query loaded from editor")
		}
		return a, nil

	case sidebar.SelectQueryMsg:
		a.preview.SetValue(msg.Content)
		a.setFocus(FocusPreview)
		a.status.SetMessage("Query loaded")
		return a, nil

	case sidebar.SelectTemplateMsg:
		a.modalState = ModalTemplateVars
		a.templateVars = modal.NewTemplateVars(msg.Template)
		a.templateVars.SetSize(a.width, a.height)
		return a, nil

	case sidebar.DeleteQueryMsg:
		a.deleteTarget = msg.Name
		a.modalState = ModalConfirm
		a.confirmModal = modal.NewConfirm(
			fmt.Sprintf("Delete query %q?", msg.Name),
			"delete-query",
		)
		a.confirmModal.SetSize(a.width, a.height)
		return a, nil

	case modal.SaveQueryMsg:
		a.modalState = ModalNone
		if err := query.Save(msg.Name, a.preview.Value()); err != nil {
			a.status.SetError("Save failed: " + err.Error())
		} else {
			a.status.SetMessage("Saved: " + msg.Name)
		}
		return a, a.loadQueriesCmd()

	case modal.SwitchConnectionMsg:
		a.modalState = ModalNone
		a.status.SetMessage("Connecting to " + msg.Name + "...")
		return a, a.connectCmd(msg.Name)

	case modal.OpenTemplateVarsMsg:
		a.modalState = ModalTemplateVars
		a.templateVars = modal.NewTemplateVars(msg.Template)
		a.templateVars.SetSize(a.width, a.height)
		return a, nil

	case modal.TemplateResultMsg:
		a.modalState = ModalNone
		a.preview.SetValue(msg.SQL)
		a.status.SetMessage("Template applied")
		return a, nil

	case modal.ConfirmMsg:
		a.modalState = ModalNone
		if msg.Confirmed && msg.Tag == "delete-query" {
			if err := query.Delete(a.deleteTarget); err != nil {
				a.status.SetError("Delete failed: " + err.Error())
			} else {
				a.status.SetMessage("Deleted: " + a.deleteTarget)
			}
			return a, a.loadQueriesCmd()
		}
		return a, nil

	case TablesLoadedMsg:
		a.sidebar.SetTables(msg.Tables)
		return a, nil

	case ColumnsLoadedMsg:
		a.sidebar.SetTableColumns(msg.Schema, msg.TableName, msg.Columns)
		return a, nil

	case sidebar.RequestColumnsMsg:
		return a, a.loadColumnsCmd(msg.Schema, msg.TableName)
	}

	// Delegate to focused panel
	switch a.focus {
	case FocusSidebar:
		var cmd tea.Cmd
		a.sidebar, cmd = a.sidebar.Update(msg)
		cmds = append(cmds, cmd)
	case FocusPreview:
		var cmd tea.Cmd
		a.preview, cmd = a.preview.Update(msg)
		cmds = append(cmds, cmd)
	case FocusResults:
		var cmd tea.Cmd
		a.results, cmd = a.results.Update(msg)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

func (a App) updateModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch a.modalState {
	case ModalConnection:
		a.connModal, cmd = a.connModal.Update(msg)
	case ModalSave:
		a.saveModal, cmd = a.saveModal.Update(msg)
	case ModalTemplatePicker:
		var pickerCmd tea.Cmd
		a.templatePicker, pickerCmd = a.templatePicker.Update(msg)
		return a, pickerCmd
	case ModalTemplateVars:
		a.templateVars, cmd = a.templateVars.Update(msg)
	case ModalConfirm:
		a.confirmModal, cmd = a.confirmModal.Update(msg)
	}
	return a, cmd
}

func (a App) View() tea.View {
	view := func(s string) tea.View {
		v := tea.NewView(s)
		v.AltScreen = true
		return v
	}
	if !a.ready {
		return view("Loading...")
	}

	// Update status bar dynamic info
	switch a.focus {
	case FocusSidebar:
		a.status.SetFocus("sidebar")
	case FocusPreview:
		a.status.SetFocus("query")
	case FocusResults:
		a.status.SetFocus("results")
	}
	if a.preview.StmtCount() > 1 {
		a.status.SetStmtInfo(fmt.Sprintf("%d/%d", a.preview.StmtIndex()+1, a.preview.StmtCount()))
	} else {
		a.status.SetStmtInfo("")
	}

	previewView := a.preview.View()
	resultsView := a.results.View()
	statusView := a.status.View()

	rightPane := lipgloss.JoinVertical(lipgloss.Left, previewView, resultsView)
	var mainArea string
	if a.sidebarVisible {
		sidebarView := a.sidebar.View()
		mainArea = lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, rightPane)
	} else {
		mainArea = rightPane
	}
	full := lipgloss.JoinVertical(lipgloss.Left, mainArea, statusView)

	switch a.modalState {
	case ModalConnection:
		return view(a.connModal.View())
	case ModalSave:
		return view(a.saveModal.View())
	case ModalTemplatePicker:
		return view(a.templatePicker.View())
	case ModalTemplateVars:
		return view(a.templateVars.View())
	case ModalConfirm:
		return view(a.confirmModal.View())
	}

	return view(full)
}
