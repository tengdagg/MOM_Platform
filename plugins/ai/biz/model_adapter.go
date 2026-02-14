package biz

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
)

// ChatCompletionMessage OpenAI 兼容消息格式
type ChatCompletionMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // function
	Function FunctionCall `json:"function"`
}

// FunctionCall 函数调用
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolDefinition 工具定义（发送给 LLM）
type ToolDefinition struct {
	Type     string              `json:"type"` // function
	Function ToolFunctionDef     `json:"function"`
}

// ToolFunctionDef 工具函数定义
type ToolFunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ChatCompletionRequest 请求
type ChatCompletionRequest struct {
	Model       string                  `json:"model"`
	Messages    []ChatCompletionMessage `json:"messages"`
	Tools       []ToolDefinition        `json:"tools,omitempty"`
	MaxTokens   int                     `json:"max_tokens,omitempty"`
	Temperature float64                 `json:"temperature,omitempty"`
	Stream      bool                    `json:"stream"`
}

// ChatCompletionResponse 非流式响应
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Index        int                   `json:"index"`
		Message      ChatCompletionMessage `json:"message"`
		FinishReason string                `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// StreamDelta 流式响应片段
type StreamDelta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// StreamChoice 流式响应选择
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

// StreamResponse 流式响应
type StreamResponse struct {
	ID      string         `json:"id"`
	Choices []StreamChoice `json:"choices"`
}

// ModelAdapter 模型适配器
type ModelAdapter struct {
	config *AIModelConfig
	client *http.Client
}

// NewModelAdapter 创建模型适配器
func NewModelAdapter(config *AIModelConfig) *ModelAdapter {
	return &ModelAdapter{
		config: config,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// getBaseURL 获取 API 基础 URL
// 所有提供商均兼容 OpenAI Chat Completions API 格式
func (a *ModelAdapter) getBaseURL() string {
	baseURL := strings.TrimRight(a.config.BaseURL, "/")
	if baseURL == "" {
		switch a.config.Provider {
		case "openai", "openai_compatible":
			baseURL = "https://api.openai.com/v1"
		case "gemini":
			// Google Gemini OpenAI 兼容端点
			baseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
		case "qwen":
			// 阿里通义千问 DashScope OpenAI 兼容端点
			baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		case "deepseek":
			baseURL = "https://api.deepseek.com/v1"
		case "doubao":
			// 字节豆包 ARK OpenAI 兼容端点
			baseURL = "https://ark.cn-beijing.volces.com/api/v3"
		case "ollama":
			baseURL = "http://localhost:11434/v1"
		default:
			baseURL = "https://api.openai.com/v1"
		}
	}
	return baseURL
}

// ChatCompletion 非流式对话
func (a *ModelAdapter) ChatCompletion(ctx context.Context, messages []ChatCompletionMessage, tools []ToolDefinition) (*ChatCompletionResponse, error) {
	req := ChatCompletionRequest{
		Model:       a.config.ModelName,
		Messages:    messages,
		MaxTokens:   a.config.MaxTokens,
		Temperature: a.config.Temperature,
		Stream:      false,
	}
	if len(tools) > 0 {
		req.Tools = tools
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	url := a.getBaseURL() + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if a.config.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+a.config.APIKey)
	}

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求模型服务失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("模型服务返回错误 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

// StreamEvent 流式事件
type StreamEvent struct {
	Type         string     `json:"type"` // text_delta / tool_call_start / tool_call_delta / done / error
	Content      string     `json:"content,omitempty"`
	ToolCalls    []ToolCall `json:"toolCalls,omitempty"`
	FinishReason string     `json:"finishReason,omitempty"`
	Error        string     `json:"error,omitempty"`
}

// ChatCompletionStream 流式对话
func (a *ModelAdapter) ChatCompletionStream(ctx context.Context, messages []ChatCompletionMessage, tools []ToolDefinition) (<-chan StreamEvent, error) {
	req := ChatCompletionRequest{
		Model:       a.config.ModelName,
		Messages:    messages,
		MaxTokens:   a.config.MaxTokens,
		Temperature: a.config.Temperature,
		Stream:      true,
	}
	if len(tools) > 0 {
		req.Tools = tools
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	url := a.getBaseURL() + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if a.config.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+a.config.APIKey)
	}

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求模型服务失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("模型服务返回错误 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan StreamEvent, 64)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		// 增大缓冲区以处理大块 JSON
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- StreamEvent{Type: "done"}
				return
			}

			var streamResp StreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) == 0 {
				continue
			}

			choice := streamResp.Choices[0]
			delta := choice.Delta

			// 文本内容
			if delta.Content != "" {
				ch <- StreamEvent{
					Type:    "text_delta",
					Content: delta.Content,
				}
			}

			// 工具调用
			if len(delta.ToolCalls) > 0 {
				ch <- StreamEvent{
					Type:      "tool_call_delta",
					ToolCalls: delta.ToolCalls,
				}
			}

			// 结束原因
			if choice.FinishReason != nil {
				ch <- StreamEvent{
					Type:         "finish",
					FinishReason: *choice.FinishReason,
				}
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- StreamEvent{Type: "error", Error: err.Error()}
		}
	}()

	return ch, nil
}

// TestConnection 测试模型连通性
func (a *ModelAdapter) TestConnection(ctx context.Context) error {
	messages := []ChatCompletionMessage{
		{Role: "user", Content: "Hi, reply with just 'ok'."},
	}

	resp, err := a.ChatCompletion(ctx, messages, nil)
	if err != nil {
		return err
	}

	if len(resp.Choices) == 0 {
		return fmt.Errorf("模型未返回任何响应")
	}

	return nil
}
