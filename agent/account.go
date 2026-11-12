package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"

	"nora/internal/protocol"
)

type APIError struct {
	Message string `json:"message"`
}

type Account struct {
	AccountID string `json:"account_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Token     string `json:"token"`
}

func makeAuthentificationRequest(cfg *Config) (*http.Response, error) {
	if cfg.Authtoken == "" {
		return nil, errors.New("No token provided")
	}
	if cfg.ServerAddr == "" {
		return nil, errors.New("No server URL provided")
	}

	token := strings.TrimSpace(cfg.Authtoken)

	req, err := http.NewRequest(http.MethodGet, cfg.ServerAddr+"/api/v1/auth", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	return client.Do(req)
}

func makeRegisterRequest(name, email, serverURL string) (*http.Response, error) {
	if serverURL == "" || name == "" || email == "" {
		return nil, errors.New("Not  enough arguments to register account")
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return nil, errors.New("Invalid email address")
	}

	data := protocol.RegisterRequest{
		Username: name,
		Email:    email,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, serverURL+"/api/v1/register", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(req)
}

func checkServerResponse(resp *http.Response) (*Account, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		var profile Account
		if err := json.Unmarshal(body, &profile); err != nil {
			return nil, fmt.Errorf("failed to parse server JSON: %w", err)
		}
		return &profile, nil

	case http.StatusUnauthorized:
		return nil, fmt.Errorf("unauthorized (401): token is invalid or expired. Please run 'mycli login'")

	case http.StatusForbidden:
		return nil, fmt.Errorf("forbidden (403): your token lacks permission to access this resource")

	default:
		var apiErr APIError
		json.Unmarshal(body, &apiErr)

		if apiErr.Message != "" {
			return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, apiErr.Message)
		}

		return nil, fmt.Errorf("unexpected status (%d): %s", resp.StatusCode, string(body))
	}
}

func Authorize(cfg *Config) (*Account, error) {
	res, err := makeAuthentificationRequest(cfg)
	if err != nil {
		return nil, err
	}

	profile, err := checkServerResponse(res)
	if err != nil {
		return nil, err
	}

	profile.Token = cfg.Authtoken
	return profile, nil
}

func Register(name, email, serverURL string) error {
	res, err := makeRegisterRequest(name, email, serverURL)
	if err != nil {
		return err
	}
	profile, err := checkServerResponse(res)
	if err != nil {
		return err
	}
	fmt.Printf("Successfully registered as %s\n", profile.Username)
	fmt.Printf("Your token is: %s\n", profile.Token)
	return nil
}
