package textx

import (
	"strings"
	"testing"
)

func TestFold(t *testing.T) {
	cases := map[string]string{
		"Gốm Sứ Bát Tràng": "gom su bat trang",
		"Đồ  thủ công\t":   "do thu cong",
		"Plain ASCII":      "plain ascii",
		"":                 "",
	}
	for in, want := range cases {
		if got := Fold(in); got != want {
			t.Errorf("Fold(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Bình hoa gốm (lớn)":   "binh-hoa-gom-lon",
		"  --Hello, World!-- ": "hello-world",
		"Đèn lồng Hội An":      "den-long-hoi-an",
		"!!!":                  "",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
		if got := Slugify(in); got != "" && !ValidSlug(got) {
			t.Errorf("Slugify(%q) produced invalid slug %q", in, got)
		}
	}
}

func TestValidSlug(t *testing.T) {
	for _, s := range []string{"a", "abc-123", "x-y-z"} {
		if !ValidSlug(s) {
			t.Errorf("ValidSlug(%q) = false", s)
		}
	}
	for _, s := range []string{"", "-a", "a-", "a--b", "A", "a b", "../etc"} {
		if ValidSlug(s) {
			t.Errorf("ValidSlug(%q) = true", s)
		}
	}
}

func TestSearchWords(t *testing.T) {
	got := SearchWords("Bình Gốm sứ", "Gốm đẹp — AT-260101", "binh")
	want := []string{"binh", "gom", "su", "dep", "at", "260101"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("SearchWords = %v, want %v", got, want)
	}
	if terms := PrefixTerms("  GỐM  bì "); strings.Join(terms, ",") != "gom,bi" {
		t.Fatalf("PrefixTerms = %v", terms)
	}
	many := strings.Repeat("word ", 10) + strings.Repeat("x", 70)
	if w := SearchWords(many); len(w) != 1 {
		t.Fatalf("duplicates and over-long words must be dropped: %v", w)
	}
}
