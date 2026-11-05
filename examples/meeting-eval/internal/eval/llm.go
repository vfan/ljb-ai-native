package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResult struct {
	Text  string
	Usage Usage
}

type LLMClient struct {
	cfg  Config
	http *http.Client
}

func NewLLMClient(cfg Config) *LLMClient {
	return &LLMClient{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

func (c *LLMClient) Chat(ctx context.Context, messages []ChatMessage) (ChatResult, error) {
	if c.cfg.APIKey == "" {
		return ChatResult{}, errors.New("未配置 DEEPSEEK_API_KEY，无法调用真实模型")
	}

	payload := map[string]any{
		"model":           c.cfg.Model,
		"messages":        messages,
		"temperature":     c.cfg.Temperature,
		"response_format": map[string]string{"type": "json_object"},
		// 结构化抽取和逐条判定不需要思维链：关掉后延迟更低、成本更可控，
		// temperature 也才会真正生效。
		"thinking": map[string]string{"type": "disabled"},
		"stream":   false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ChatResult{}, err
	}

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		started := time.Now()
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return ChatResult{}, err
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

		response, err := c.http.Do(request)
		if err != nil {
			lastErr = fmt.Errorf("请求失败：%w", err)
		} else {
			raw, readErr := io.ReadAll(io.LimitReader(response.Body, 4<<20))
			response.Body.Close()
			if readErr != nil {
				lastErr = fmt.Errorf("读取响应失败：%w", readErr)
			} else if response.StatusCode >= 400 {
				lastErr = fmt.Errorf("HTTP %d：%s", response.StatusCode, truncate(string(raw), 400))
				if response.StatusCode >= 400 && response.StatusCode < 500 && response.StatusCode != 429 {
					return ChatResult{}, lastErr
				}
			} else {
				var parsed chatResponse
				if err := json.Unmarshal(raw, &parsed); err != nil {
					return ChatResult{}, fmt.Errorf("响应不是合法 JSON：%w", err)
				}
				if len(parsed.Choices) == 0 {
					return ChatResult{}, errors.New("响应中没有 choices")
				}
				return ChatResult{
					Text: parsed.Choices[0].Message.Content,
					Usage: Usage{
						LatencyMs:        int(time.Since(started).Milliseconds()),
						PromptTokens:     parsed.Usage.PromptTokens,
						CompletionTokens: parsed.Usage.CompletionTokens,
						Cost:             c.cfg.Cost(parsed.Usage.PromptTokens, parsed.Usage.CompletionTokens),
					},
				}, nil
			}
		}
		if attempt == 0 {
			time.Sleep(1200 * time.Millisecond)
		}
	}
	return ChatResult{}, lastErr
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func truncate(text string, limit int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "…"
}
