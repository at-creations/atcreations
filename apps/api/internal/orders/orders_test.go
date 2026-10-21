package orders

import (
	"context"
	"strings"
	"testing"

	"github.com/atcreations/atcreations/apps/api/internal/mail"
)

func TestTransitions(t *testing.T) {
	allowed := [][2]Status{
		{Pending, Confirmed}, {Pending, Cancelled}, {Confirmed, Processing}, {Processing, Shipped},
		{Shipped, Delivered}, {Delivered, Completed}, {Delivered, Refunded}, {Completed, Refunded},
	}
	for _, tr := range allowed {
		if !CanTransition(tr[0], tr[1]) {
			t.Errorf("%s -> %s should be allowed", tr[0], tr[1])
		}
	}
	denied := [][2]Status{
		{Cancelled, Pending}, {Refunded, Completed}, {Shipped, Cancelled}, {Pending, Shipped}, {Completed, Pending}, {Pending, Pending},
	}
	for _, tr := range denied {
		if CanTransition(tr[0], tr[1]) {
			t.Errorf("%s -> %s should be denied", tr[0], tr[1])
		}
	}
}

func TestRandomCode(t *testing.T) {
	seen := map[string]bool{}
	for range 1000 {
		c := randomCode(10)
		if len(c) != 10 || strings.Trim(c, codeAlphabet) != "" {
			t.Fatalf("bad code %q", c)
		}
		if seen[c] {
			t.Fatalf("duplicate code %q", c)
		}
		seen[c] = true
	}
}

func TestHashCodeNormalizesAndIsKeyed(t *testing.T) {
	a := &Service{pepper: []byte("pepper-a")}
	b := &Service{pepper: []byte("pepper-b")}
	if a.hashCode("abcd") != a.hashCode(" ABCD ") {
		t.Error("hash should ignore case and surrounding space")
	}
	if a.hashCode("ABCD") == b.hashCode("ABCD") {
		t.Error("hash must depend on the pepper")
	}
}

type captureSender struct{ last mail.Message }

func (c *captureSender) Send(_ context.Context, m mail.Message) error { c.last = m; return nil }

func TestEmailEscapesCustomerInput(t *testing.T) {
	cs := &captureSender{}
	m := NewMailer(cs, "AT Creations", "https://example.com")
	o := &Order{Number: "AT-1", Customer: Customer{Name: `<script>alert(1)</script>`, Email: "a@b.c"},
		Shipping: Address{Name: `<img src=x onerror=alert(1)>`}}
	if err := m.Confirmation(context.Background(), o, "CODE"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cs.last.HTML, "<script>") || strings.Contains(cs.last.HTML, "<img src=x") {
		t.Fatalf("HTML email contains unescaped input:\n%s", cs.last.HTML)
	}
}

func TestMoney(t *testing.T) {
	for in, want := range map[int64]string{0: "$0.00", 5: "$0.05", 12345: "$123.45", -250: "-$2.50"} {
		if got := money(in); got != want {
			t.Errorf("money(%d) = %s, want %s", in, got, want)
		}
	}
}
