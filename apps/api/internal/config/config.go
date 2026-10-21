// Package config loads and validates process configuration from the environment.
// The process refuses to start when required values are missing or unsafe.
package config

import (
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Env      string `env:"APP_ENV" envDefault:"development"`
	HTTPAddr string `env:"HTTP_ADDR" envDefault:":8080"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	// Comma-separated origins allowed to call the API from a browser.
	CORSOrigins []string `env:"CORS_ALLOWED_ORIGINS" envSeparator:","`
	// CIDRs of reverse proxies whose X-Forwarded-For header is trusted.
	TrustedProxies []string `env:"TRUSTED_PROXIES" envSeparator:"," envDefault:"127.0.0.1/32,::1/128"`

	MongoURI      string        `env:"MONGO_URI,required,unset"`
	MongoDatabase string        `env:"MONGO_DATABASE" envDefault:"atcreations"`
	MongoTimeout  time.Duration `env:"MONGO_TIMEOUT" envDefault:"10s"`

	ClerkSecretKey string `env:"CLERK_SECRET_KEY,required,unset"`
	// Origins of the admin app; checked against the token's azp claim.
	ClerkAuthorizedParties []string `env:"CLERK_AUTHORIZED_PARTIES" envSeparator:","`
	// Where Clerk invitation emails send the invitee.
	ClerkInviteRedirectURL string `env:"CLERK_INVITE_REDIRECT_URL"`

	S3Endpoint        string `env:"S3_ENDPOINT,required"`
	S3Region          string `env:"S3_REGION" envDefault:"auto"`
	S3Bucket          string `env:"S3_BUCKET,required"`
	S3AccessKeyID     string `env:"S3_ACCESS_KEY_ID,required,unset"`
	S3SecretAccessKey string `env:"S3_SECRET_ACCESS_KEY,required,unset"`
	// Public base URL serving bucket objects, e.g. https://cdn.atcreations.ca
	CDNBaseURL string `env:"CDN_BASE_URL,required"`

	MailProvider  string `env:"MAIL_PROVIDER" envDefault:"log"` // log | sendgrid
	SendGridKey   string `env:"SENDGRID_API_KEY,unset"`
	MailFrom      string `env:"MAIL_FROM" envDefault:"orders@atcreations.ca"`
	MailFromName  string `env:"MAIL_FROM_NAME" envDefault:"AT Creations"`
	StoreName     string `env:"STORE_NAME" envDefault:"AT Creations"`
	StorefrontURL string `env:"STOREFRONT_URL" envDefault:"http://localhost:5173"`

	// Secret pepper for hashing order access codes and visitor fingerprints.
	HashPepper string `env:"HASH_PEPPER,required,unset"`

	PublicCheckoutEnabled bool `env:"FEATURE_PUBLIC_CHECKOUT" envDefault:"false"`
	JobsEnabled           bool `env:"JOBS_ENABLED" envDefault:"true"`

	TrustedProxyPrefixes []netip.Prefix `env:"-"`
}

func (c *Config) IsProduction() bool { return c.Env == "production" }

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	var errs []error

	switch c.Env {
	case "development", "test", "staging", "production":
	default:
		errs = append(errs, fmt.Errorf("APP_ENV must be development|test|staging|production, got %q", c.Env))
	}

	c.CORSOrigins = trimAll(c.CORSOrigins)
	for _, o := range c.CORSOrigins {
		if o == "*" {
			errs = append(errs, errors.New("CORS_ALLOWED_ORIGINS must list explicit origins, not *"))
			continue
		}
		if err := checkOrigin(o, c.IsProduction()); err != nil {
			errs = append(errs, fmt.Errorf("CORS_ALLOWED_ORIGINS: %w", err))
		}
	}
	c.ClerkAuthorizedParties = trimAll(c.ClerkAuthorizedParties)
	if c.IsProduction() && len(c.ClerkAuthorizedParties) == 0 {
		errs = append(errs, errors.New("CLERK_AUTHORIZED_PARTIES is required in production"))
	}

	for _, p := range trimAll(c.TrustedProxies) {
		prefix, err := netip.ParsePrefix(p)
		if err != nil {
			errs = append(errs, fmt.Errorf("TRUSTED_PROXIES: %w", err))
			continue
		}
		c.TrustedProxyPrefixes = append(c.TrustedProxyPrefixes, prefix.Masked())
	}

	if len(c.HashPepper) < 32 {
		errs = append(errs, errors.New("HASH_PEPPER must be at least 32 characters (use `openssl rand -hex 32`)"))
	}
	if c.IsProduction() && !strings.HasPrefix(c.ClerkSecretKey, "sk_live_") {
		errs = append(errs, errors.New("CLERK_SECRET_KEY must be a live key in production"))
	}

	switch c.MailProvider {
	case "log":
		if c.IsProduction() {
			errs = append(errs, errors.New("MAIL_PROVIDER=log is not allowed in production"))
		}
	case "sendgrid":
		if c.SendGridKey == "" {
			errs = append(errs, errors.New("SENDGRID_API_KEY is required when MAIL_PROVIDER=sendgrid"))
		}
	default:
		errs = append(errs, fmt.Errorf("MAIL_PROVIDER must be log|sendgrid, got %q", c.MailProvider))
	}

	for name, raw := range map[string]string{"CDN_BASE_URL": c.CDNBaseURL, "S3_ENDPOINT": c.S3Endpoint, "STOREFRONT_URL": c.StorefrontURL} {
		u, err := url.Parse(raw)
		switch {
		case err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https"):
			errs = append(errs, fmt.Errorf("%s must be an absolute http(s) URL", name))
		case c.IsProduction() && u.Scheme != "https":
			errs = append(errs, fmt.Errorf("%s must use https in production", name))
		}
	}
	c.CDNBaseURL = strings.TrimRight(c.CDNBaseURL, "/")
	c.StorefrontURL = strings.TrimRight(c.StorefrontURL, "/")

	return errors.Join(errs...)
}

func checkOrigin(origin string, requireHTTPS bool) error {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || (u.Path != "" && u.Path != "/") || u.RawQuery != "" {
		return fmt.Errorf("invalid origin %q", origin)
	}
	if u.Scheme != "https" && (requireHTTPS || u.Scheme != "http") {
		return fmt.Errorf("origin %q must use https", origin)
	}
	return nil
}

func trimAll(in []string) []string {
	out := in[:0]
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, strings.TrimRight(s, "/"))
		}
	}
	return out
}
