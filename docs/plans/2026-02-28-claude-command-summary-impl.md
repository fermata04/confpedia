# Claude コマンド整形機能 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** DuckDuckGo 検索結果を Claude API（Haiku）に渡し、実際に使えるコマンド＋説明の形式に整形してフロントに返す。

**Architecture:** Go バックエンドが `search/summarize.go` で Anthropic SDK を呼び、検索結果スニペットからコマンドリストを生成する。APIキーは `ANTHROPIC_API_KEY` 環境変数から読む。フロントはコマンドセクションをカードの上に表示し、各行にコピーボタンを付ける。

**Tech Stack:** Go, `github.com/anthropics/anthropic-sdk-go`, `claude-haiku-4-5-20251001`, HTML/CSS/JavaScript

---

## 前提条件

- 作業ディレクトリ: `C:/Users/kaito/project/configuan/infra-search/`
- Go バイナリ: `C:/Users/kaito/AppData/Local/Temp/goinstall/go/bin/go.exe`
- `ANTHROPIC_API_KEY` 環境変数に有効な Anthropic API キーが設定されていること

---

### Task 1: Anthropic SDK の追加と summarize.go の実装

**Files:**
- Modify: `infra-search/go.mod`（SDK 追加）
- Create: `infra-search/search/summarize.go`

**Step 1: Anthropic SDK を追加**

```bash
cd C:/Users/kaito/project/configuan/infra-search && C:/Users/kaito/AppData/Local/Temp/goinstall/go/bin/go.exe get github.com/anthropics/anthropic-sdk-go
```

Expected: `go.mod` と `go.sum` が更新される

**Step 2: search/summarize.go を作成**

`C:/Users/kaito/project/configuan/infra-search/search/summarize.go`:

```go
package search

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type CommandItem struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// Summarize は検索結果スニペットを Claude API に渡し、
// コマンド＋説明のリストを返す。
// ANTHROPIC_API_KEY が未設定の場合は nil, nil を返す。
func Summarize(query string, results []SearchResult) ([]CommandItem, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, nil
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

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

	msg, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude_Haiku_4_5,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(sb.String())),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude api error: %w", err)
	}

	if len(msg.Content) == 0 {
		return nil, nil
	}

	raw := msg.Content[0].Text

	// JSON 部分だけを取り出す（```json ... ``` で囲まれている場合も対応）
	raw = strings.TrimSpace(raw)
	if idx := strings.Index(raw, "{"); idx > 0 {
		raw = raw[idx:]
	}
	if idx := strings.LastIndex(raw, "}"); idx >= 0 {
		raw = raw[:idx+1]
	}

	var resp struct {
		Commands []CommandItem `json:"commands"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("parse response error: %w", err)
	}

	return resp.Commands, nil
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
git add search/summarize.go go.mod go.sum
git commit -m "feat: add Claude API summarize to extract commands"
```

---

### Task 2: ハンドラーに Summarize を組み込む

**Files:**
- Modify: `infra-search/handlers/search.go`

**Step 1: handlers/search.go を読む**

`C:/Users/kaito/project/configuan/infra-search/handlers/search.go` を Read ツールで確認する。

**Step 2: `search.Search()` の後に `search.Summarize()` を追加**

現在の `c.JSON(http.StatusOK, gin.H{"results": results})` を以下に置き換える:

```go
commands, _ := search.Summarize(req.Query, results)
c.JSON(http.StatusOK, gin.H{
    "results":  results,
    "commands": commands,
})
```

修正後の `SearchHandler` 全体:

```go
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

**Step 1: index.html を読む**

`C:/Users/kaito/project/configuan/infra-search/templates/index.html` を Read ツールで確認する。

**Step 2: render() 関数のボット応答部分を修正**

現在の `render()` 内のボット表示（`m.role !== 'user'` の分岐）を以下に置き換える:

```javascript
// commands セクション（存在する場合のみ）
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

`appendMessage` 関数を修正して `commands` も保持できるようにする:

```javascript
function appendMessage(role, content, commands) {
  messages.push({ role, content, commands: commands || [] });
  render();
}
```

fetch 後の呼び出しを修正:

```javascript
const data = await res.json();
appendMessage('bot', data.results || [], data.commands || []);
```

`copyToClipboard` 関数を追加（`escapeHtml` の後などに追加）:

```javascript
function copyToClipboard(text) {
  navigator.clipboard.writeText(text).catch(() => {
    // フォールバック: textarea を使う古い方法
    const el = document.createElement('textarea');
    el.value = text;
    document.body.appendChild(el);
    el.select();
    document.execCommand('copy');
    document.body.removeChild(el);
  });
}
```

**Step 3: style.css にコマンドセクションのスタイルを追加**

`C:/Users/kaito/project/configuan/infra-search/static/style.css` の末尾に以下を追記する:

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

**Step 4: ビルド確認**

```bash
cd C:/Users/kaito/project/configuan/infra-search && C:/Users/kaito/AppData/Local/Temp/goinstall/go/bin/go.exe build ./...
```

Expected: エラーなし

**Step 5: コミット**

```bash
cd C:/Users/kaito/project/configuan/infra-search
git add templates/index.html static/style.css
git commit -m "feat: display command section with copy buttons in UI"
```

---

### Task 4: .gitignore に .env を追加

**Files:**
- Create: `infra-search/.gitignore`（または Modify）

**Step 1: .gitignore を確認・作成**

`C:/Users/kaito/project/configuan/infra-search/.gitignore` が存在するか確認する。存在しない場合は新規作成:

```
.env
*.env
```

存在する場合は `.env` と `*.env` の行を追記する。

**Step 2: コミット**

```bash
cd C:/Users/kaito/project/configuan/infra-search
git add .gitignore
git commit -m "chore: add .env to .gitignore"
```

---

## 動作確認手順

```bash
# 1. API キーを環境変数に設定
export ANTHROPIC_API_KEY=sk-ant-...

# 2. サーバー起動
cd C:/Users/kaito/project/configuan/infra-search
C:/Users/kaito/AppData/Local/Temp/goinstall/go/bin/go.exe run main.go

# 3. ブラウザで http://localhost:8080 を開く
# 4. "CiscoでBGPを設定したい" と入力して検索
# 5. コマンドセクションが表示され、コピーボタンが動作することを確認
```

**API キー未設定時の確認:**

```bash
# ANTHROPIC_API_KEY を未設定で起動
unset ANTHROPIC_API_KEY
C:/Users/kaito/AppData/Local/Temp/goinstall/go/bin/go.exe run main.go
# → コマンドセクションが表示されず、検索結果カードのみ表示される（既存動作を維持）
```
