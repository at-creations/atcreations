package catalog

import (
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

func TestSearchTermsAreQuotedAndFolded(t *testing.T) {
	f := ProductQuery{Text: "Gốm (.*)+"}.filter()
	and, ok := f["$and"].(bson.A)
	if !ok || len(and) != 2 {
		t.Fatalf("expected two term clauses, got %#v", f["$and"])
	}
	re := and[1].(bson.M)["search"].(bson.Regex)
	if re.Pattern != `\(\.\*\)\+` || re.Options != "i" {
		t.Fatalf("regex not escaped: %#v", re)
	}
	if first := and[0].(bson.M)["search"].(bson.Regex); first.Pattern != "gom" {
		t.Fatalf("term not folded: %#v", first)
	}
}

func TestApplyProductInputDerivedFields(t *testing.T) {
	var p Product
	applyProductInput(&p, &ProductInput{
		Status: StatusPublished, Name: NameInput{EN: "Vase", VI: "Bình gốm"}, PriceCents: ptr(2000), DiscountPercent: 25,
	})
	if !p.HasPrice || *p.FinalPriceCents != 1500 || p.SortNameVI != "binh gom" || p.Search != "vase binh gom" {
		t.Fatalf("unexpected derived fields: %+v", p)
	}
}
