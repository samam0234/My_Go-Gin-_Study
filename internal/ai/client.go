package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Result struct {
	Text         string
	InputTokens  int
	OutputTokens int
}

type Client struct {
	Provider  string
	Model     string
	APIKey    string
	OllamaURL string
	HTTP      *http.Client
}

func New(provider, model, apiKey, ollamaURL string) *Client {
	if strings.TrimSpace(model) == "" {
		model = "mock"
	}
	if strings.TrimSpace(ollamaURL) == "" {
		ollamaURL = "http://localhost:11434"
	}
	return &Client{
		Provider:  strings.TrimSpace(provider),
		Model:     model,
		APIKey:    apiKey,
		OllamaURL: strings.TrimRight(ollamaURL, "/"),
		HTTP:      &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) Complete(ctx context.Context, system string, messages []Message) (Result, error) {
	switch c.Provider {
	case "", "mock":
		return mockResult(system, messages), nil
	case "ollama":
		return c.completeOllama(ctx, system, messages)
	case "anthropic":
		return c.completeAnthropic(ctx, system, messages)
	default:
		return Result{}, fmt.Errorf("알 수 없는 AI_PROVIDER %q", c.Provider)
	}
}

func (c *Client) Stream(ctx context.Context, system string, messages []Message, emit func(string) error) (Result, error) {
	switch c.Provider {
	case "", "mock":
		result := mockResult(system, messages)
		if err := emitChunks(result.Text, emit); err != nil {
			return Result{}, err
		}
		return result, nil
	case "ollama":
		return c.streamOllama(ctx, system, messages, emit)
	case "anthropic":
		return c.streamAnthropic(ctx, system, messages, emit)
	default:
		return Result{}, fmt.Errorf("알 수 없는 AI_PROVIDER %q", c.Provider)
	}
}

func mockResult(system string, messages []Message) Result {
	user := lastUser(messages)
	text := MockComplete(system, user)
	return Result{Text: text, InputTokens: Estimate(system + user), OutputTokens: Estimate(text)}
}

func MockComplete(system, user string) string {
	if strings.Contains(system, "[발췌") {
		if label := firstExcerptLabel(system); label != "" {
			return "발췌에 적힌 내용을 바탕으로 답합니다. (" + label + ")"
		}
		return "문서에서 찾을 수 없습니다."
	}
	trimmed := strings.TrimSpace(user)
	runes := []rune(trimmed)
	if len(runes) > 80 {
		trimmed = string(runes[:80])
	}
	return "요청을 확인했습니다. 핵심만 짧게 다듬으면 다음과 같습니다: " + trimmed
}

func firstExcerptLabel(system string) string {
	const marker = "[발췌 "
	i := strings.Index(system, marker)
	if i < 0 {
		return ""
	}
	rest := system[i+len(marker):]
	j := strings.Index(rest, "]")
	if j < 0 {
		return ""
	}
	return "발췌 " + rest[:j]
}

func Estimate(s string) int {
	n := utf8.RuneCountInString(s) / 4
	if n < 1 {
		return 1
	}
	return n
}

func lastUser(messages []Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	if len(messages) == 0 {
		return ""
	}
	return messages[len(messages)-1].Content
}

func emitChunks(text string, emit func(string) error) error {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}
	for i := 0; i < len(runes); i += 12 {
		j := i + 12
		if j > len(runes) {
			j = len(runes)
		}
		if err := emit(string(runes[i:j])); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) completeOllama(ctx context.Context, system string, messages []Message) (Result, error) {
	body := map[string]any{
		"model":    c.modelName("llama3.2"),
		"messages": withSystem(system, messages),
		"stream":   false,
	}
	var decoded struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		PromptEval int `json:"prompt_eval_count"`
		Eval       int `json:"eval_count"`
	}
	if err := c.postJSON(ctx, c.OllamaURL+"/api/chat", body, nil, &decoded); err != nil {
		return Result{}, err
	}
	return Result{Text: decoded.Message.Content, InputTokens: positive(decoded.PromptEval), OutputTokens: positive(decoded.Eval)}, nil
}

func (c *Client) streamOllama(ctx context.Context, system string, messages []Message, emit func(string) error) (Result, error) {
	body := map[string]any{
		"model":    c.modelName("llama3.2"),
		"messages": withSystem(system, messages),
		"stream":   true,
	}
	res, err := c.post(ctx, c.OllamaURL+"/api/chat", body, nil)
	if err != nil {
		return Result{}, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 4000))
		return Result{}, fmt.Errorf("ollama %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}
	sc := bufio.NewScanner(res.Body)
	var full strings.Builder
	inTok, outTok := 1, 1
	for sc.Scan() {
		var line struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done       bool `json:"done"`
			PromptEval int  `json:"prompt_eval_count"`
			Eval       int  `json:"eval_count"`
		}
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			continue
		}
		if line.Message.Content != "" {
			full.WriteString(line.Message.Content)
			if err := emit(line.Message.Content); err != nil {
				return Result{}, err
			}
		}
		if line.Done {
			inTok, outTok = positive(line.PromptEval), positive(line.Eval)
		}
	}
	return Result{Text: full.String(), InputTokens: inTok, OutputTokens: outTok}, sc.Err()
}

func (c *Client) completeAnthropic(ctx context.Context, system string, messages []Message) (Result, error) {
	if c.APIKey == "" {
		return Result{}, fmt.Errorf("ANTHROPIC_API_KEY가 없습니다")
	}
	body := map[string]any{
		"model":      c.modelName("claude-opus-5-5"),
		"max_tokens": 800,
		"system":     system,
		"messages":   messages,
	}
	var decoded struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			Input  int `json:"input_tokens"`
			Output int `json:"output_tokens"`
		} `json:"usage"`
	}
	headers := map[string]string{
		"x-api-key":         c.APIKey,
		"anthropic-version": "2023-06-01",
	}
	if err := c.postJSON(ctx, "https://api.anthropic.com/v1/messages", body, headers, &decoded); err != nil {
		return Result{}, err
	}
	if decoded.StopReason == "refusal" {
		return Result{Text: "이 요청에는 답할 수 없습니다.", InputTokens: positive(decoded.Usage.Input), OutputTokens: 1}, nil
	}
	var text strings.Builder
	for _, block := range decoded.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}
	return Result{Text: text.String(), InputTokens: positive(decoded.Usage.Input), OutputTokens: positive(decoded.Usage.Output)}, nil
}

func (c *Client) streamAnthropic(ctx context.Context, system string, messages []Message, emit func(string) error) (Result, error) {
	if c.APIKey == "" {
		return Result{}, fmt.Errorf("ANTHROPIC_API_KEY가 없습니다")
	}
	body := map[string]any{
		"model":      c.modelName("claude-opus-5-5"),
		"max_tokens": 800,
		"system":     system,
		"messages":   messages,
		"stream":     true,
	}
	headers := map[string]string{
		"x-api-key":         c.APIKey,
		"anthropic-version": "2023-06-01",
	}
	res, err := c.post(ctx, "https://api.anthropic.com/v1/messages", body, headers)
	if err != nil {
		return Result{}, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 4000))
		return Result{}, fmt.Errorf("anthropic %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}
	sc := bufio.NewScanner(res.Body)
	var full strings.Builder
	inTok, outTok := 1, 1
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
			Usage struct {
				Input  int `json:"input_tokens"`
				Output int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}
		if event.Delta.Text != "" {
			full.WriteString(event.Delta.Text)
			if err := emit(event.Delta.Text); err != nil {
				return Result{}, err
			}
		}
		if event.Usage.Output > 0 || event.Usage.Input > 0 {
			inTok, outTok = positive(event.Usage.Input), positive(event.Usage.Output)
		}
	}
	return Result{Text: full.String(), InputTokens: inTok, OutputTokens: outTok}, sc.Err()
}

func (c *Client) modelName(fallback string) string {
	if c.Model == "" || c.Model == "mock" {
		return fallback
	}
	return c.Model
}

func withSystem(system string, messages []Message) []Message {
	out := make([]Message, 0, len(messages)+1)
	if strings.TrimSpace(system) != "" {
		out = append(out, Message{Role: "system", Content: system})
	}
	return append(out, messages...)
}

func positive(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func (c *Client) postJSON(ctx context.Context, url string, body any, headers map[string]string, dest any) error {
	res, err := c.post(ctx, url, body, headers)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 400 {
		return fmt.Errorf("ai http %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}
	if dest == nil {
		return nil
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("ai 응답 파싱: %w", err)
	}
	return nil
}

func (c *Client) post(ctx context.Context, url string, body any, headers map[string]string) (*http.Response, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.HTTP.Do(req)
}
