package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func deriveEncryptionKey() []byte {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "unknown"
	}

	macAddress := "unknown"
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			if iface.HardwareAddr != nil {
				macAddress = iface.HardwareAddr.String()
				break
			}
		}
	}

	seedData := fmt.Sprintf("%s%s%s", hostname, homeDir, macAddress)

	hash := sha256.Sum256([]byte(seedData))
	return hash[:]
}

func encrypt(data []byte, key []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Println("Erro ao criar cifra: %w", err)
		return nil, nil, err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		log.Println("Erro ao gerar IV: %w", err)
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
		log.Println("Erro ao criar cifra: %w", err)
		return nil, err
	}

	stream := cipher.NewCFBDecrypter(block, iv)

	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}
