package util

import "strings"

// SplitCSV 拆分逗号分隔字符串，去掉每段空白并丢弃空段；空串返回 nil。
func SplitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
