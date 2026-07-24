package modal

import (
	"sort"
	"strings"

	"github.com/tardanoir/seshat/internal/config"
	"github.com/tardanoir/seshat/internal/ui/style"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ── Messages ────────────────────────────────────────────────────────────────

// AIProviderSetDefaultMsg sets the active AI provider.
type AIProviderSetDefaultMsg struct{ Name string }

// OpenAIProviderFormMsg opens the add/edit form. Edit is "" to add a new one.
type OpenAIProviderFormMsg struct{ Edit string }

// DeleteAIProviderMsg removes a provider.
type DeleteAIProviderMsg struct{ Name string }

// SaveAIProviderMsg persists a provider. RawKey is the plaintext API key the
// App writes to the OS keyring; empty means "keep the existing key" (edit).
type SaveAIProviderMsg struct {
	Name   string
	Conf   config.AIProviderConf
	RawKey string
}

// BackToAIProvidersMsg returns from the form to the provider list.
type BackToAIProvidersMsg struct{}

// ── Provider list ───────────────────────────────────────────────────────────

const newProviderEntry = "+ Add provider"

type AIProvidersModel struct {
	names  []string
	provs  map[string]config.AIProviderConf
	def    string
	cursor int
	width  int
	height int
}

func NewAIProviders(c config.AIConfig) AIProvidersModel {
	names := make([]string, 0, len(c.Providers))
	for k := range c.Providers {
		names = append(names, k)
	}
	sort.Strings(names)
	names = append(names, newProviderEntry)
	return AIProvidersModel{names: names, provs: c.Providers, def: c.DefaultProvider}
}

func (m *AIProvidersModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m AIProvidersModel) Update(msg tea.Msg) (AIProvidersModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, style.Keys.Up), key.Matches(msg, key.NewBinding(key.WithKeys("k"))):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, style.Keys.Down), key.Matches(msg, key.NewBinding(key.WithKeys("j"))):
			if m.cursor < len(m.names)-1 {
				m.cursor++
			}
		case key.Matches(msg, style.Keys.Enter):
			name := m.names[m.cursor]
			if name == newProviderEntry {
				return m, func() tea.Msg { return OpenAIProviderFormMsg{Edit: ""} }
			}
			return m, func() tea.Msg { return AIProviderSetDefaultMsg{Name: name} }
		case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
			return m, func() tea.Msg { return OpenAIProviderFormMsg{Edit: ""} }
		case key.Matches(msg, key.NewBinding(key.WithKeys("e"))):
			if name := m.names[m.cursor]; name != newProviderEntry {
				return m, func() tea.Msg { return OpenAIProviderFormMsg{Edit: name} }
			}
		case key.Matches(msg, style.Keys.Delete):
			if name := m.names[m.cursor]; name != newProviderEntry {
				return m, func() tea.Msg { return DeleteAIProviderMsg{Name: name} }
			}
		}
	}
	return m, nil
}

func (m AIProvidersModel) View() string {
	var b strings.Builder
	b.WriteString(style.Title.Render("AI Providers"))
	b.WriteString("\n\n")

	for i, name := range m.names {
		selected := i == m.cursor
		prefix := "  "
		if selected {
			prefix = style.ListSelected.Render("▸ ")
		}
		if name == newProviderEntry {
			b.WriteString(style.StatusMsg.Render("────────────────────────"))
			b.WriteString("\n")
			if selected {
				b.WriteString(style.ListSelected.Render("▸ " + newProviderEntry))
			} else {
				b.WriteString(style.ListItem.Render("  " + newProviderEntry))
			}
			b.WriteString("\n")
			continue
		}

		p := m.provs[name]
		label := name
		if name == m.def {
			label += style.Success.Render(" *")
		}
		detail := p.Kind
		if p.Model != "" {
			detail += " · " + p.Model
		}
		if len(p.Argv) > 0 {
			detail += " · " + strings.Join(p.Argv, " ")
		}

		if selected {
			b.WriteString(prefix + style.ListSelected.Render(label))
		} else {
			b.WriteString(prefix + style.ListItem.Render(label))
		}
		b.WriteString("\n    " + style.StatusMsg.Render(detail) + "\n")
	}

	b.WriteString("\n")
	hints := []string{
		style.StatusKey.Render("↑↓") + " " + style.StatusKeyLabel.Render("navigate"),
		style.StatusKey.Render("↵") + " " + style.StatusKeyLabel.Render("set default"),
		style.StatusKey.Render("a") + " " + style.StatusKeyLabel.Render("add"),
		style.StatusKey.Render("e") + " " + style.StatusKeyLabel.Render("edit"),
		style.StatusKey.Render("C-d") + " " + style.StatusKeyLabel.Render("delete"),
		style.StatusKey.Render("Esc") + " " + style.StatusKeyLabel.Render("close"),
	}
	b.WriteString("  " + strings.Join(hints, "  "))

	content := style.ModalOverlay.Width(62).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// ── Add / edit form ─────────────────────────────────────────────────────────

var aiKinds = []string{"anthropic", "openai", "gemini", "cli"}

var aiKindDesc = map[string]string{
	"anthropic": "Anthropic Claude (HTTP API)",
	"openai":    "OpenAI / compatible (HTTP API)",
	"gemini":    "Google Gemini (HTTP API)",
	"cli":       "Local CLI subprocess (claude, opencode, …)",
}

type aiFormStep int

const (
	aiStepPickKind aiFormStep = iota
	aiStepForm
)

type aiFormField int

const (
	afName aiFormField = iota
	afModel
	afAPIKey
	afBaseURL
	afArgv
)

type AIProviderFormModel struct {
	editing string // original name when editing; "" when adding
	step    aiFormStep
	kindSel int
	inputs  map[aiFormField]textinput.Model
	focus   aiFormField
	err     string
	width   int
	height  int
}

func NewAIProviderForm(edit string, conf config.AIProviderConf) AIProviderFormModel {
	mk := func(placeholder string, width int) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = 256
		ti.SetWidth(width)
		return ti
	}
	inputs := map[aiFormField]textinput.Model{
		afName:    mk("my-provider", 36),
		afModel:   mk("(provider default)", 36),
		afAPIKey:  mk("sk-…", 36),
		afBaseURL: mk("(optional override)", 40),
		afArgv:    mk("claude -p {prompt}", 40),
	}
	pw := inputs[afAPIKey]
	pw.EchoMode = textinput.EchoPassword
	inputs[afAPIKey] = pw

	m := AIProviderFormModel{editing: edit, inputs: inputs}
	if edit != "" {
		m.step = aiStepForm
		for i, k := range aiKinds {
			if k == conf.Kind {
				m.kindSel = i
			}
		}
		setVal(inputs, afName, edit)
		setVal(inputs, afModel, conf.Model)
		setVal(inputs, afBaseURL, conf.BaseURL)
		setVal(inputs, afArgv, strings.Join(conf.Argv, " "))
		ph := inputs[afAPIKey]
		ph.Placeholder = "leave blank to keep current"
		inputs[afAPIKey] = ph
		m.focusField(afName)
	}
	return m
}

func setVal(inputs map[aiFormField]textinput.Model, f aiFormField, v string) {
	ti := inputs[f]
	ti.SetValue(v)
	inputs[f] = ti
}

func (m *AIProviderFormModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *AIProviderFormModel) currentKind() string { return aiKinds[m.kindSel] }
func (m *AIProviderFormModel) isCLI() bool          { return m.currentKind() == "cli" }

func (m *AIProviderFormModel) formFields() []aiFormField {
	if m.isCLI() {
		return []aiFormField{afName, afArgv}
	}
	return []aiFormField{afName, afModel, afAPIKey, afBaseURL}
}

func (m *AIProviderFormModel) focusField(f aiFormField) {
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

func (m *AIProviderFormModel) step2(delta int) {
	order := m.formFields()
	for i, f := range order {
		if f == m.focus {
			j := i + delta
			if j >= 0 && j < len(order) {
				m.focusField(order[j])
			}
			return
		}
	}
}

func (m AIProviderFormModel) Update(msg tea.Msg) (AIProviderFormModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if m.step == aiStepPickKind {
		switch {
		case key.Matches(keyMsg, style.Keys.Escape):
			return m, func() tea.Msg { return BackToAIProvidersMsg{} }
		case key.Matches(keyMsg, style.Keys.Up), key.Matches(keyMsg, key.NewBinding(key.WithKeys("k"))):
			if m.kindSel > 0 {
				m.kindSel--
			}
		case key.Matches(keyMsg, style.Keys.Down), key.Matches(keyMsg, key.NewBinding(key.WithKeys("j"))):
			if m.kindSel < len(aiKinds)-1 {
				m.kindSel++
			}
		case key.Matches(keyMsg, style.Keys.Enter):
			m.step = aiStepForm
			m.err = ""
			m.focusField(afName)
		}
		return m, nil
	}

	switch {
	case key.Matches(keyMsg, style.Keys.Escape):
		if m.editing == "" {
			m.step = aiStepPickKind
			m.err = ""
			return m, nil
		}
		return m, func() tea.Msg { return BackToAIProvidersMsg{} }
	case key.Matches(keyMsg, key.NewBinding(key.WithKeys("tab"))):
		m.step2(1)
		return m, nil
	case key.Matches(keyMsg, key.NewBinding(key.WithKeys("shift+tab"))):
		m.step2(-1)
		return m, nil
	case key.Matches(keyMsg, style.Keys.Enter):
		if e := m.validate(); e != "" {
			m.err = e
			return m, nil
		}
		name, conf, rawKey := m.build()
		return m, func() tea.Msg { return SaveAIProviderMsg{Name: name, Conf: conf, RawKey: rawKey} }
	}

	if inp, ok := m.inputs[m.focus]; ok {
		var cmd tea.Cmd
		inp, cmd = inp.Update(msg)
		m.inputs[m.focus] = inp
		return m, cmd
	}
	return m, nil
}

func (m AIProviderFormModel) validate() string {
	if strings.TrimSpace(m.inputs[afName].Value()) == "" {
		return "name is required"
	}
	if m.isCLI() {
		if strings.TrimSpace(m.inputs[afArgv].Value()) == "" {
			return "argv is required (e.g. claude -p {prompt})"
		}
		return ""
	}
	if m.editing == "" && strings.TrimSpace(m.inputs[afAPIKey].Value()) == "" {
		return "api key is required"
	}
	return ""
}

func (m AIProviderFormModel) build() (string, config.AIProviderConf, string) {
	name := strings.TrimSpace(m.inputs[afName].Value())
	conf := config.AIProviderConf{Kind: m.currentKind()}
	rawKey := ""
	if m.isCLI() {
		conf.Argv = strings.Fields(m.inputs[afArgv].Value())
	} else {
		conf.Model = strings.TrimSpace(m.inputs[afModel].Value())
		conf.BaseURL = strings.TrimSpace(m.inputs[afBaseURL].Value())
		rawKey = strings.TrimSpace(m.inputs[afAPIKey].Value())
	}
	return name, conf, rawKey
}

func (m AIProviderFormModel) View() string {
	if m.step == aiStepPickKind {
		return m.viewKindPicker()
	}
	return m.viewForm()
}

func (m AIProviderFormModel) viewKindPicker() string {
	var b strings.Builder
	b.WriteString(style.Title.Render("New AI Provider"))
	b.WriteString("\n\n")
	b.WriteString(style.Label.Render("Select kind:"))
	b.WriteString("\n\n")
	for i, k := range aiKinds {
		line := k + "  " + style.StatusMsg.Render(aiKindDesc[k])
		if i == m.kindSel {
			b.WriteString(style.ListSelected.Render("▸ " + line))
		} else {
			b.WriteString(style.ListItem.Render("  " + line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(style.ListItem.Render("↑↓ navigate  ↵ select  Esc back"))
	content := style.ModalOverlay.Width(54).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m AIProviderFormModel) viewForm() string {
	var b strings.Builder
	title := "New AI Provider"
	if m.editing != "" {
		title = "Edit AI Provider"
	}
	b.WriteString(style.Title.Render(title))
	b.WriteString("  ")
	b.WriteString(style.StatusMsg.Render("(" + m.currentKind() + ")"))
	b.WriteString("\n\n")

	row := func(label string, f aiFormField) {
		prefix := "  "
		if f == m.focus {
			prefix = style.ListSelected.Render("▸ ")
		}
		b.WriteString(prefix)
		b.WriteString(style.Label.Render(padRight(label, 9)))
		b.WriteString(" ")
		b.WriteString(m.inputs[f].View())
		b.WriteString("\n")
	}

	row("Name", afName)
	if m.isCLI() {
		b.WriteString("\n")
		row("Argv", afArgv)
		b.WriteString("\n")
		b.WriteString(style.StatusMsg.Render("  {prompt} is replaced with the request; else sent on stdin."))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
		row("Model", afModel)
		row("API Key", afAPIKey)
		row("Base URL", afBaseURL)
		b.WriteString("\n")
		b.WriteString(style.StatusMsg.Render("  Key is stored in your OS keyring, not in config.toml."))
		b.WriteString("\n")
	}

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(style.Error.Render("  " + m.err))
	}

	b.WriteString("\n")
	b.WriteString(style.ListItem.Render("Tab/S-Tab navigate  ↵ save  Esc back"))

	content := style.ModalOverlay.Width(58).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
