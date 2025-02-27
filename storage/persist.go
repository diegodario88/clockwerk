package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	security "github.com/diegodario88/clockwerk/infra"
)

type UserCredentials struct {
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
		return fmt.Errorf("erro ao serializar credenciais: %v", err)
	}

	key := security.DeriveEncryptionKey()

	ciphertext, iv, err := security.Encrypt(jsonData, key)
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

func LoadCredentials() (UserCredentials, error) {
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

	key := security.DeriveEncryptionKey()

	plaintext, err := security.Decrypt(ciphertext, key, iv)
	if err != nil {
		return creds, fmt.Errorf("erro ao descriptografar: %v", err)
	}

	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return creds, fmt.Errorf("erro ao desserializar credenciais: %v", err)
	}

	return creds, nil
}

func DeleteCredentials() error {
	filePath := getCredentialsFilePath()
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("erro ao remover arquivo de credenciais: %v", err)
	}
	return nil
}

func getCredentialsFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Erro ao obter diretório do usuário: %v", err)
		return "clockwerk_credentials.enc"
	}
	return filepath.Join(homeDir, ".clockwerk_credentials.enc")
}
