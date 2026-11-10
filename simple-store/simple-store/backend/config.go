package main

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	Port           string
	SupabaseURL    string
	SupabaseKey    string // service_role key — ห้ามส่งให้ frontend เด็ดขาด
	JWTSecret      string
	AllowedOrigins []string
	AdminEmail     string
	AdminPassword  string
	FakeStoreURL   string
	SeedOnStart    bool
}

func LoadConfig() (Config, error) {
	c := Config{
		Port:          getenv("PORT", "8080"),
		SupabaseURL:   strings.TrimRight(os.Getenv("SUPABASE_URL"), "/"),
		SupabaseKey:   os.Getenv("SUPABASE_SERVICE_KEY"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		AdminEmail:    strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		FakeStoreURL:  strings.TrimRight(getenv("FAKESTORE_URL", "https://fakestoreapi.com"), "/"),
		SeedOnStart:   getenv("SEED_ON_START", "true") == "true",
	}
	for _, o := range strings.Split(getenv("ALLOWED_ORIGINS", "http://localhost:5173"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			c.AllowedOrigins = append(c.AllowedOrigins, o)
		}
	}
	switch {
	case c.SupabaseURL == "" || c.SupabaseKey == "":
		return c, errors.New("SUPABASE_URL and SUPABASE_SERVICE_KEY are required")
	case len(c.JWTSecret) < 32:
		return c, errors.New("JWT_SECRET must be at least 32 characters")
	}
	return c, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
