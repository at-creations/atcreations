package orders

import (
	"encoding/json"
	"testing"
)

func TestAddressNormalisesCountry(t *testing.T) {
	var a Address
	if err := json.Unmarshal([]byte(`{"name":"A","phone":"1","line1":"1","city":"c","postalCode":"p","country":" ca "}`), &a); err != nil {
		t.Fatal(err)
	}
	if a.Country != "CA" {
		t.Fatalf("country = %q, want CA", a.Country)
	}
	if err := json.Unmarshal([]byte(`{"name":"A","country":"CA","admin":true}`), &a); err == nil {
		t.Fatal("unknown fields must still be rejected")
	}
}
