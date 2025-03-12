package internal

import "github.com/charmbracelet/bubbles/key"

type customHelp []key.Binding

type keyMap struct {
	Punch       key.Binding
	ForgetCreds key.Binding
	Quit        key.Binding
	Exit        key.Binding
	MoveBack    key.Binding
	MoveForward key.Binding
	Retry       key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.MoveBack,
		k.MoveForward,
		k.Punch,
		k.ForgetCreds,
		k.Exit,
		k.Quit,
	}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			k.MoveBack,
			k.MoveForward,
			k.Punch,
			k.ForgetCreds,
			k.Exit,
			k.Quit,
		},
	}
}

var keys = keyMap{
	MoveBack: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "Anterior"),
	),
	MoveForward: key.NewBinding(
		key.WithKeys("right", "l", "tab"),
		key.WithHelp("→/l", "Próxima"),
	),
	Punch: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("<space>", "Ponto"),
	),
	ForgetCreds: key.NewBinding(
		key.WithKeys("e", "E"),
		key.WithHelp("<e>", "Esquecer"),
	),
	Retry: key.NewBinding(
		key.WithKeys("r", "R"),
		key.WithHelp("<r>", "Tentar novamente"),
	),
	Exit: key.NewBinding(
		key.WithKeys("q", "Q"),
		key.WithHelp("<q>", "Fechar"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("<ctrl+c>", "Sair"),
	),
}

func (c customHelp) ShortHelp() []key.Binding {
	return []key.Binding(c)
}

func (c customHelp) FullHelp() [][]key.Binding {
	return [][]key.Binding{c}
}
