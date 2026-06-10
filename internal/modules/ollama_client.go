package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OllamaClient struct {
	baseURL string
	model   string
	client  *http.Client
	enabled bool
}

type OllamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func NewOllamaClient(baseURL, model string) *OllamaClient {
	return &OllamaClient{baseURL: baseURL, model: model, client: &http.Client{Timeout: 60 * time.Second}, enabled: true}
}

func (oc *OllamaClient) SetEnabled(enabled bool) { oc.enabled = enabled }

func (oc *OllamaClient) Generate(prompt string) (string, error) {
	if !oc.enabled { return "", fmt.Errorf("ollama disabled") }
	req := OllamaRequest{Model: oc.model, Prompt: prompt, Stream: false, Options: map[string]interface{}{"temperature": 0.3, "num_ctx": 8192, "num_predict": 512}}
	body, _ := json.Marshal(req)
	resp, err := oc.client.Post(oc.baseURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil { return "", err }
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var ollamaResp OllamaResponse
	if err := json.Unmarshal(data, &ollamaResp); err != nil { return "", err }
	return ollamaResp.Response, nil
}
