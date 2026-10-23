package analytics

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestNormalizePath(t *testing.T) {
	ok := map[string]string{
		"/":                      "/",
		"/en/products/Vase/":     "/en/products/vase",
		"/vi?utm_source=x":       "/vi",
		"/en/products/a#reviews": "/en/products/a",
		"/en/products/bình-hoa":  "/en/products/bình-hoa",
	}
	for in, want := range ok {
		got, valid := NormalizePath(in)
		if !valid || got != want {
			t.Errorf("NormalizePath(%q) = %q, %v; want %q", in, got, valid, want)
		}
	}
	for _, in := range []string{"", "en", "//evil.com/x", "/a b", "/x\n", "https://x.com/", "/" + string(make([]byte, 250))} {
		if got, valid := NormalizePath(in); valid {
			t.Errorf("NormalizePath(%q) = %q, want rejected", in, got)
		}
	}
}

func TestReferrerHost(t *testing.T) {
	s := New(nil0(), nil, "pepper", Options{StorefrontURL: "https://www.atcreations.ca"})
	cases := map[string]string{
		"":                                     DirectReferrer,
		"not a url":                            DirectReferrer,
		"javascript:alert(1)":                  DirectReferrer,
		"https://www.atcreations.ca/en":        DirectReferrer, // internal navigation
		"https://atcreations.ca/vi":            DirectReferrer,
		"https://www.Google.com/search?q=vase": "google.com",
		"http://l.facebook.com/l.php?u=x":      "l.facebook.com",
	}
	for in, want := range cases {
		if got := s.ReferrerHost(in); got != want {
			t.Errorf("ReferrerHost(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDeviceAndBots(t *testing.T) {
	devices := map[string]string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) Mobile/15E148":      "mobile",
		"Mozilla/5.0 (Linux; Android 15; Pixel 9) Mobile Safari/537.36":             "mobile",
		"Mozilla/5.0 (iPad; CPU OS 18_0 like Mac OS X)":                             "tablet",
		"Mozilla/5.0 (Linux; Android 14; SM-X710) Safari/537.36":                    "tablet",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/140.0 Safari/537.36":      "desktop",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 15_0) Version/18.0 Safari/605.1.15": "desktop",
	}
	for ua, want := range devices {
		if got := Device(ua); got != want {
			t.Errorf("Device(%q) = %q, want %q", ua, got, want)
		}
		if IsBot(ua) {
			t.Errorf("IsBot(%q) = true for a browser", ua)
		}
	}
	for _, ua := range []string{"", "Googlebot/2.1", "facebookexternalhit/1.1", "curl/8.9", "Mozilla/5.0 HeadlessChrome/140"} {
		if !IsBot(ua) {
			t.Errorf("IsBot(%q) = false, want true", ua)
		}
	}
}

func TestNormalizeCountry(t *testing.T) {
	for in, want := range map[string]string{"ca": "CA", " VN ": "VN", "XX": "", "T1": "", "C1": "", "CAN": "", "": ""} {
		if got := NormalizeCountry(in); got != want {
			t.Errorf("NormalizeCountry(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPeriodsUseStoreDays(t *testing.T) {
	loc, err := time.LoadLocation("America/Vancouver")
	if err != nil {
		t.Fatal(err)
	}
	s := New(nil0(), nil, "pepper", Options{Location: loc})
	// 02:00 UTC on Nov 3 is still Nov 2 in Vancouver.
	s.now = func() time.Time { return time.Date(2026, 11, 3, 2, 0, 0, 0, time.UTC) }

	cur, prev := s.periods(7)
	if got := cur.start.Format(time.DateOnly); got != "2026-10-27" {
		t.Errorf("current start = %s, want 2026-10-27", got)
	}
	if got := cur.end.Format(time.DateOnly); got != "2026-11-03" {
		t.Errorf("current end = %s, want 2026-11-03 (exclusive)", got)
	}
	if !prev.end.Equal(cur.start) || prev.start.Format(time.DateOnly) != "2026-10-20" {
		t.Errorf("previous period = %v..%v", prev.start, prev.end)
	}
	if day, _ := s.dayOf(s.now()); day != "2026-11-02" {
		t.Errorf("dayOf = %s, want 2026-11-02", day)
	}
}

// nil0 returns a database handle that is never used (the client connects lazily).
func nil0() *mongo.Database {
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://127.0.0.1:1"))
	if err != nil {
		panic(err)
	}
	return client.Database("unused")
}
