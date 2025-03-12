package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type UserCredentials struct {
	Domain   string `json:"domain"`
	CPF      string `json:"cpf"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

type EncryptedData struct {
	Data string `json:"data"`
	IV   string `json:"iv"`
}

func SaveCredentials(creds UserCredentials) error {
	jsonData, err := json.Marshal(creds)
	if err != nil {
		log.Println("Erro ao serializar credenciais: %w", err)
		return fmt.Errorf("erro ao serializar credenciais: %v", err)
	}

	key := deriveEncryptionKey()

	ciphertext, iv, err := encrypt(jsonData, key)
	if err != nil {
		log.Println("Erro ao realizar encrypt: %w", err)
		return fmt.Errorf("erro ao criptografar: %v", err)
	}

	encData := EncryptedData{
		Data: base64.StdEncoding.EncodeToString(ciphertext),
		IV:   base64.StdEncoding.EncodeToString(iv),
	}

	encJson, err := json.Marshal(encData)
	if err != nil {
		log.Println("Erro ao serializar dados criptografados: %w", err)
		return fmt.Errorf("erro ao serializar dados criptografados: %v", err)
	}

	err = os.WriteFile(GetCredentialsFilePath(), encJson, 0600) // Permissão apenas para o usuário
	if err != nil {
		log.Println("Erro ao salvar arquivo de credenciais: %w", err)
		return fmt.Errorf("erro ao salvar arquivo de credenciais: %v", err)
	}

	return nil
}

func LoadCredentials() (UserCredentials, error) {
	var creds UserCredentials

	filePath := GetCredentialsFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return creds, nil
		}
		return creds, fmt.Errorf("erro ao ler arquivo de credenciais: %v", err)
	}

	var encData EncryptedData
	if err := json.Unmarshal(data, &encData); err != nil {
		log.Println("Erro ao desserializar dados criptografados: %w", err)
		return creds, fmt.Errorf("erro ao desserializar dados criptografados: %v", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encData.Data)
	if err != nil {
		log.Println("Erro ao decodificar ciphertext: %w", err)
		return creds, fmt.Errorf("erro ao decodificar ciphertext: %v", err)
	}

	iv, err := base64.StdEncoding.DecodeString(encData.IV)
	if err != nil {
		log.Println("Erro ao decodificar IV: %w", err)
		return creds, fmt.Errorf("erro ao decodificar IV: %v", err)
	}

	key := deriveEncryptionKey()

	plaintext, err := decrypt(ciphertext, key, iv)
	if err != nil {
		log.Println("Erro ao descriptografar: %w", err)
		return creds, fmt.Errorf("erro ao descriptografar: %v", err)
	}

	if err := json.Unmarshal(plaintext, &creds); err != nil {
		log.Println("Erro ao desserializar credenciais: %w", err)
		return creds, fmt.Errorf("erro ao desserializar credenciais: %v", err)
	}

	return creds, nil
}

func DeleteCredentials() error {
	filePath := GetCredentialsFilePath()
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		log.Println("Erro ao remover arquivo de credenciais: %w", err)
		return fmt.Errorf("erro ao remover arquivo de credenciais: %v", err)
	}
	return nil
}

func GetCredentialsFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Println("Erro ao obter diretório do usuário: %w", err)
		return "clockwerk_credentials.enc"
	}
	return filepath.Join(homeDir, ".clockwerk_credentials.enc")
}
