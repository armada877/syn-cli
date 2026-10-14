package app

import (
	"strings"
	"testing"
)

func TestArticleDisplayMetadata(t *testing.T) {
	if got := ArticleDisplayText("Title", "URL", "<p>Summary</p>", "Alice\x1b[31m", "<p>Body one</p><p>Body two</p>"); got != "Title\nURL\n\nSummary\n\nBy Alice\n\nBody one\n\nBody two" {
		t.Fatal(got)
	}
	for _, subtitle := range []string{"", "<p>Body one</p>  <p>Body two</p>", "Body one Body two"} {
		got := ArticleDisplayText("Title", "URL", subtitle, "", "<p>Body one</p><p>Body two</p>")
		if strings.Count(got, "Body one") != 1 || strings.Contains(got, "By ") {
			t.Fatal(got)
		}
	}
	if ArticleByline("<script>bad</script>\x1b]0;bad\a") != "" {
		t.Fatal("Unsafe author")
	}
}
