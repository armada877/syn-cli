package domain_test

import (
	"errors"
	"fmt"
	"syn-cli/internal/domain"
	"testing"
)

func TestNormalizeFeedURL(t *testing.T) {
	cases := []struct{ input, want string }{
		{"HTTPS://EXAMPLE.COM/Feed%2fItems?q=A%2FB#section", "https://example.com/Feed%2fItems?q=A%2FB"},
		{"http://Example.COM:8080/rss", "http://example.com:8080/rss"},
		{"https://[2001:DB8::1]/RSS", "https://[2001:db8::1]/RSS"},
		{"https://example.com?", "https://example.com?"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := domain.NormalizeFeedURL(tc.input)
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
			again, err := domain.NormalizeFeedURL(got)
			if err != nil || again != got {
				t.Fatalf("normalization changed %q to %q: %v", got, again, err)
			}
		})
	}
}

func TestInvalidFeedURL(t *testing.T) {
	for _, input := range []string{"", "example.com/rss", "//example.com/rss", "ftp://example.com/rss", "https:///rss", "https://:80/rss", "http://[broken", "https://example.com/%zz"} {
		t.Run(input, func(t *testing.T) {
			got, err := domain.NormalizeFeedURL(input)
			if got != "" || !errors.Is(err, domain.ErrInvalidURL) {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
}

func TestDomainErrors(t *testing.T) {
	for _, target := range []error{domain.ErrNotFound, domain.ErrInvalidURL} {
		if !errors.Is(fmt.Errorf("operation: %w", target), target) {
			t.Fatalf("cannot inspect %v", target)
		}
	}
	if errors.Is(domain.ErrNotFound, domain.ErrInvalidURL) {
		t.Fatal("domain errors must differ")
	}
}
