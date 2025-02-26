package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
)

type loginRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

type errorResponse struct {
	Message string   `json:"message"`
	Errors  []string `json:"errors"`
}

type UserCredentials struct {
	CPF      string `json:"cpf"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

type EncryptedData struct {
	Data string `json:"data"`
	IV   string `json:"iv"`
}

type doneMsg struct{}
type tickMsg struct{}

type loginFailMsg struct {
	error string
}

type loginSuccessMsg struct {
	token string
}

type Model struct {
	step            int
	cpfForm         *huh.Form
	passwordForm    *huh.Form
	keepForm        *huh.Form
	punchForm       *huh.Form
	forgetForm      *huh.Form
	cpf             string
	password        string
	token           string
	keepLogged      bool
	loginFailMsg    loginFailMsg
	loginSuccessMsg loginSuccessMsg
	spinner         spinner.Model
	timerRunning    bool
	elapsed         time.Duration
	punchCount      int
	tickScheduled   bool
}

var theme *huh.Theme = huh.ThemeBase()

var defaultConfirm = true
var clockWerkColor = "#E28413"

func gatewayLogin(user, password string) (string, error) {
	requestBody := loginRequest{
		User:     user,
		Password: password,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("erro ao serializar dados: %w", err)
	}

	req, err := http.NewRequest(
		"POST",
		"https://snr-getaway.fly.dev/senior/login",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return "", fmt.Errorf("erro ao criar requisição: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro ao executar requisição: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var successResponse loginResponse
		if err := json.NewDecoder(resp.Body).Decode(&successResponse); err != nil {
			return "", fmt.Errorf("erro ao decodificar resposta: %w", err)
		}
		return successResponse.Token, nil

	case http.StatusUnauthorized:
		fallthrough
	case http.StatusUnprocessableEntity:
		var errorResponse errorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return "", fmt.Errorf("erro ao decodificar resposta de erro: %w", err)
		}
		return "", fmt.Errorf("%s", errorResponse.Message)

	default:
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("resposta inesperada (status %d): %s", resp.StatusCode, string(body))
	}
}

func deriveEncryptionKey() []byte {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "unknown"
	}

	seedData := hostname + homeDir + "clockwerk-salt-19538276"

	hash := sha256.Sum256([]byte(seedData))
	return hash[:]
}

func getCredentialsFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Erro ao obter diretório do usuário: %v", err)
		return "clockwerk_credentials.enc"
	}
	return filepath.Join(homeDir, ".clockwerk_credentials.enc")
}

func encrypt(data []byte, key []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)

	ciphertext := make([]byte, len(data))
	stream.XORKeyStream(ciphertext, data)

	return ciphertext, iv, nil
}

func decrypt(ciphertext []byte, key []byte, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCFBDecrypter(block, iv)

	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}

func saveCredentials(creds UserCredentials) error {
	jsonData, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("erro ao serializar credenciais: %v", err)
	}

	key := deriveEncryptionKey()

	ciphertext, iv, err := encrypt(jsonData, key)
	if err != nil {
		return fmt.Errorf("erro ao criptografar: %v", err)
	}

	encData := EncryptedData{
		Data: base64.StdEncoding.EncodeToString(ciphertext),
		IV:   base64.StdEncoding.EncodeToString(iv),
	}

	encJson, err := json.Marshal(encData)
	if err != nil {
		return fmt.Errorf("erro ao serializar dados criptografados: %v", err)
	}

	err = os.WriteFile(getCredentialsFilePath(), encJson, 0600) // Permissão apenas para o usuário
	if err != nil {
		return fmt.Errorf("erro ao salvar arquivo de credenciais: %v", err)
	}

	return nil
}

func loadCredentials() (UserCredentials, error) {
	var creds UserCredentials

	filePath := getCredentialsFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return creds, nil
		}
		return creds, fmt.Errorf("erro ao ler arquivo de credenciais: %v", err)
	}

	var encData EncryptedData
	if err := json.Unmarshal(data, &encData); err != nil {
		return creds, fmt.Errorf("erro ao desserializar dados criptografados: %v", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encData.Data)
	if err != nil {
		return creds, fmt.Errorf("erro ao decodificar ciphertext: %v", err)
	}

	iv, err := base64.StdEncoding.DecodeString(encData.IV)
	if err != nil {
		return creds, fmt.Errorf("erro ao decodificar IV: %v", err)
	}

	key := deriveEncryptionKey()

	plaintext, err := decrypt(ciphertext, key, iv)
	if err != nil {
		return creds, fmt.Errorf("erro ao descriptografar: %v", err)
	}

	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return creds, fmt.Errorf("erro ao desserializar credenciais: %v", err)
	}

	return creds, nil
}

func deleteCredentials() error {
	filePath := getCredentialsFilePath()
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("erro ao remover arquivo de credenciais: %v", err)
	}
	return nil
}

func waitCmd(seconds int) tea.Cmd {
	return tea.Tick(time.Duration(seconds)*time.Second, func(t time.Time) tea.Msg {
		return doneMsg{}
	})
}

func handleAuthentication(user string, password string) tea.Cmd {
	return func() tea.Msg {
		token, err := gatewayLogin(user, password)
		if err != nil {
			return loginFailMsg{error: err.Error()}
		}

		return loginSuccessMsg{token: token}
	}
}

func newCPFForm(initialValue string) *huh.Form {
	validateCPF := func(s string) error {
		if len(s) != 11 {
			return fmt.Errorf("CPF deve conter 11 dígitos")
		}
		for _, c := range s {
			if c < '0' || c > '9' {
				return fmt.Errorf("Apenas números permitidos")
			}
		}
		return nil
	}
	cpfInput := huh.NewInput().
		Key("cpf").
		Title("CPF").
		Placeholder("Digite").
		Value(&initialValue).
		CharLimit(11).
		Validate(validateCPF)
	nextConfirm0 := huh.NewConfirm().
		Value(&defaultConfirm).
		Key("next").
		Affirmative("Prosseguir").
		Negative("")
	return huh.NewForm(
		huh.NewGroup(cpfInput, nextConfirm0),
	).WithWidth(45).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)
}

func newPasswordForm(initialValue string) *huh.Form {
	passwordInput := huh.NewInput().
		Key("password").
		Title("Senha").
		Placeholder("Digite sua senha").
		Value(&initialValue).
		Validate(func(s string) error {
			if len(s) < 3 {
				return fmt.Errorf("Password deve conter 3 ou mais caracteres")
			}
			return nil
		}).
		EchoMode(huh.EchoModePassword)
	nextConfirm1 := huh.NewConfirm().
		Key("next").
		Value(&defaultConfirm).
		Negative("Voltar").
		Affirmative("Prosseguir")
	return huh.NewForm(
		huh.NewGroup(passwordInput, nextConfirm1),
	).WithWidth(45).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)
}

func newKeepForm(initialValue bool) *huh.Form {
	keepConfirm := huh.NewConfirm().
		Key("keep").
		Title("Deseja se manter logado?").
		Value(&initialValue).
		Affirmative("Sim").
		Negative("Não")
	proceedConfirm := huh.NewConfirm().
		Key("next").
		Value(&defaultConfirm).
		Affirmative("Prosseguir").
		Negative("Voltar")
	return huh.NewForm(
		huh.NewGroup(keepConfirm, proceedConfirm),
	).WithWidth(45).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)
}

func newPunchConfirmForm() *huh.Form {
	defaultValue := true
	confirm := huh.NewConfirm().
		Key("confirm").
		Title("Bater o ponto?").
		Value(&defaultValue).
		Affirmative("Sim").
		Negative("Cancelar")
	return huh.NewForm(
		huh.NewGroup(confirm),
	).WithWidth(45).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)
}

func newForgetForm() *huh.Form {
	defaultValue := false
	confirm := huh.NewConfirm().
		Key("confirm").
		Title("Deseja esquecer suas credenciais?").
		Value(&defaultValue).
		Affirmative("Sim").
		Negative("Não")
	return huh.NewForm(
		huh.NewGroup(confirm),
	).WithWidth(45).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)
}

func NewModel() Model {
	sp := spinner.New()
	sp.Spinner = spinner.Points
	sp.Style = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color(clockWerkColor))

	creds, err := loadCredentials()
	initialStep := 0
	initialCPF := ""
	initialPassword := ""
	initialToken := ""

	if err == nil && creds.CPF != "" && creds.Password != "" && creds.Token != "" {
		initialStep = 4
		initialCPF = creds.CPF
		initialPassword = creds.Password
		initialToken = creds.Token
	}

	return Model{
		step:         initialStep,
		cpf:          initialCPF,
		password:     initialPassword,
		token:        initialToken,
		cpfForm:      newCPFForm(initialCPF),
		passwordForm: newPasswordForm(initialPassword),
		keepForm:     newKeepForm(true),
		punchForm:    nil,
		forgetForm:   nil,
		spinner:      sp,
		timerRunning: false,
		elapsed:      0,
		punchCount:   0,
		keepLogged:   true,
	}
}

func (m Model) Init() tea.Cmd {
	tea.SetWindowTitle("Clockwerk")

	theme.Focused.Base = lipgloss.NewStyle().
		PaddingLeft(1).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color(clockWerkColor)).
		BorderLeft(true)

	if m.step == 0 {
		return m.cpfForm.Init()
	} else if m.step == 4 {
		return tea.Batch(waitCmd(1), m.spinner.Tick)
	}

	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Interrupt
		}
	}
	var cmd tea.Cmd

	switch m.step {
	case 0: // Etapa 1: CPF
		newForm, c := m.cpfForm.Update(msg)
		if f, ok := newForm.(*huh.Form); ok {
			m.cpfForm = f
		}
		cmd = c
		if m.cpfForm.State == huh.StateCompleted {
			m.cpf = m.cpfForm.GetString("cpf")
			m.step = 1
			m.passwordForm = newPasswordForm(m.passwordForm.GetString("password"))
			return m, m.passwordForm.Init()
		}
		return m, cmd

	case 1: // Etapa 2: Senha
		newForm, c := m.passwordForm.Update(msg)
		if f, ok := newForm.(*huh.Form); ok {
			m.passwordForm = f
		}
		cmd = c
		if m.passwordForm.State == huh.StateCompleted {
			if !m.passwordForm.GetBool("next") {
				m.step = 0
				m.cpfForm = newCPFForm(m.cpfForm.GetString("cpf"))
				return m, m.cpfForm.Init()
			}
			m.password = m.passwordForm.GetString("password")
			m.step = 2
			m.keepForm = newKeepForm(m.keepLogged)
			return m, m.keepForm.Init()
		}
		return m, cmd

	case 2: // Etapa 3: Manter Logado
		newForm, c := m.keepForm.Update(msg)
		if f, ok := newForm.(*huh.Form); ok {
			m.keepForm = f
		}
		cmd = c
		if m.keepForm.State == huh.StateCompleted {
			if !m.keepForm.GetBool("next") {
				m.step = 1
				m.passwordForm = newPasswordForm(m.passwordForm.GetString("password"))
				m.keepLogged = m.keepForm.GetBool("keep")
				return m, m.passwordForm.Init()
			}
			m.keepLogged = m.keepForm.GetBool("keep")
			m.step = 3
			return m, tea.Batch(
				handleAuthentication(fmt.Sprintf("%s@gazin.com.br", m.cpf), m.password),
				m.spinner.Tick,
			)
		}
		return m, cmd

	case 3: // Etapa 4: Spinner (simulação de autenticação)
		m.spinner, cmd = m.spinner.Update(msg)
		switch msg := msg.(type) {
		case loginFailMsg:
			m.loginFailMsg = msg
			return m, nil
		case loginSuccessMsg:
			m.loginSuccessMsg = msg
			m.step = 4

			if m.keepLogged {
				creds := UserCredentials{
					CPF:      m.cpf,
					Password: m.password,
					Token:    msg.token,
				}
				if err := saveCredentials(creds); err != nil {
					log.Printf("Erro ao salvar credenciais: %v", err)
				}
			}

			return m, tea.Batch(waitCmd(1), m.spinner.Tick)
		case tea.KeyMsg:
			switch msg.String() {
			case "r":
				fallthrough
			case "R":
				if m.loginFailMsg.error != "" {
					m.step = 0
					m.cpfForm = newCPFForm(m.cpfForm.GetString("cpf"))
					return m, m.cpfForm.Init()
				}
			}
		}
		return m, cmd

	case 4: // Etapa 5: Spinner (simulação de busca de eventos)
		m.spinner, cmd = m.spinner.Update(msg)
		switch msg.(type) {
		case doneMsg:
			m.step = 5
			return m, nil
		}
		return m, cmd

	case 5:
		// Tratamento para o formulário de esquecer credenciais
		if m.forgetForm != nil {
			updatedForm, c := m.forgetForm.Update(msg)
			if f, ok := updatedForm.(*huh.Form); ok {
				m.forgetForm = f
			}
			cmd = c
			if m.forgetForm.State == huh.StateCompleted {
				if m.forgetForm.GetBool("confirm") {
					if err := deleteCredentials(); err != nil {
						log.Printf("Erro ao deletar credenciais: %v", err)
					}
				}
				m.forgetForm = nil
				return m, nil
			}
			return m, cmd
		}

		// Tratamento para o formulário de confirmação de ponto
		if m.punchForm != nil {
			updatedForm, c := m.punchForm.Update(msg)
			if f, ok := updatedForm.(*huh.Form); ok {
				m.punchForm = f
			}
			cmd = c
			if m.punchForm.State == huh.StateCompleted {
				if m.punchForm.GetBool("confirm") {
					// Se confirmado, Inicie aqui um spinner para simular um HTTP POST.
					// Se tudo ocorrer bem, alterna o estado do timer e registra a batida.
					m.step = 6
					m.punchForm = nil
					return m, tea.Batch(waitCmd(1), m.spinner.Tick)
				} else {
					m.punchForm = nil
					if m.timerRunning {
						return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
							return tickMsg{}
						})
					}
				}
			}
			return m, cmd
		}

		// Se o formulário de confirmação não está ativo, então:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case " ":
				// Ao pressionar SPACE e sem formulário ativo, dispara o formulário de confirmação.
				m.punchForm = newPunchConfirmForm()
				return m, m.punchForm.Init()
			case "e", "E":
				// Ao pressionar E, dispara o formulário para esquecer credenciais
				m.forgetForm = newForgetForm()
				return m, m.forgetForm.Init()
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		case tickMsg:
			if m.timerRunning {
				m.elapsed += time.Second
				return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg{} })
			}
			return m, nil
		}
		return m, nil

	case 6:
		m.spinner, cmd = m.spinner.Update(msg)
		switch msg.(type) {
		case doneMsg:
			m.step = 5
			m.timerRunning = !m.timerRunning
			m.punchCount++
			if m.timerRunning {
				return m, tea.Batch(cmd, tea.Tick(time.Second, func(t time.Time) tea.Msg {
					m.tickScheduled = false
					return tickMsg{}
				}))
			}
			return m, nil
		}
		return m, cmd

	default:
		return m, nil
	}
}

func (m Model) View() string {
	var b strings.Builder

	switch m.step {
	case 0:
		b.WriteString(lipgloss.NewStyle().
			Bold(true).
			Render("Autenticação - Etapa 1/3: Identificação") + "\n\n")
		b.WriteString(m.cpfForm.View())

	case 1:
		loginEmail := fmt.Sprintf("%s@gazin.com.br", m.cpf)
		b.WriteString(
			lipgloss.NewStyle().Bold(true).Render("Autenticação - Etapa 2/3: Senha") + "\n\n",
		)
		b.WriteString(lipgloss.NewStyle().Italic(true).Render("Login: "+loginEmail) + "\n\n")
		b.WriteString(m.passwordForm.View())

	case 2:
		b.WriteString(lipgloss.NewStyle().
			Bold(true).
			Render("Autenticação - Etapa 3/3: Manter Logado") + "\n\n")
		b.WriteString(m.keepForm.View())

	case 3:
		if m.loginFailMsg.error != "" {
			b.WriteString(
				lipgloss.NewStyle().
					Bold(true).
					Render("Ops.. Falha no processo de autenticação") +
					"\n\n",
			)
			b.WriteString(
				lipgloss.NewStyle().
					Italic(true).
					Render(fmt.Sprintf("Login: %s@gazin.com.br", m.cpf)) + "\n",
			)
			b.WriteString(
				lipgloss.NewStyle().
					Italic(true).
					Render(fmt.Sprintf("Mensagem: %s", m.loginFailMsg.error)) +
					"\n\n",
			)
			b.WriteString(
				lipgloss.NewStyle().
					PaddingLeft(2).
					Blink(true).
					Foreground(lipgloss.Color(clockWerkColor)).
					Render(
						"¯\\_(ツ)_/¯",
					) + "\n\n",
			)
			b.WriteString("Pressione R para tentar novamente.")
		} else {
			b.WriteString(lipgloss.NewStyle().Bold(true).Render("Autenticando...") + "\n\n")
			b.WriteString(m.spinner.View())
		}

	case 4:
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Buscando últimos eventos...") + "\n\n")
		b.WriteString(m.spinner.View())

	case 5:
		now := time.Now()
		h := int(m.elapsed.Hours())
		mm := int(m.elapsed.Minutes()) % 60
		ss := int(m.elapsed.Seconds()) % 60
		timeStr := fmt.Sprintf("%02d:%02d:%02d", h, mm, ss)
		b.WriteString(
			lipgloss.NewStyle().Bold(true).Width(30).Render("Registro de Ponto - Timer") + "\n\n",
		)
		b.WriteString(
			lipgloss.NewStyle().Render("Data atual: " + now.Local().Format("02/01/2006") + "\n"),
		)
		b.WriteString(fmt.Sprintf("Registros de ponto: %d\n", m.punchCount))
		b.WriteString(lipgloss.NewStyle().Render("Tempo de trabalho:\n\n"))
		b.WriteString(
			lipgloss.NewStyle().
				Width(200).
				Height(10).
				Bold(true).
				Foreground(lipgloss.Color(clockWerkColor)).
				PaddingLeft(2).
				Render(figure.NewFigure(timeStr, "starwars", true).String() + "\n"),
		)

		if m.forgetForm != nil {
			b.WriteString("\n")
			b.WriteString(m.forgetForm.View())
		} else if m.punchForm != nil {
			b.WriteString("\nConfirma o registro de ponto?\n")
			b.WriteString(m.punchForm.View())
		} else {
			b.WriteString("\nPressione SPACE para iniciar/parar o timer.")
			b.WriteString("\nPressione E para esquecer credenciais salvas.")
			b.WriteString("\nPressione Q para sair.")
		}

	case 6:
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Registrando evento ponto...") + "\n\n")
		b.WriteString(m.spinner.View())
	}

	return b.String()
}

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()
	if _, err := tea.NewProgram(NewModel(), tea.WithAltScreen()).Run(); err != nil {
		log.Printf("Erro ao executar o programa: %s\n", err)
		os.Exit(1)
	}
}
