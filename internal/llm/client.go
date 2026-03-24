package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"socialpilot/pkg/jsonx"
)

type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func New(baseURL, apiKey, model string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	transport := &http.Transport{}
	if useProxyFromEnv() {
		transport.Proxy = http.ProxyFromEnvironment
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		Model:   model,
		HTTP: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

func (c *Client) Chat(systemPrompt, userPrompt string) (string, error) {
	reqBody := chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	url := c.chatCompletionsURL()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "proxyconnect") {
			return "", fmt.Errorf("%w (hint: detected proxy connect failure, check HTTP_PROXY/HTTPS_PROXY or proxy service)", err)
		}
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var cr chatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", fmt.Errorf("invalid llm response: %w", err)
	}
	if resp.StatusCode >= 400 {
		if cr.Error != nil {
			return "", fmt.Errorf("llm error: %s", cr.Error.Message)
		}
		return "", fmt.Errorf("llm http status: %d", resp.StatusCode)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("empty llm choices")
	}
	return strings.TrimSpace(cr.Choices[0].Message.Content), nil
}

func (c *Client) chatCompletionsURL() string {
	base := strings.TrimRight(c.BaseURL, "/")
	lower := strings.ToLower(base)
	if strings.HasSuffix(lower, "/chat/completions") {
		return base
	}
	if strings.HasSuffix(lower, "/v1") {
		return base + "/chat/completions"
	}
	if strings.Contains(lower, "/v2/") || strings.HasSuffix(lower, "/v2") {
		return base + "/chat/completions"
	}
	u, err := url.Parse(base)
	if err == nil && (u.Path == "" || u.Path == "/") {
		return base + "/v1/chat/completions"
	}
	return base + "/chat/completions"
}

func useProxyFromEnv() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("SOCIALPILOT_USE_PROXY")))
	return v == "1" || v == "true" || v == "yes"
}

func (c *Client) ChatJSON(systemPrompt, userPrompt string) (string, error) {
	text, err := c.Chat(systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}
	j := jsonx.ExtractJSONObject(text)
	if j == "" {
		return "", fmt.Errorf("cannot extract json")
	}
	return j, nil
}
