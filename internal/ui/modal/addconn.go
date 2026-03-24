package modal

import (
	"strconv"
	"strings"

	"github.com/tardanoir/seshat/internal/config"
	"github.com/tardanoir/seshat/internal/ui/style"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type AddConnectionMsg struct {
	Name       string
	Connection config.Connection
}

type BackToConnListMsg struct{}

var connTypes = []string{"postgres", "sqlite", "csv", "json"}

var connTypeDesc = map[string]string{
	"postgres": "PostgreSQL server",
	"sqlite":   "SQLite database file",
	"csv":      "CSV file (queried via SQL)",
	"json":     "JSON file (queried via SQL)",
}

type addConnStep int

const (
	stepPickType addConnStep = iota
	stepForm
)

type addConnField int

const (
	fieldName addConnField = iota
	fieldHost
	fieldPort
	fieldDatabase
	fieldUser
	fieldPassword
	fieldSSLMode
	fieldPath
	fieldSSHHost
	fieldSSHPort
	fieldSSHUser
	fieldSSHKey
	fieldSSHPassword
)

type AddConnModel struct {
	step    addConnStep
	typeSel int // index into connTypes
	inputs  map[addConnField]textinput.Model
	focus   addConnField
	err     string
	width   int
	height  int
}

func NewAddConn() AddConnModel {
	inputs := make(map[addConnField]textinput.Model)

	mk := func(placeholder string, width int) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = 128
		ti.SetWidth(width)
		return ti
	}

	inputs[fieldName] = mk("my-connection", 36)
	inputs[fieldHost] = mk("localhost", 36)
	inputs[fieldPort] = mk("5432", 10)
	inputs[fieldDatabase] = mk("postgres", 36)
	inputs[fieldUser] = mk("postgres", 36)
	inputs[fieldPassword] = mk("", 36)
	inputs[fieldSSLMode] = mk("disable", 20)
	inputs[fieldPath] = mk("/path/to/file.csv", 40)
	inputs[fieldSSHHost] = mk("ec2-xx.compute-1.amazonaws.com", 36)
	inputs[fieldSSHPort] = mk("22", 10)
	inputs[fieldSSHUser] = mk("ubuntu", 36)
	inputs[fieldSSHKey] = mk("~/.ssh/id_rsa", 36)
	inputs[fieldSSHPassword] = mk("", 36)

	pw := inputs[fieldPassword]
	pw.EchoMode = textinput.EchoPassword
	inputs[fieldPassword] = pw

	sshPw := inputs[fieldSSHPassword]
	sshPw.EchoMode = textinput.EchoPassword
	inputs[fieldSSHPassword] = sshPw

	return AddConnModel{
		step:   stepPickType,
		inputs: inputs,
	}
}

func (m *AddConnModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *AddConnModel) currentType() string {
	return connTypes[m.typeSel]
}

func (m *AddConnModel) isFileType() bool {
	t := m.currentType()
	return t == "sqlite" || t == "csv" || t == "json"
}

func (m *AddConnModel) formFields() []addConnField {
	fields := []addConnField{fieldName}
	if m.isFileType() {
		fields = append(fields, fieldPath)
	} else {
		fields = append(fields, fieldHost, fieldPort, fieldDatabase, fieldUser, fieldPassword, fieldSSLMode,
			fieldSSHHost, fieldSSHPort, fieldSSHUser, fieldSSHKey, fieldSSHPassword)
	}
	return fields
}

func (m *AddConnModel) focusField(f addConnField) {
	for k, inp := range m.inputs {
		inp.Blur()
		m.inputs[k] = inp
	}
	m.focus = f
	if inp, ok := m.inputs[f]; ok {
		inp.Focus()
		m.inputs[f] = inp
	}
}

func (m *AddConnModel) nextField() {
	order := m.formFields()
	for i, f := range order {
		if f == m.focus && i < len(order)-1 {
			m.focusField(order[i+1])
			return
		}
	}
}

func (m *AddConnModel) prevField() {
	order := m.formFields()
	for i, f := range order {
		if f == m.focus && i > 0 {
			m.focusField(order[i-1])
			return
		}
	}
}

func (m *AddConnModel) enterForm() {
	m.step = stepForm
	m.err = ""
	m.focusField(fieldName)
}

func (m AddConnModel) buildConnection() (string, config.Connection) {
	name := strings.TrimSpace(m.inputs[fieldName].Value())
	conn := config.Connection{Type: m.currentType()}
	if m.isFileType() {
		conn.Path = strings.TrimSpace(m.inputs[fieldPath].Value())
	} else {
		conn.Host = strings.TrimSpace(m.inputs[fieldHost].Value())
		port, _ := strconv.Atoi(strings.TrimSpace(m.inputs[fieldPort].Value()))
		if port == 0 {
			port = 5432
		}
		conn.Port = port
		conn.Database = strings.TrimSpace(m.inputs[fieldDatabase].Value())
		conn.User = strings.TrimSpace(m.inputs[fieldUser].Value())
		conn.Password = strings.TrimSpace(m.inputs[fieldPassword].Value())
		conn.SSLMode = strings.TrimSpace(m.inputs[fieldSSLMode].Value())

		sshHost := strings.TrimSpace(m.inputs[fieldSSHHost].Value())
		if sshHost != "" {
			sshPort, _ := strconv.Atoi(strings.TrimSpace(m.inputs[fieldSSHPort].Value()))
			if sshPort == 0 {
				sshPort = 22
			}
			conn.SSH = &config.SSHTunnel{
				Host:     sshHost,
				Port:     sshPort,
				User:     strings.TrimSpace(m.inputs[fieldSSHUser].Value()),
				Key:      strings.TrimSpace(m.inputs[fieldSSHKey].Value()),
				Password: strings.TrimSpace(m.inputs[fieldSSHPassword].Value()),
			}
		}
	}
	return name, conn
}

func (m *AddConnModel) validate() string {
	if strings.TrimSpace(m.inputs[fieldName].Value()) == "" {
		return "name is required"
	}
	if m.isFileType() {
		if strings.TrimSpace(m.inputs[fieldPath].Value()) == "" {
			return "file path is required"
		}
	} else {
		if strings.TrimSpace(m.inputs[fieldHost].Value()) == "" {
			return "host is required"
		}
	}
	return ""
}

func (m AddConnModel) Update(msg tea.Msg) (AddConnModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.step == stepPickType {
			return m.updateTypePicker(msg)
		}
		return m.updateForm(msg)
	}
	return m, nil
}

func (m AddConnModel) updateTypePicker(msg tea.KeyMsg) (AddConnModel, tea.Cmd) {
	switch {
	case key.Matches(msg, style.Keys.Escape):
		return m, func() tea.Msg { return BackToConnListMsg{} }
	case key.Matches(msg, style.Keys.Up), key.Matches(msg, key.NewBinding(key.WithKeys("k"))):
		if m.typeSel > 0 {
			m.typeSel--
		}
	case key.Matches(msg, style.Keys.Down), key.Matches(msg, key.NewBinding(key.WithKeys("j"))):
		if m.typeSel < len(connTypes)-1 {
			m.typeSel++
		}
	case key.Matches(msg, style.Keys.Enter):
		m.enterForm()
	}
	return m, nil
}

func (m AddConnModel) updateForm(msg tea.KeyMsg) (AddConnModel, tea.Cmd) {
	switch {
	case key.Matches(msg, style.Keys.Escape):
		m.step = stepPickType
		m.err = ""
		return m, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
		m.nextField()
		return m, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
		m.prevField()
		return m, nil
	case key.Matches(msg, style.Keys.Enter):
		if errMsg := m.validate(); errMsg != "" {
			m.err = errMsg
			return m, nil
		}
		name, conn := m.buildConnection()
		return m, func() tea.Msg {
			return AddConnectionMsg{Name: name, Connection: conn}
		}
	}

	// Forward to focused text input.
	if inp, ok := m.inputs[m.focus]; ok {
		var cmd tea.Cmd
		inp, cmd = inp.Update(msg)
		m.inputs[m.focus] = inp
		return m, cmd
	}
	return m, nil
}

func (m AddConnModel) View() string {
	if m.step == stepPickType {
		return m.viewTypePicker()
	}
	return m.viewForm()
}

func (m AddConnModel) viewTypePicker() string {
	var b strings.Builder
	b.WriteString(style.Title.Render("New Connection"))
	b.WriteString("\n\n")
	b.WriteString(style.Label.Render("Select type:"))
	b.WriteString("\n\n")

	for i, t := range connTypes {
		desc := connTypeDesc[t]
		line := t + "  " + style.StatusMsg.Render(desc)
		if i == m.typeSel {
			b.WriteString(style.ListSelected.Render("▸ " + line))
		} else {
			b.WriteString(style.ListItem.Render("  " + line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(style.ListItem.Render("↑↓ navigate  ↵ select  Esc back"))

	modalW := 50
	content := style.ModalOverlay.Width(modalW).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m AddConnModel) viewForm() string {
	var b strings.Builder
	b.WriteString(style.Title.Render("New Connection"))
	b.WriteString("  ")
	b.WriteString(style.StatusMsg.Render("(" + m.currentType() + ")"))
	b.WriteString("\n\n")

	row := func(label string, f addConnField) {
		prefix := "  "
		if f == m.focus {
			prefix = style.ListSelected.Render("▸ ")
		}
		b.WriteString(prefix)
		b.WriteString(style.Label.Render(padRight(label, 10)))
		b.WriteString(" ")
		b.WriteString(m.inputs[f].View())
		b.WriteString("\n")
	}

	row("Name", fieldName)
	b.WriteString("\n")

	if m.isFileType() {
		row("Path", fieldPath)
	} else {
		row("Host", fieldHost)
		row("Port", fieldPort)
		row("Database", fieldDatabase)
		row("User", fieldUser)
		row("Password", fieldPassword)
		row("SSL Mode", fieldSSLMode)
		b.WriteString("\n")
		b.WriteString(style.StatusMsg.Render("  SSH Tunnel (optional)"))
		b.WriteString("\n")
		row("SSH Host", fieldSSHHost)
		row("SSH Port", fieldSSHPort)
		row("SSH User", fieldSSHUser)
		row("SSH Key", fieldSSHKey)
		row("SSH Pass", fieldSSHPassword)
	}

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(style.Error.Render("  " + m.err))
	}

	b.WriteString("\n\n")
	b.WriteString(style.ListItem.Render("Tab/S-Tab navigate  ↵ save  Esc back"))

	modalW := 58
	content := style.ModalOverlay.Width(modalW).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
