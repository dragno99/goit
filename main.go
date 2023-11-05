package main

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

const (
	initialInputs = 1
	maxInputs     = 6
	minInputs     = 1
	helpHeight    = 5
)

var (
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	cursorLineStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("57")).
			Foreground(lipgloss.Color("230"))

	placeholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("238"))

	endOfBufferStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("235"))

	focusedPlaceholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("99"))

	focusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238"))

	blurredBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.HiddenBorder())

	zones     []string
	store     modelTextInput
	text      string
	isPresent bool
)

func max(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func min(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

type keymap = struct {
	next, prev, add, remove, quit, save key.Binding
}

type modelTextInput struct {
	textInput textinput.Model
	err       error
	idx       int
}

func initialModel(idx int) modelTextInput {
	ti := textinput.New()
	ti.Placeholder = "Filename"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return modelTextInput{
		textInput: ti,
		err:       nil,
		idx:       idx,
	}
}

func (m modelTextInput) Init() tea.Cmd {
	return textinput.Blink
}

func (m modelTextInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			isPresent = false
			return m, nil
		case tea.KeyEnter:
			filename := m.textInput.Value()
			os.Create(filename)
			_ = os.WriteFile(filename, []byte(text), fs.ModePerm)
			isPresent = false
			return m, nil
		}
	}

	return m, nil

}

func (m modelTextInput) View() string {
	return fmt.Sprintf(
		"Enter Filename?\n\n%s\n\n%s",
		m.textInput.View(),
		"(esc to quit)",
	) + "\n" + "\n\n"
}

func newTextarea() textarea.Model {
	t := textarea.New()
	t.Prompt = ""
	t.Placeholder = "Type something"
	t.ShowLineNumbers = true
	t.Cursor.Style = cursorStyle
	t.FocusedStyle.Placeholder = focusedPlaceholderStyle
	t.BlurredStyle.Placeholder = placeholderStyle
	t.FocusedStyle.CursorLine = cursorLineStyle
	t.FocusedStyle.Base = focusedBorderStyle
	t.BlurredStyle.Base = blurredBorderStyle
	t.FocusedStyle.EndOfBuffer = endOfBufferStyle
	t.BlurredStyle.EndOfBuffer = endOfBufferStyle
	t.KeyMap.DeleteWordBackward.SetEnabled(false)
	t.KeyMap.LineNext = key.NewBinding(key.WithKeys("down"))
	t.KeyMap.LinePrevious = key.NewBinding(key.WithKeys("up"))
	t.Blur()

	return t
}

type model struct {
	width  int
	height int
	keymap keymap
	help   help.Model
	inputs []textarea.Model
	focus  int
}

func newModel() model {
	m := model{
		inputs: make([]textarea.Model, initialInputs),
		help:   help.New(),
		keymap: keymap{
			next: key.NewBinding(
				key.WithKeys("tab"),
				key.WithHelp("tab", "next"),
			),
			prev: key.NewBinding(
				key.WithKeys("shift+tab"),
				key.WithHelp("shift+tab", "prev"),
			),
			add: key.NewBinding(
				key.WithKeys("ctrl+n"),
				key.WithHelp("ctrl+n", "add an editor"),
			),
			remove: key.NewBinding(
				key.WithKeys("ctrl+w"),
				key.WithHelp("ctrl+w", "remove an editor"),
			),
			quit: key.NewBinding(
				key.WithKeys("esc", "ctrl+c"),
				key.WithHelp("esc", "quit"),
			),
			save: key.NewBinding(
				key.WithKeys("ctrl+s"),
				key.WithHelp("ctrl+s", "save file"),
			),
		},
	}
	for i := 0; i < initialInputs; i++ {
		m.inputs[i] = newTextarea()
	}
	m.inputs[m.focus].Focus()
	m.updateKeybindings()
	return m
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	if isPresent {
		store.Update(msg)
		store.textInput, _ = store.textInput.Update(msg)
		return m, nil
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.quit):
			for i := range m.inputs {
				m.inputs[i].Blur()
			}
			return m, tea.Quit
		case key.Matches(msg, m.keymap.next):
			m.inputs[m.focus].Blur()
			m.focus++
			if m.focus > len(m.inputs)-1 {
				m.focus = 0
			}
			cmd := m.inputs[m.focus].Focus()
			cmds = append(cmds, cmd)
		case key.Matches(msg, m.keymap.prev):
			m.inputs[m.focus].Blur()
			m.focus--
			if m.focus < 0 {
				m.focus = len(m.inputs) - 1
			}
			cmd := m.inputs[m.focus].Focus()
			cmds = append(cmds, cmd)
		case key.Matches(msg, m.keymap.add):
			m.inputs = append(m.inputs, newTextarea())
		case key.Matches(msg, m.keymap.remove):
			if len(m.inputs) > 1 {
				idx := m.focus
				m.inputs = append(m.inputs[:idx], m.inputs[idx+1:]...)
				zones = append(zones[:idx], zones[idx+1:]...)
				m.focus = m.focus % len(m.inputs)
				cmds = append(cmds, m.inputs[m.focus].Focus())
			}
		case key.Matches(msg, m.keymap.save):
			store = initialModel(m.focus)
			isPresent = true
			text = m.inputs[m.focus].Value()
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	case tea.MouseMsg:
		if msg.Type == tea.MouseWheelUp {
			m.inputs[m.focus].CursorUp()
			return m, nil
		} else if msg.Type == tea.MouseWheelDown {
			m.inputs[m.focus].CursorDown()
			return m, nil
		} else if msg.Type == tea.MouseLeft {
			for i, id := range zones {
				if zone.Get(id).InBounds(msg) {

					if i == m.focus { // current box
						x, y := zone.Get(id).Pos(msg)

						y--

						y = min(max(0, y), m.inputs[m.focus].LineCount()-1)

						for m.inputs[m.focus].Line() < y {
							m.inputs[m.focus].CursorDown()
						}
						for y < m.inputs[m.focus].Line() {
							m.inputs[m.focus].CursorUp()
						}

						m.inputs[m.focus].SetCursor(x - 4)

						return m, nil

					} else {

						m.inputs[m.focus].Blur()
						m.focus = i

						return m, m.inputs[m.focus].Focus()
					}

				}

			}
		}
	}

	m.updateKeybindings()
	m.sizeInputs()

	if !isPresent {
		for i := range m.inputs {
			newModel, cmd := m.inputs[i].Update(msg)
			m.inputs[i] = newModel
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *model) sizeInputs() {
	for i := range m.inputs {
		m.inputs[i].SetWidth(m.width / len(m.inputs))
		m.inputs[i].SetHeight(m.height - helpHeight)
	}
}

func (m *model) updateKeybindings() {
	m.keymap.add.SetEnabled(len(m.inputs) < maxInputs)
	m.keymap.remove.SetEnabled(len(m.inputs) > minInputs)
}

func (m model) View() string {
	help := m.help.ShortHelpView([]key.Binding{
		m.keymap.next,
		m.keymap.prev,
		m.keymap.add,
		m.keymap.remove,
		m.keymap.save,
		m.keymap.quit,
	})

	var views []string
	zones = make([]string, 0)

	if isPresent {
		return zone.Scan(store.View())
	}

	for i := range m.inputs {

		zoneId := zone.NewPrefix()
		zones = append(zones, zoneId)
		views = append(views, zone.Mark(zoneId, m.inputs[i].View()))

	}

	return zone.Scan(lipgloss.JoinHorizontal(lipgloss.Top, views...) + "\n\n" + help)
}

func main() {
	zone.NewGlobal()
	defer zone.Close()

	if _, err := tea.NewProgram(newModel(), tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Println("Error while running program:", err)
		os.Exit(1)
	}
}
