# リッチコマンド出力 設計ドキュメント

> 作成日: 2026-03-04

---

## 概要

ネットワーク機器（Cisco 等）の対話型コマンドフローと、パラメータ・背景説明の充実により、検索結果ページを参照しなくてもコマンドが使えるレベルの情報を提供する。

---

## 課題

1. **対話型コマンドの流れが再現されない** — `configure terminal` → `interface` → 各設定コマンドの階層が失われる
2. **説明があっさりすぎる** — 1行 description では目的・パラメータの意味が不明で、結局検索結果ページを見に行く必要がある

---

## 変更ファイル

| ファイル | 変更内容 |
|---|---|
| `search/summarize.go` | `CommandItem` に `Prompt`, `Purpose`, `Params` を追加、システムプロンプト更新 |
| `templates/index.html` | `prompt` の有無で表示切り替え、`purpose`・`params` の表示追加 |
| `static/style.css` | ターミナル風プロンプト表示用スタイル追加 |

---

## データ構造

### 変更前

```go
type CommandItem struct {
    Step        int      `json:"step"`
    Command     string   `json:"command"`
    Description string   `json:"description"`
    Options     []string `json:"options"`
}
```

### 変更後

```go
type CommandItem struct {
    Step        int      `json:"step"`
    Prompt      string   `json:"prompt"`      // 対話型CLIのプロンプト。Linux コマンドは ""
    Command     string   `json:"command"`
    Description string   `json:"description"`
    Purpose     string   `json:"purpose"`     // なぜ必要か 1〜2文
    Params      []string `json:"params"`      // コマンド内パラメータ値の意味
    Options     []string `json:"options"`
}
```

---

## システムプロンプト

```
あなたはインフラエンジニア向けのアシスタントです。
与えられた検索結果から、実際にターミナルで使えるコマンドを抽出してください。
必ず以下のJSON形式のみを返してください。説明文や前置きは不要です。

フィールドの説明:
- step: 実行順序（1から連番）
- prompt: CLIプロンプト（Cisco等の対話型コマンドの場合のみ。例: "Router#", "Router(config-if)#"。Linuxコマンドは空文字列）
- command: 実行するコマンド
- description: このコマンドの目的を1行で
- purpose: なぜこのコマンドが必要か1〜2文で説明
- params: コマンド内の具体的なパラメータ値の意味（例: ["192.168.1.1: ルーターのIPアドレス"]）。パラメータがない場合は空配列
- options: 主要なオプションフラグの説明（例: ["-t: 構文チェックのみ実行"]）。ない場合は空配列

{"commands": [{"step": 1, "prompt": "", "command": "コマンド", "description": "目的を1行で", "purpose": "なぜ必要か1〜2文", "params": ["値: 説明"], "options": ["-x: 説明"]}]}
```

---

## UI 表示

### prompt あり（ネットワーク機器）

```
Router(config-if)# ip address 192.168.1.1 255.255.255.0    [コピー]
  インターフェースにIPアドレスを割り当てる
  > IPアドレスを設定するために必要。no shutdown と合わせて実行する。
  • 192.168.1.1: ルーターのIPアドレス
  • 255.255.255.0: /24 サブネットマスク
```

### prompt なし（Linux）

```
$ nginx -t                                                  [コピー]
  設定ファイルの構文チェック
  > 起動・リロード前に構文エラーがないか確認するために実行する。
  • -t: 構文チェックのみ（サービス起動しない）
```

---

## 採用しなかった案

| 案 | 理由 |
|---|---|
| interactive / standalone で型を分ける | LLM の出力制御が難しく UI 分岐も複雑 |
| セッション全体を生テキスト1フィールド | コマンドごとのコピーボタン・params 表示が不可 |
