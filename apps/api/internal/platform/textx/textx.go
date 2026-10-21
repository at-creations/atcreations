// Package textx provides accent-insensitive normalization and slug generation
// for English and Vietnamese text.
package textx

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var stripMarks = runes.Remove(runes.In(unicode.Mn))

// Fold lowercases s and removes diacritics, so "Gốm Sứ" becomes "gom su".
// Vietnamese đ/Đ is not a combining sequence and is mapped explicitly.
func Fold(s string) string {
	s = strings.NewReplacer("đ", "d", "Đ", "D").Replace(s)
	out, _, err := transform.String(transform.Chain(norm.NFD, stripMarks, norm.NFC), s)
	if err != nil {
		out = s
	}
	return strings.ToLower(strings.Join(strings.Fields(out), " "))
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

const maxSlugLen = 80

// Slugify produces a URL-safe slug: lowercase ASCII letters, digits and single dashes.
func Slugify(s string) string {
	slug := strings.Trim(nonSlug.ReplaceAllString(Fold(s), "-"), "-")
	if len(slug) > maxSlugLen {
		slug = strings.TrimRight(slug[:maxSlugLen], "-")
	}
	return slug
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func ValidSlug(s string) bool { return len(s) <= 120 && slugPattern.MatchString(s) }

// SearchText builds the folded haystack stored alongside a document for search.
func SearchText(parts ...string) string {
	return Fold(strings.Join(parts, " "))
}
