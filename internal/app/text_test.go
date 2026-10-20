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

func TestOrderedArticleImages(t *testing.T) {
	source := `<head><title>hidden</title></head><p>First</p><img src="../a.png" alt="A&#10;B&#27;[31m"><p>Second</p><img src="//cdn.example/b.png"><script><img src="https://bad.example"></script><img src="data:image/png,aaa" alt="Invalid">`
	blocks := ArticleBlocks(source, "https://example.com/news/post")
	if len(blocks) != 5 || blocks[0].Text != "First" || blocks[1].Source != "https://example.com/a.png" || blocks[1].Alt != "A B" || blocks[2].Text != "Second" || blocks[3].Source != "https://cdn.example/b.png" || blocks[4].Source != "" {
		t.Fatalf("%+v", blocks)
	}
	if got := PlainText(source); got != "First\n\n[image: A B]\n\nSecond\n\n[image]\n\n[image: Invalid]" {
		t.Fatal(got)
	}
	for _, source := range []string{"javascript:bad", "file:///tmp/a", "https://user:secret@example.com/a", "http://", "%zz", "", "https://example.com:bad/a"} {
		if ImageURL(source, "https://example.com") != "" {
			t.Fatal(source)
		}
	}
}
