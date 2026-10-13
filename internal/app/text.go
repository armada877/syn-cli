package app

import (
	"golang.org/x/net/html"
	"strings"
	"unicode"
)

func SafeText(value string) string {
	var out strings.Builder
	runes := []rune(value)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == 27 {
			if i+1 < len(runes) && runes[i+1] == '[' {
				i += 2
				for i < len(runes) && !(runes[i] >= 0x40 && runes[i] <= 0x7e) {
					i++
				}
			} else if i+1 < len(runes) && (runes[i+1] == ']' || runes[i+1] == 'P' || runes[i+1] == '_' || runes[i+1] == '^') {
				i += 2
				for i < len(runes) {
					if runes[i] == 7 {
						break
					}
					if runes[i] == 27 && i+1 < len(runes) && runes[i+1] == '\\' {
						i++
						break
					}
					i++
				}
			}
			continue
		}
		if r == '\n' || r == '\t' {
			out.WriteRune(r)
		} else if !unicode.IsControl(r) && !unicode.Is(unicode.Cf, r) {
			out.WriteRune(r)
		}
	}
	return out.String()
}
func PlainText(source string) string {
	root, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return SafeText(source)
	}
	var out strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style" || n.Data == "head") {
			return
		}
		block := n.Type == html.ElementNode && strings.Contains("|p|div|br|li|h1|h2|h3|h4|blockquote|pre|tr|section|article|", "|"+n.Data+"|")
		if block {
			out.WriteByte('\n')
		}
		if n.Type == html.TextNode {
			out.WriteString(n.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
		if block {
			out.WriteByte('\n')
		}
	}
	walk(root)
	lines := []string{}
	for _, line := range strings.Split(SafeText(out.String()), "\n") {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n\n")
}
func ArticleText(title, url, content string) string {
	return SafeText(title) + "\n" + SafeText(url) + "\n\n" + PlainText(content)
}
