package senior

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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

type requestFilter struct {
	ActivePlatformUser bool     `json:"activePlatformUser"`
	PageInfo           pageInfo `json:"pageInfo"`
	NameSearch         string   `json:"nameSearch"`
	Sort               sort     `json:"sort"`
}

type pageInfo struct {
	Page     int    `json:"page"`
	PageSize string `json:"pageSize"`
}

type sort struct {
	Field interface{} `json:"field"`
	Order string      `json:"order"`
}

type clockingEventRequest struct {
	Filter requestFilter `json:"filter"`
}

type clockingEventResponse struct {
	Result []clockingEvent `json:"result"`
}

type clockingEventImported struct {
	DateEvent string `json:"dateEvent"`
	TimeEvent string `json:"timeEvent"`
}

type clockingResult struct {
	EventImported clockingEventImported `json:"clockingEventImported"`
}

type postClockingEventResponse struct {
	Result clockingResult `json:"clockingResult"`
}

type ClockingCompany struct {
	ID         string `json:"id"`
	ArpID      string `json:"arpId"`
	Identifier string `json:"identifier"`
	Caepf      string `json:"caepf"`
	CnoNumber  string `json:"cnoNumber"`
}

type ClockingEmployee struct {
	ID    string `json:"id"`
	ArpID string `json:"arpId"`
	Cpf   string `json:"cpf"`
	Pis   string `json:"pis"`
}

type ClockingSignature struct {
	SignatureVersion int    `json:"signatureVersion"`
	Signature        string `json:"signature"`
}

type ClockingInfo struct {
	Company    ClockingCompany   `json:"company"`
	Employee   ClockingEmployee  `json:"employee"`
	AppVersion string            `json:"appVersion"`
	TimeZone   string            `json:"timeZone"`
	Signature  ClockingSignature `json:"signature"`
	Use        string            `json:"use"`
}

type ClockingRequest struct {
	ClockingInfo ClockingInfo `json:"clockingInfo"`
}

type clockingEvent struct {
	ID               string   `json:"id"`
	DateEvent        string   `json:"dateEvent"`
	TimeEvent        string   `json:"timeEvent"`
	Cnpj             string   `json:"cnpj"`
	Caepf            string   `json:"caepf"`
	CnoNumber        string   `json:"cnoNumber"`
	AppVersion       string   `json:"appVersion"`
	TimeZone         string   `json:"timeZone"`
	Signature        string   `json:"signature"`
	SignatureVersion int      `json:"signatureVersion"`
	Employee         employee `json:"employee"`
	Platform         string   `json:"platform"`
	Use              int      `json:"use"`
}

type employee struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Pis       string  `json:"pis"`
	Shift     string  `json:"shift"`
	Timetable string  `json:"timeTable"`
	Company   company `json:"company"`
	ArpID     string  `json:"arpId"`
	CpfNumber string  `json:"cpfNumber"`
}

type company struct {
	Cnpj  string `json:"cnpj"`
	Name  string `json:"name"`
	ID    string `json:"id"`
	ArpID string `json:"arpId"`
}

func GatewayLogin(user, password string) (string, error) {
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

func GetClockingEvents(token string) ([]clockingEvent, error) {
	requestBody := clockingEventRequest{
		Filter: requestFilter{
			ActivePlatformUser: true,
			PageInfo: pageInfo{
				Page:     0,
				PageSize: "20",
			},
			NameSearch: "",
			Sort: sort{
				Field: nil,
				Order: "ASC",
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar dados: %w", err)
	}

	req, err := http.NewRequest(
		"POST",
		"https://platform.senior.com.br/t/senior.com.br/bridge/1.0/rest/hcm/pontomobile/queries/clockingEventByActiveUserQuery",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar requisição: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var response clockingEventResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
		}
		return response.Result, nil

	case http.StatusUnauthorized:
		var errorResponse errorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return nil, fmt.Errorf("token expirado ou inválido: %w", err)
		}
		return nil, fmt.Errorf("autorização falhou: %s", errorResponse.Message)

	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro inesperado (status %d): %s", resp.StatusCode, string(body))
	}
}

func PostClockingEvent(token string, body ClockingRequest) (postClockingEventResponse, error) {
	// requestBody := ClockingRequest{
	// 	ClockingInfo: ClockingInfo{
	// 		Company: ClockingCompany{
	// 			ID:         "dbdfc7df-f1b9-4acd-9f31-6a95e4245196",
	// 			ArpID:      "7d750c36-6195-43a1-831c-6a93d32cac07",
	// 			Identifier: "77941490000155",
	// 			Caepf:      "0",
	// 			CnoNumber:  "0",
	// 		},
	// 		Employee: ClockingEmployee{
	// 			ID:    "2c1053b0-735f-414e-997f-f2219415cc02",
	// 			ArpID: "f7a74567-0d24-434e-97b1-4faa465fe1a4",
	// 			Cpf:   "07403847911",
	// 			Pis:   "19030655812",
	// 		},
	// 		AppVersion: "3.12.3",
	// 		TimeZone:   "America/Sao_Paulo",
	// 		Signature: ClockingSignature{
	// 			SignatureVersion: 1,
	// 			Signature:        "ZDc0NDBhYzIxMzFlMmY4YTFkNGQyNjJiM2Y4YjAxZTRmNjIxZTZhY2UxNWZlNzYwMWMyYzU0NDVkMmQ2MzIxZg==",
	// 		},
	// 		Use: "02",
	// 	},
	// }

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return postClockingEventResponse{}, fmt.Errorf("erro ao serializar dados: %w", err)
	}

	req, err := http.NewRequest(
		"POST",
		"https://platform.senior.com.br/t/senior.com.br/bridge/1.0/rest/hcm/pontomobile_clocking_event/actions/clockingEventImportByBrowser",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return postClockingEventResponse{}, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return postClockingEventResponse{}, fmt.Errorf("erro ao executar requisição: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var result postClockingEventResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return postClockingEventResponse{}, fmt.Errorf("erro ao decodificar resposta: %w", err)
		}
		return result, nil

	case http.StatusUnauthorized:
		var errorResponse errorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return postClockingEventResponse{}, fmt.Errorf("erro de autenticação: %w", err)
		}
		return postClockingEventResponse{}, fmt.Errorf("não autorizado: %s", errorResponse.Message)

	default:
		body, _ := io.ReadAll(resp.Body)
		return postClockingEventResponse{}, fmt.Errorf(
			"erro inesperado (status %d): %s",
			resp.StatusCode,
			string(body),
		)
	}
}
