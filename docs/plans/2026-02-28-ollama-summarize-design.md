# Ollama コマンド整形機能 設計書

**日付**: 2026-02-28

## 概要

DuckDuckGo 検索結果をローカルの Ollama モデルに渡し、実際に使えるコマンド＋説明の形式に整形する。Claude API は使わず、Ollama の `/api/chat` エンドポイントを標準ライブラリのみで直接叩く。

## アーキテクチャ

```
POST /api/search
    ↓
handlers/search.go（変更なし）
    ↓ Summarize()
search/summarize.go（Ollama 版）
    ↓ HTTP POST（標準ライブラリ）
http://localhost:11434/api/chat
    ↓ []CommandItem
handlers/search.go → JSON レスポンス
```

## 環境変数

| 変数 | デフォルト | 説明 |
|---|---|---|
| `OLLAMA_URL` | `http://localhost:11434` | Ollama エンドポイント |
| `OLLAMA_MODEL` | `llama3.2` | 使用するモデル名 |

## search/summarize.go

```go
type CommandItem struct {
    Command     string `json:"command"`
    Description string `json:"description"`
}

func Summarize(query string, results []SearchResult) ([]CommandItem, error)
```

- 外部依存なし（標準ライブラリのみ）
- Ollama `/api/chat` に `stream: false` でリクエスト
- レスポンスの `message.content` から `{"commands": [...]}` を JSON パース

## フォールバック

| 状況 | 動作 |
|---|---|
| `OLLAMA_URL` 未設定 | スキップ（`nil, nil` を返す） |
| Ollama 接続失敗・タイムアウト（30秒） | スキップ（`nil, nil` を返す） |
| JSON パース失敗 | スキップ（`nil, nil` を返す） |

いずれの場合も検索結果カードのみ表示する（既存動作を維持）。

## 変更ファイル

- **置き換え**: `search/summarize.go`（Ollama 版に書き直し）
- **変更なし**: `handlers/search.go`、`templates/index.html`、`static/style.css`
- **不要**: Anthropic SDK（`go.mod` から削除）
