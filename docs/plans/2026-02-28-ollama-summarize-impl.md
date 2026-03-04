# Ollama コマンド整形機能 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** DuckDuckGo 検索結果をローカルの Ollama モデルに渡し、実際に使えるコマンド＋説明リストを生成してフロントに返す。

**Architecture:** `search/summarize.go` が標準ライブラリ（`net/http` + `encoding/json`）だけで Ollama の `/api/chat` エンドポイントを叩く。モデル名とURLは環境変数で設定可能。未起動・タイムアウト時はコマンドセクションをスキップして既存の検索結果カードのみ返す。

**Tech Stack:** Go 標準ライブラリ（追加 SDK なし）、Ollama `/api/chat` REST API、HTML/CSS/JavaScript

---

## 前提条件

- 作業ディレクトリ: `C:/Users/kaito/project/configuan/infra-search/`
- Go バイナリ: `C:/Users/kaito/AppData/Local/Temp/goinstall/go/bin/go.exe`
- Ollama が `http://localhost:11434` で起動していること（動作確認時のみ必要）

---

### Task 1: search/summarize.go を Ollama 版で実装

**Files:**
- Create: `infra-search/search/summarize.go`

**Step 1: search/summarize.go を作成**

`C:/Users/kaito/project/configuan/infra-search/search/summarize.go`:

```go
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
```

**Step 2: ビルド確認**

```bash
cd C:/Users/kaito/project/configuan/infra-search && C:/Users/kaito/AppData/Local/Temp/goinstall/go/bin/go.exe build ./...
```

Expected: エラーなし

**Step 3: コミット**

```bash
cd C:/Users/kaito/project/configuan/infra-search
git add search/summarize.go
git commit -m "feat: add Ollama-based command summarizer"
```

---

### Task 2: ハンドラーに Summarize を組み込む

**Files:**
- Modify: `infra-search/handlers/search.go`

**Step 1: handlers/search.go を Read ツールで読む**

**Step 2: SearchHandler を以下の内容で置き換える**

```go
package handlers

import (
	"infra-search/search"
	"net/http"

	"github.com/gin-gonic/gin"
)

type searchRequest struct {
	Query string `json:"query" binding:"required"`
}

func SearchHandler(c *gin.Context) {
	var req searchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query は必須です"})
		return
	}

	query := search.BuildQuery(req.Query)
	results, err := search.Search(query)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"results":  []interface{}{},
			"commands": nil,
			"message":  err.Error(),
		})
		return
	}

	if len(results) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"results":  []interface{}{},
			"commands": nil,
			"message":  "結果が見つかりませんでした",
		})
		return
	}

	commands, _ := search.Summarize(req.Query, results)
	c.JSON(http.StatusOK, gin.H{
		"results":  results,
		"commands": commands,
	})
}
```

**Step 3: ビルド確認**

```bash
cd C:/Users/kaito/project/configuan/infra-search && C:/Users/kaito/AppData/Local/Temp/goinstall/go/bin/go.exe build ./...
```

Expected: エラーなし

**Step 4: コミット**

```bash
cd C:/Users/kaito/project/configuan/infra-search
git add handlers/search.go
git commit -m "feat: call Summarize in handler, add commands to response"
```

---

### Task 3: フロントエンドにコマンドセクションを追加

**Files:**
- Modify: `infra-search/templates/index.html`
- Modify: `infra-search/static/style.css`

**Step 1: index.html を Read ツールで読む**

**Step 2: appendMessage 関数を修正（commands を保持できるようにする）**

現在:
```javascript
function appendMessage(role, content) {
  messages.push({ role, content });
  render();
}
```

修正後:
```javascript
function appendMessage(role, content, commands) {
  messages.push({ role, content, commands: commands || [] });
  render();
}
```

**Step 3: render() のボット表示部分を修正**

現在のボット表示（`m.role !== 'user'` 分岐内の `const cards = ...` と `return ...`）を以下に置き換える:

```javascript
const commandSection = (m.commands && m.commands.length > 0)
  ? `<div class="commands">
      <div class="commands-header">コマンド</div>
      ${m.commands.map(cmd =>
        `<div class="command-item">
          <div class="command-line">
            <code>$ ${escapeHtml(cmd.command)}</code>
            <button class="copy-btn" onclick="copyToClipboard('${escapeHtml(cmd.command)}')">コピー</button>
          </div>
          <div class="command-desc">${escapeHtml(cmd.description)}</div>
        </div>`
      ).join('')}
    </div>`
  : '';

const cards = m.content.map(r =>
  `<div class="card">
    <a href="${escapeHtml(r.url)}" target="_blank">${escapeHtml(r.title)}</a>
    <span class="source">${escapeHtml(r.source)}</span>
    <p>${escapeHtml(r.snippet)}</p>
   </div>`
).join('');
return `<div class="message bot">${commandSection}${cards || '結果が見つかりませんでした'}</div>`;
```

**Step 4: copyToClipboard 関数を追加（escapeHtml の直後に追加）**

```javascript
function copyToClipboard(text) {
  navigator.clipboard.writeText(text).catch(() => {
    const el = document.createElement('textarea');
    el.value = text;
    document.body.appendChild(el);
    el.select();
    document.execCommand('copy');
    document.body.removeChild(el);
  });
}
```

**Step 5: fetch 後の appendMessage 呼び出しを修正**

現在:
```javascript
appendMessage('bot', data.results || []);
```

修正後:
```javascript
appendMessage('bot', data.results || [], data.commands || []);
```

**Step 6: style.css の末尾にコマンドセクションのスタイルを追記**

```css
/* コマンドセクション */
.commands {
  background: #11111b;
  border: 1px solid #45475a;
  border-radius: 6px;
  padding: 10px 14px;
  margin-bottom: 10px;
}

.commands-header {
  font-size: 0.75em;
  color: #a6e3a1;
  font-weight: bold;
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.command-item {
  margin-bottom: 8px;
}

.command-line {
  display: flex;
  align-items: center;
  gap: 8px;
}

.command-line code {
  flex: 1;
  background: #1e1e2e;
  color: #cba6f7;
  padding: 4px 8px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 0.9em;
}

.copy-btn {
  padding: 3px 10px;
  font-size: 0.75em;
  background: #45475a;
  color: #cdd6f4;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-weight: normal;
}
.copy-btn:hover { background: #585b70; }

.command-desc {
  font-size: 0.82em;
  color: #a6adc8;
  margin-top: 3px;
  padding-left: 4px;
}
```

**Step 7: ビルド確認**

```bash
cd C:/Users/kaito/project/configuan/infra-search && C:/Users/kaito/AppData/Local/Temp/goinstall/go/bin/go.exe build ./...
```

Expected: エラーなし

**Step 8: コミット**

```bash
cd C:/Users/kaito/project/configuan/infra-search
git add templates/index.html static/style.css
git commit -m "feat: display command section with copy buttons in UI"
```

---

### Task 4: .gitignore に .env を追加

**Files:**
- Create or Modify: `infra-search/.gitignore`

**Step 1: .gitignore を確認**

`C:/Users/kaito/project/configuan/infra-search/.gitignore` が存在するか確認する。

**Step 2: .gitignore を作成または編集**

存在しない場合は以下の内容で新規作成:

```
.env
*.env
```

存在する場合は `.env` と `*.env` の2行を追記する。

**Step 3: コミット**

```bash
cd C:/Users/kaito/project/configuan/infra-search
git add .gitignore
git commit -m "chore: add .env to .gitignore"
```

---

## 動作確認手順

```bash
# 1. Ollama を起動してモデルを用意（未導入の場合）
ollama pull llama3.2

# 2. サーバー起動（デフォルト設定で OK）
cd C:/Users/kaito/project/configuan/infra-search
C:/Users/kaito/AppData/Local/Temp/goinstall/go/bin/go.exe run main.go

# 3. ブラウザで http://localhost:8080 を開く
# 4. "CiscoでBGPを設定したい" と入力して検索
# 5. コマンドセクションが表示され、コピーボタンが動作することを確認

# カスタムモデルを使いたい場合
OLLAMA_MODEL=gemma3 C:/Users/kaito/AppData/Local/Temp/goinstall/go/bin/go.exe run main.go

# Ollama 未起動時の確認（フォールバック動作）
# → コマンドセクションが表示されず、検索結果カードのみ表示される
```
