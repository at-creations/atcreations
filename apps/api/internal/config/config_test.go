package config

import (
	"strings"
	"testing"
)

func base() Config {
	return Config{
		Env: "production", MongoURI: "mongodb://x", ClerkSecretKey: "sk_live_abc",
		ClerkAuthorizedParties: []string{"https://admin.example.com"},
		CORSOrigins:            []string{"https://example.com"},
		TrustedProxies:         []string{"10.0.0.0/8"},
		S3Endpoint:             "https://acc.r2.cloudflarestorage.com", S3Bucket: "b",
		CDNBaseURL: "https://cdn.example.com", StorefrontURL: "https://example.com",
		HashPepper: strings.Repeat("a", 64), MailProvider: "sendgrid", SendGridKey: "SG.x",
		RedisURL: "redis://:pw@redis:6379/0",
	}
}

func TestValidateAcceptsSaneProduction(t *testing.T) {
	c := base()
	if err := c.validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsUnsafeSettings(t *testing.T) {
	mutations := map[string]func(*Config){
		"wildcard CORS":       func(c *Config) { c.CORSOrigins = []string{"*"} },
		"http CORS in prod":   func(c *Config) { c.CORSOrigins = []string{"http://example.com"} },
		"short pepper":        func(c *Config) { c.HashPepper = "short" },
		"test clerk key":      func(c *Config) { c.ClerkSecretKey = "sk_test_abc" },
		"log mailer in prod":  func(c *Config) { c.MailProvider = "log" },
		"no azp in prod":      func(c *Config) { c.ClerkAuthorizedParties = nil },
		"bad proxy CIDR":      func(c *Config) { c.TrustedProxies = []string{"nope"} },
		"http CDN in prod":    func(c *Config) { c.CDNBaseURL = "http://cdn.example.com" },
		"unknown environment": func(c *Config) { c.Env = "prod" },
		"no redis in prod":    func(c *Config) { c.RedisURL = "" },
		"bad redis scheme":    func(c *Config) { c.RedisURL = "http://redis:6379" },
	}
	for name, mutate := range mutations {
		c := base()
		mutate(&c)
		if err := c.validate(); err == nil {
			t.Errorf("%s: expected validation error", name)
		}
	}
}
