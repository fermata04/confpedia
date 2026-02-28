package search

import (
	"strings"
)

// vendorRules はベンダーキーワードと site: フィルタの優先順位付きリスト。
// マップではなくスライスを使うことで検出順序を決定論的にしている。
var vendorRules = []struct {
	keyword string
	site    string
}{
	{"cisco", "site:cisco.com OR site:community.cisco.com"},
	{"juniper", "site:juniper.net"},
	{"arista", "site:arista.com"},
	{"aws", "site:docs.aws.amazon.com"},
	{"azure", "site:learn.microsoft.com"},
	{"windows", "site:learn.microsoft.com"},
	{"linux", ""},
}

// 日本語意図キーワード → 英語の追加クエリ語
var intentMap = map[string]string{
	"設定":     "configuration",
	"コマンド":   "command",
	"トラブル":   "troubleshoot",
	"反映されない": "not working troubleshoot",
	"確認":     "check verify",
	"エラー":    "error troubleshoot",
	"接続":     "connection",
	"ルーティング": "routing",
}

// BuildQuery は日本語の自然言語入力から検索クエリ文字列を生成する。
func BuildQuery(input string) string {
	lower := strings.ToLower(input)

	// ベンダー検出（優先順位順に最初にマッチしたものを使用）
	siteFilter := ""
	for _, rule := range vendorRules {
		if strings.Contains(lower, rule.keyword) {
			siteFilter = rule.site
			break
		}
	}

	// 意図検出（追加キーワード）
	extras := []string{}
	for jp, en := range intentMap {
		if strings.Contains(input, jp) {
			extras = append(extras, en)
		}
	}

	// クエリ構築: 元の入力 + 英語追加語 + site:フィルタ
	parts := []string{input}
	parts = append(parts, extras...)
	if siteFilter != "" {
		parts = append(parts, siteFilter)
	}

	return strings.Join(parts, " ")
}
