package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strings"

	"nora/internal/protocol"
)

type AuthClient struct {
	httpClient *http.Client
	baseURL    string
}

type APIError struct {
	Message string `json:"message"`
}

func NewAuthClient(baseURL string, httpClient *http.Client) (*AuthClient, error) {
	if httpClient == nil {
		return nil, errors.New("HTTP client cannot be nil")
	}

	baseURL = strings.TrimRight(baseURL, "/")

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("Invalid API URL: %w", err)
	}

	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("API URL must be an absolute HTTP(S) URL")
	}

	return &AuthClient{
		httpClient: httpClient,
		baseURL:    baseURL,
	}, nil
}

type Account struct {
	AccountID string `json:"account_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
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

func (a *AuthClient) Authorize(ctx context.Context, token string) (*Account, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("No token provided")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/api/v1/auth", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	res, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	account, err := checkServerResponse(res)
	if err != nil {
		return nil, err
	}

	account.Token = token
	return account, nil
}

func (a *AuthClient) Register(ctx context.Context, name, email string) (*Account, error) {
	if name == "" || email == "" {
		return nil, errors.New("Not enough arguments to register account")
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return nil, fmt.Errorf("Invalid email: %w", err)
	}

	data := protocol.RegisterRequest{
		Username: name,
		Email:    email,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/v1/register", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Registration request failed: %w", err)
	}

	account, err := checkServerResponse(res)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Successfully registered as %s\n", account.Username)
	fmt.Printf("Your token is: %s\n", account.Token)
	return account, nil
}
