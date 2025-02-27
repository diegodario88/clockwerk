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

type RequestFilter struct {
	ActivePlatformUser bool     `json:"activePlatformUser"`
	PageInfo           PageInfo `json:"pageInfo"`
	NameSearch         string   `json:"nameSearch"`
	Sort               Sort     `json:"sort"`
}

type PageInfo struct {
	Page     int    `json:"page"`
	PageSize string `json:"pageSize"`
}

type Sort struct {
	Field interface{} `json:"field"`
	Order string      `json:"order"`
}

type ClockingEventRequest struct {
	Filter RequestFilter `json:"filter"`
}

type ClockingEventResponse struct {
	Result []ClockingEvent `json:"result"`
}

type ClockingEvent struct {
	ID        string   `json:"id"`
	DateEvent string   `json:"dateEvent"`
	TimeEvent string   `json:"timeEvent"`
	Cnpj      string   `json:"cnpj"`
	Employee  Employee `json:"employee"`
	Platform  string   `json:"platform"`
}

type Employee struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Pis       string  `json:"pis"`
	Shift     string  `json:"shift"`
	Timetable string  `json:"timeTable"`
	Company   Company `json:"company"`
}

type Company struct {
	Cnpj string `json:"cnpj"`
	Name string `json:"name"`
}

type ErrorResponse struct {
	Message string `json:"message"`
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

func GetClockingEvents(token string) ([]ClockingEvent, error) {
	requestBody := ClockingEventRequest{
		Filter: RequestFilter{
			ActivePlatformUser: true,
			PageInfo: PageInfo{
				Page:     0,
				PageSize: "20",
			},
			NameSearch: "",
			Sort: Sort{
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

	req.Header.Set("accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("accept-language", "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7,es;q=0.6")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("dnt", "1")
	req.Header.Set("origin", "https://platform.senior.com.br")
	req.Header.Set("priority", "u=1, i")
	req.Header.Set(
		"referer",
		"https://platform.senior.com.br/login/?redirectTo=https%3A%2F%2Fplatform.senior.com.br%2Fsenior-x%2F",
	)
	req.Header.Set("sec-ch-ua", `"Not(A:Brand";v="99", "Google Chrome";v="133", "Chromium";v="133"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Linux"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set(
		"user-agent",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
	)
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar requisição: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var response ClockingEventResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
		}
		return response.Result, nil

	case http.StatusUnauthorized:
		var errorResponse ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return nil, fmt.Errorf("token expirado ou inválido: %w", err)
		}
		return nil, fmt.Errorf("autorização falhou: %s", errorResponse.Message)

	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro inesperado (status %d): %s", resp.StatusCode, string(body))
	}
}
