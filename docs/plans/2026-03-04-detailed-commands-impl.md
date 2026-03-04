# 詳細コマンドサマリー 実装プラン

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** コマンドサマリーに実行順序（`step`）とオプション説明（`options`）を追加する。

**Architecture:** `CommandItem` 構造体に2フィールドを追加し、システムプロンプトを更新。フロントエンドはステップ番号・オプション箇条書きを表示するよう変更。

**Tech Stack:** Go (Gin), Vanilla JS, Ollama (gpt-oss:20b)

**Design Doc:** `docs/plans/2026-03-04-detailed-commands-design.md`

---

### Task 1: CommandItem 構造体とシステムプロンプトの更新

**Files:**
- Modify: `search/summarize.go`

**Step 1: `CommandItem` に `Step` と `Options` を追加**

`search/summarize.go` の `CommandItem` を以下に変更:

```go
type CommandItem struct {
	Step        int      `json:"step"`
	Command     string   `json:"command"`
	Description string   `json:"description"`
	Options     []string `json:"options"`
}
```

**Step 2: システムプロンプトを更新**

`systemPrompt` 変数を以下に変更:

```go
systemPrompt := `あなたはインフラエンジニア向けのアシスタントです。
与えられた検索結果から、実際にターミナルで使えるコマンドを抽出してください。
必ず以下のJSON形式のみを返してください。説明文や前置きは不要です。
step は実行順序（1から連番）、options は主要なオプションの説明を配列で記載してください。

{"commands": [{"step": 1, "command": "実際のコマンド", "description": "このコマンドの目的を1行で", "options": ["-x: オプションの説明"]}]}`
```

**Step 3: ビルド確認**

```bash
cd C:/Users/kaito/project/configuan/infra-search
go build ./...
```

Expected: エラーなし

**Step 4: Commit**

```bash
git add search/summarize.go
git commit -m "feat: add step and options to CommandItem"
```

---

### Task 2: UI 表示の更新

**Files:**
- Modify: `templates/index.html`

**Step 1: コマンド表示部分を更新**

`index.html` の `commandSection` の `${m.commands.map(...)}` 部分を以下に変更:

```javascript
${m.commands.map(cmd =>
  `<div class="command-item">
    <div class="command-line">
      <span class="command-step">${cmd.step}.</span>
      <code>$ ${escapeHtml(cmd.command)}</code>
      <button class="copy-btn" onclick="copyToClipboard('${escapeHtml(cmd.command)}')">コピー</button>
    </div>
    <div class="command-desc">${escapeHtml(cmd.description)}</div>
    ${cmd.options && cmd.options.length > 0
      ? `<ul class="command-options">${cmd.options.map(o => `<li>${escapeHtml(o)}</li>`).join('')}</ul>`
      : ''}
  </div>`
).join('')}
```

**Step 2: CSS を追加**

`static/style.css` に以下を追加:

```css
.command-step {
  font-weight: bold;
  margin-right: 4px;
  color: var(--mauve);
}

.command-options {
  margin: 4px 0 0 16px;
  padding: 0;
  list-style: disc;
  font-size: 0.85em;
  color: var(--subtext0);
}

.command-options li {
  margin: 2px 0;
}
```

**Step 3: 動作確認**

サーバーを起動し、ブラウザで `http://localhost:8080` を開く:

```bash
go run main.go
```

「nginx 設定確認」などで検索し、コマンドにステップ番号とオプション箇条書きが表示されることを確認する。

**Step 4: Commit**

```bash
git add templates/index.html static/style.css
git commit -m "feat: display step number and options in command summary"
```
