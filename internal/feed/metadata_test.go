package feed

import (
	"strings"
	"testing"

	"github.com/mmcdole/gofeed"
)

func TestArticleMetadata(t *testing.T) {
	for _, source := range []string{
		`<rss version="2.0"><channel><title>Feed</title><item><title>Title</title><author>alice@example.com</author><description>Summary</description></item></channel></rss>`,
		`<feed xmlns="http://www.w3.org/2005/Atom"><title>Feed</title><entry><title>Title</title><author><name>Alice</name></author><author><name>Bob</name></author><summary>Summary</summary></entry></feed>`,
	} {
		parsed, err := gofeed.NewParser().Parse(strings.NewReader(source))
		if err != nil {
			t.Fatal(err)
		}
		article := normalizeArticle(parsed.Items[0])
		want := "alice@example.com"
		if strings.Contains(source, "Atom") {
			want = "Alice, Bob"
		}
		if article.Author != want || article.Subtitle != "Summary" || article.Content != "Summary" {
			t.Fatal(article)
		}
	}
	item := &gofeed.Item{Title: "Title", Content: "Body", Description: "Summary", Authors: []*gofeed.Person{nil, {Name: " Alice "}, {Name: "Alice"}, {Email: " bob@example.com "}, {}}}
	article := normalizeArticle(item)
	if article.Author != "Alice, bob@example.com" {
		t.Fatal(article)
	}
	item.Authors = nil
	item.Author = &gofeed.Person{Name: "Carol"}
	item.Description = "Changed"
	changed := normalizeArticle(item)
	if changed.Author != "Carol" || changed.Key != article.Key {
		t.Fatal(changed)
	}
	item.Author = nil
	item.Description = ""
	missing := normalizeArticle(item)
	if missing.Author != "" || missing.Subtitle != "" || missing.Key != article.Key {
		t.Fatal(missing)
	}
}
