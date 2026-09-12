package search

import (
	"strings"

	"n1h41/fw-oci/internal/tui/theme"
)

// highlightJSON colorizes a JSON document for display. It expects the output of
// json.MarshalIndent but handles any valid JSON.
func highlightJSON(s string) string {
	var b strings.Builder
	b.Grow(len(s) * 2)

	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '"':
			j := i + 1
			for j < len(s) {
				if s[j] == '\\' {
					j += 2
					continue
				}
				if s[j] == '"' {
					j++
					break
				}
				j++
			}
			if j > len(s) {
				j = len(s)
			}
			if isJSONKey(s, j) {
				b.WriteString(theme.JSONKey.Render(s[i:j]))
			} else {
				b.WriteString(theme.JSONString.Render(s[i:j]))
			}
			i = j
		case c == '-' || (c >= '0' && c <= '9'):
			j := i + 1
			for j < len(s) && isJSONNumberByte(s[j]) {
				j++
			}
			b.WriteString(theme.JSONNumber.Render(s[i:j]))
			i = j
		case strings.HasPrefix(s[i:], "true"):
			b.WriteString(theme.JSONLiteral.Render("true"))
			i += 4
		case strings.HasPrefix(s[i:], "false"):
			b.WriteString(theme.JSONLiteral.Render("false"))
			i += 5
		case strings.HasPrefix(s[i:], "null"):
			b.WriteString(theme.JSONLiteral.Render("null"))
			i += 4
		case c == '{' || c == '}' || c == '[' || c == ']' || c == ',' || c == ':':
			b.WriteString(theme.JSONPunct.Render(string(c)))
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// isJSONKey reports whether the string literal ending at end is an object key,
// i.e. the next non-space byte is a colon.
func isJSONKey(s string, end int) bool {
	for end < len(s) {
		switch s[end] {
		case ' ', '\t', '\n', '\r':
			end++
		case ':':
			return true
		default:
			return false
		}
	}
	return false
}

func isJSONNumberByte(c byte) bool {
	return c == '-' || c == '+' || c == '.' || c == 'e' || c == 'E' || (c >= '0' && c <= '9')
}
