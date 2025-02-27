package infra

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"
	"os"
)

func DeriveEncryptionKey() []byte {
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

func Encrypt(data []byte, key []byte) ([]byte, []byte, error) {
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

func Decrypt(ciphertext []byte, key []byte, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCFBDecrypter(block, iv)

	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}
