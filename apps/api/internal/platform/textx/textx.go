// Package textx provides accent-insensitive normalization and slug generation
// for English and Vietnamese text.
package textx

import (
	"regexp"
	"strings"
	"unicode"

	"go.mongodb.org/mongo-driver/v2/bson"

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

// MaxSearchWords caps the words indexed per document (names come first, so they are always in).
const MaxSearchWords = 256

// SearchWords returns the distinct folded words of parts, in order, for a word-prefix index
// (a multikey index on the returned array). Words are split on anything that is not a letter
// or digit, so "AT-260101" yields "at" and "260101".
func SearchWords(parts ...string) []string {
	seen := map[string]bool{}
	words := []string{}
	for _, part := range parts {
		for _, w := range strings.FieldsFunc(Fold(part), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
			if len(w) > 64 || seen[w] {
				continue
			}
			seen[w] = true
			words = append(words, w)
			if len(words) == MaxSearchWords {
				return words
			}
		}
	}
	return words
}

// PrefixTerms turns a search query into at most 8 folded word prefixes, split like SearchWords.
func PrefixTerms(q string) []string {
	terms := strings.FieldsFunc(Fold(q), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	return terms[:min(len(terms), 8)]
}

// WordPrefixes is the condition for a SearchWords array field that has a word starting with
// every term. The anchored, case-sensitive patterns are index range scans on that field.
func WordPrefixes(terms []string) bson.M {
	all := make(bson.A, len(terms))
	for i, t := range terms {
		all[i] = regexpPrefix(t)
	}
	return bson.M{"$all": all}
}

func regexpPrefix(term string) bson.Regex {
	return bson.Regex{Pattern: "^" + regexp.QuoteMeta(term)}
}
