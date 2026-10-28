package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const vaultAddr = "http://127.0.0.1:8200"

// SecretStore reads the notification provider API key from HashiCorp Vault (KV v2).
type SecretStore struct {
	addr   string
	token  string
	client *http.Client
}

func NewSecretStore(addr, token string) *SecretStore {
	if addr == "" {
		addr = vaultAddr
	}
	return &SecretStore{addr: addr, token: token, client: &http.Client{Timeout: 5 * time.Second}}
}

// APIKey fetches secret/notify and returns the "api_key" field.
func (s *SecretStore) APIKey() (string, error) {
	req, err := http.NewRequest("GET", s.addr+"/v1/secret/data/notify", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Vault-Token", s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vault returned status %d", resp.StatusCode)
	}

	var body struct {
		Data struct {
			Data map[string]any `json:"data"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	key, _ := body.Data.Data["api_key"].(string)
	if key == "" {
		return "", fmt.Errorf("vault secret has no api_key field")
	}
	return key, nil
}

// LoadAPIKey returns the key and where it came from. There is no fallback:
// any Vault problem is returned as an error so the service refuses to start.
func LoadAPIKey(s *SecretStore) (string, string, error) {
	key, err := s.APIKey()
	if err != nil {
		return "", "", fmt.Errorf("loading api key from vault: %w", err)
	}
	log.Printf("notify-service loaded api key from vault (last4=%s)", last4(key))
	return key, "vault", nil
}
