package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type CommandItem struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// Ollama API のリクエスト・レスポンス型
type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

var ollamaClient = &http.Client{Timeout: 30 * time.Second}

// Summarize は検索結果スニペットを Ollama に渡し、
// コマンド＋説明のリストを返す。
// OLLAMA_URL が未設定、または Ollama が応答しない場合は nil, nil を返す。
func Summarize(query string, results []SearchResult) ([]CommandItem, error) {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama3.2"
	}

	// 検索結果スニペットを連結してプロンプトを構築
	var sb strings.Builder
	sb.WriteString("ユーザーの質問: ")
	sb.WriteString(query)
	sb.WriteString("\n\n検索結果:\n")
	for i, r := range results {
		fmt.Fprintf(&sb, "%d. %s\n%s\n\n", i+1, r.Title, r.Snippet)
	}

	systemPrompt := `あなたはインフラエンジニア向けのアシスタントです。
与えられた検索結果から、実際にターミナルで使えるコマンドを抽出してください。
必ず以下のJSON形式のみを返してください。説明文や前置きは不要です。

{"commands": [{"command": "実際のコマンド", "description": "このコマンドの目的を1行で"}]}`

	reqBody := ollamaRequest{
		Model: model,
		Messages: []ollamaMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: sb.String()},
		},
		Stream: false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil
	}

	resp, err := ollamaClient.Post(
		ollamaURL+"/api/chat",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		// Ollama 未起動などの接続エラーはスキップ
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, nil
	}

	raw := strings.TrimSpace(ollamaResp.Message.Content)

	// JSON 部分だけを取り出す（```json ... ``` で囲まれている場合も対応）
	if idx := strings.Index(raw, "{"); idx > 0 {
		raw = raw[idx:]
	}
	if idx := strings.LastIndex(raw, "}"); idx >= 0 {
		raw = raw[:idx+1]
	}

	var parsed struct {
		Commands []CommandItem `json:"commands"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, nil
	}

	return parsed.Commands, nil
}
