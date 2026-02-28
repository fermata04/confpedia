# Claude コマンド整形機能 設計書

**日付**: 2026-02-28

## 概要

DuckDuckGo 検索結果を Claude API に渡し、実際に使えるコマンド＋説明の形式に整形して返す機能を追加する。API キーはバックエンド（Go）で環境変数から読み取り、ブラウザには露出しない。

## アーキテクチャ

```
POST /api/search
    ↓
handlers/search.go
    ↓ BuildQuery()
search/query.go
    ↓ Search()
search/fetch.go → DuckDuckGo（既存）
    ↓ []SearchResult（スニペット×10件）
search/summarize.go（新規）
    ↓ Anthropic API（ANTHROPIC_API_KEY 環境変数）
    ↓ []CommandItem
handlers/search.go → JSON レスポンス
```

## 新規ファイル

### `search/summarize.go`

```go
type CommandItem struct {
    Command     string `json:"command"`
    Description string `json:"description"`
}

func Summarize(query string, results []SearchResult) ([]CommandItem, error)
```

- 使用モデル: `claude-haiku-4-5-20251001`
- system プロンプト: インフラエンジニア向けアシスタント、JSON のみ返す
- user プロンプト: クエリ + 検索結果スニペットの連結テキスト
- レスポンス: `{"commands": [{"command": "...", "description": "..."}]}`
- `ANTHROPIC_API_KEY` 未設定時はスキップ（`nil, nil` を返す）

## 変更ファイル

### `handlers/search.go`

`search.Summarize()` を呼び出して `commands` フィールドをレスポンスに追加する。

### `go.mod`

`github.com/anthropics/anthropic-sdk-go` を追加。

### `templates/index.html` + `static/style.css`

`commands` 配列が存在する場合、検索結果カードの上に「コマンド」セクションを表示する。各コマンドにコピーボタンを付ける。

## レスポンス形式

```json
{
  "results": [...],
  "commands": [
    { "command": "router bgp 65000", "description": "BGP プロセスを開始する" },
    { "command": "neighbor 10.0.0.1 remote-as 65001", "description": "隣接ルーターを設定する" }
  ]
}
```

## セキュリティ

- `ANTHROPIC_API_KEY` は環境変数で管理し、コードに埋め込まない
- `.gitignore` に `.env` を追加する
- API キー未設定時はコマンド整形をスキップし既存動作を維持する
