package network

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateID(byteLength int) (string, error) {
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func GenerateAgentID() (string, error) {
	id, err := GenerateID(8)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("agent-%s", id), nil
}

func GenerateImplantID() (string, error) {
	id, err := GenerateID(8)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("implant-%s", id), nil
}

func GenerateRandomConnectionID() (string, error) {
	id, err := GenerateID(8)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("conn-%s", id), nil
}
