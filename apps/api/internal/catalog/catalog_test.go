package catalog

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func ptr(v int64) *int64 { return &v }

func TestFinalPrice(t *testing.T) {
	cases := []struct {
		price    *int64
		discount int
		want     *int64
	}{
		{nil, 10, nil},
		{ptr(1000), 0, ptr(1000)},
		{ptr(1000), 15, ptr(850)},
		{ptr(999), 33, ptr(669)}, // 669.33 rounds down
		{ptr(5), 50, ptr(3)},     // 2.5 rounds half up
		{ptr(0), 90, ptr(0)},
	}
	for _, c := range cases {
		got := FinalPrice(c.price, c.discount)
		if (got == nil) != (c.want == nil) || (got != nil && *got != *c.want) {
			t.Errorf("FinalPrice(%v, %d) = %v, want %v", c.price, c.discount, got, c.want)
		}
	}
}

func TestPublicFilterAlwaysRestrictsToPublished(t *testing.T) {
	f := ProductQuery{Published: true, Status: StatusDraft}.filter()
	if f["status"] != StatusPublished {
		t.Fatalf("public query must force published, got %v", f["status"])
	}
}

func TestSearchTermsAreFoldedWordPrefixes(t *testing.T) {
	// Punctuation never reaches a pattern: terms are letters and digits only.
	f := ProductQuery{Text: "Gốm (.*)+ Bình2"}.filter()
	all, ok := f["searchWords"].(bson.M)["$all"].(bson.A)
	if !ok || len(all) != 2 {
		t.Fatalf("expected two prefix clauses, got %#v", f["searchWords"])
	}
	for i, want := range []string{"^gom", "^binh2"} {
		if re := all[i].(bson.Regex); re.Pattern != want || re.Options != "" {
			t.Fatalf("clause %d = %#v, want anchored case-sensitive %q", i, re, want)
		}
	}
}

func TestApplyProductInputDerivedFields(t *testing.T) {
	var p Product
	applyProductInput(&p, &ProductInput{
		Status: StatusPublished, Name: NameInput{EN: "Vase", VI: "Bình gốm"}, PriceCents: ptr(2000), DiscountPercent: 25,
	})
	if !p.HasPrice || *p.FinalPriceCents != 1500 || p.SortNameVI != "binh gom" || strings.Join(p.SearchWords, " ") != "vase binh gom" {
		t.Fatalf("unexpected derived fields: %+v", p)
	}
}
