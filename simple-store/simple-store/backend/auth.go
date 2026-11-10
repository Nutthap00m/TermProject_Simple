package main

import (
	"context"
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ---------- Password hashing (PBKDF2-SHA256, stdlib) ----------

var pbkdf2Iterations = 600_000

func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, pbkdf2Iterations, 32)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pbkdf2$%d$%s$%s", pbkdf2Iterations,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

func CheckPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2" {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter < 1 {
		return false
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[2])
	want, err2 := base64.RawStdEncoding.DecodeString(parts[3])
	if err1 != nil || err2 != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iter, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}

// ---------- JWT (HS256, stdlib) ----------

type Claims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Role  string `json:"role"`
	Iat   int64  `json:"iat"`
	Exp   int64  `json:"exp"`
}

var b64 = base64.RawURLEncoding

const tokenTTL = 12 * time.Hour

func SignJWT(secret []byte, c Claims) (string, error) {
	c.Iat = time.Now().Unix()
	c.Exp = time.Now().Add(tokenTTL).Unix()
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	head := b64.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body := head + "." + b64.EncodeToString(payload)
	return body + "." + b64.EncodeToString(sign(secret, body)), nil
}

func sign(secret []byte, msg string) []byte {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(msg))
	return m.Sum(nil)
}

func ParseJWT(secret []byte, token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed token")
	}
	hb, err := b64.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("bad header")
	}
	var h struct {
		Alg string `json:"alg"`
	}
	if json.Unmarshal(hb, &h) != nil || h.Alg != "HS256" {
		return nil, errors.New("unsupported alg")
	}
	sig, err := b64.DecodeString(parts[2])
	if err != nil || !hmac.Equal(sig, sign(secret, parts[0]+"."+parts[1])) {
		return nil, errors.New("bad signature")
	}
	pb, err := b64.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("bad payload")
	}
	var c Claims
	if err := json.Unmarshal(pb, &c); err != nil {
		return nil, errors.New("bad payload")
	}
	if c.Exp < time.Now().Unix() {
		return nil, errors.New("token expired")
	}
	return &c, nil
}

// ---------- Middleware ----------

type ctxKey struct{}

func claimsFrom(ctx context.Context) *Claims {
	c, _ := ctx.Value(ctxKey{}).(*Claims)
	return c
}

// member: ต้อง login (member หรือ admin ก็ได้)
func (a *App) member(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		tok, ok := strings.CutPrefix(h, "Bearer ")
		if !ok || tok == "" {
			writeErr(w, http.StatusUnauthorized, "login required")
			return
		}
		c, err := ParseJWT([]byte(a.cfg.JWTSecret), tok)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, c)))
	})
}

// admin: ต้อง login และ role = admin
func (a *App) admin(next http.HandlerFunc) http.Handler {
	return a.member(func(w http.ResponseWriter, r *http.Request) {
		if claimsFrom(r.Context()).Role != "admin" {
			writeErr(w, http.StatusForbidden, "admin only")
			return
		}
		next(w, r)
	})
}
