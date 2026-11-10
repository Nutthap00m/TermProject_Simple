package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func init() { pbkdf2Iterations = 1000 } // ให้เทสต์เร็ว

const testSecret = "test-secret-test-secret-test-secret-123"

func newTestApp(t *testing.T) (*App, *MemStore, http.Handler) {
	t.Helper()
	st := NewMemStore()
	app := &App{
		cfg:   Config{JWTSecret: testSecret, AllowedOrigins: []string{"https://shop.vercel.app", "https://*.vercel.app"}, AdminEmail: "admin@shop.test", AdminPassword: "admin-pass-123", SeedOnStart: false},
		store: st,
		hc:    http.DefaultClient,
	}
	if err := app.seedAdmin(t.Context()); err != nil {
		t.Fatal(err)
	}
	return app, st, app.routes()
}

func call(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func tokenFrom(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var r authResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil || r.Token == "" {
		t.Fatalf("no token in response: %s", rec.Body.String())
	}
	return r.Token
}

func register(t *testing.T, h http.Handler, email string) string {
	rec := call(t, h, "POST", "/api/auth/register", "", map[string]string{"email": email, "name": "Tester", "password": "password123"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	return tokenFrom(t, rec)
}

func TestPasswordAndJWT(t *testing.T) {
	h, err := HashPassword("secret-pass")
	if err != nil || !CheckPassword("secret-pass", h) || CheckPassword("wrong", h) {
		t.Fatal("password hash/verify failed")
	}
	tok, _ := SignJWT([]byte(testSecret), Claims{Sub: "u1", Role: "member"})
	if c, err := ParseJWT([]byte(testSecret), tok); err != nil || c.Sub != "u1" {
		t.Fatalf("parse: %v", err)
	}
	if _, err := ParseJWT([]byte("another-secret-another-secret-12345"), tok); err == nil {
		t.Fatal("token with wrong secret must be rejected")
	}
	if _, err := ParseJWT([]byte(testSecret), tok[:len(tok)-2]+"xx"); err == nil {
		t.Fatal("tampered token must be rejected")
	}
}

func TestGuestCanBrowseButNotOrder(t *testing.T) {
	_, st, h := newTestApp(t)
	st.CreateProduct(t.Context(), ProductInput{Title: "Backpack", Price: 10, Category: "bags"})

	if rec := call(t, h, "GET", "/api/products", "", nil); rec.Code != 200 || !strings.Contains(rec.Body.String(), "Backpack") {
		t.Fatalf("guest list products: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(t, h, "GET", "/api/products/999", "", nil); rec.Code != 404 {
		t.Fatalf("missing product should 404, got %d", rec.Code)
	}
	for _, p := range []string{"/api/orders", "/api/me"} {
		if rec := call(t, h, "GET", p, "", nil); rec.Code != 401 {
			t.Fatalf("guest %s should be 401, got %d", p, rec.Code)
		}
	}
	if rec := call(t, h, "POST", "/api/orders", "", map[string]any{}); rec.Code != 401 {
		t.Fatalf("guest order should be 401, got %d", rec.Code)
	}
}

func TestRegisterLoginValidation(t *testing.T) {
	_, _, h := newTestApp(t)
	register(t, h, "a@shop.test")

	if rec := call(t, h, "POST", "/api/auth/register", "", map[string]string{"email": "a@shop.test", "name": "X", "password": "password123"}); rec.Code != 409 {
		t.Fatalf("duplicate email should 409, got %d", rec.Code)
	}
	if rec := call(t, h, "POST", "/api/auth/register", "", map[string]string{"email": "bad", "name": "X", "password": "password123"}); rec.Code != 400 {
		t.Fatalf("bad email should 400, got %d", rec.Code)
	}
	if rec := call(t, h, "POST", "/api/auth/register", "", map[string]string{"email": "b@shop.test", "name": "X", "password": "short"}); rec.Code != 400 {
		t.Fatalf("short password should 400, got %d", rec.Code)
	}
	if rec := call(t, h, "POST", "/api/auth/login", "", map[string]string{"email": "a@shop.test", "password": "password123"}); rec.Code != 200 {
		t.Fatalf("login should succeed, got %d", rec.Code)
	}
	if rec := call(t, h, "POST", "/api/auth/login", "", map[string]string{"email": "a@shop.test", "password": "nope-nope"}); rec.Code != 401 {
		t.Fatalf("wrong password should 401, got %d", rec.Code)
	}
}

func TestProfileAndChangePassword(t *testing.T) {
	_, _, h := newTestApp(t)
	tok := register(t, h, "p@shop.test")

	rec := call(t, h, "PUT", "/api/me", tok, map[string]string{"name": "New Name"})
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "New Name") {
		t.Fatalf("update profile: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(t, h, "POST", "/api/me/password", tok, map[string]string{"current_password": "wrong-pass", "new_password": "brand-new-pass"}); rec.Code != 400 {
		t.Fatalf("wrong current password should 400, got %d", rec.Code)
	}
	if rec := call(t, h, "POST", "/api/me/password", tok, map[string]string{"current_password": "password123", "new_password": "short"}); rec.Code != 400 {
		t.Fatalf("short new password should 400, got %d", rec.Code)
	}
	if rec := call(t, h, "POST", "/api/me/password", tok, map[string]string{"current_password": "password123", "new_password": "brand-new-pass"}); rec.Code != 200 {
		t.Fatalf("change password: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(t, h, "POST", "/api/auth/login", "", map[string]string{"email": "p@shop.test", "password": "password123"}); rec.Code != 401 {
		t.Fatal("old password must stop working")
	}
	if rec := call(t, h, "POST", "/api/auth/login", "", map[string]string{"email": "p@shop.test", "password": "brand-new-pass"}); rec.Code != 200 {
		t.Fatal("new password must work")
	}
}

func TestMemberOrderUsesServerPrices(t *testing.T) {
	_, st, h := newTestApp(t)
	p, _ := st.CreateProduct(t.Context(), ProductInput{Title: "Ring", Price: 25.5, Category: "jewelery"})
	tok := register(t, h, "m@shop.test")

	// client พยายามส่งราคาเอง — ต้องถูกละเลย
	body := map[string]any{"shipping_address": "1 Main St", "items": []map[string]any{{"product_id": p.ID, "qty": 2, "price": 0.01}}}
	rec := call(t, h, "POST", "/api/orders", tok, body)
	if rec.Code != 201 {
		t.Fatalf("create order: %d %s", rec.Code, rec.Body.String())
	}
	var o Order
	json.Unmarshal(rec.Body.Bytes(), &o)
	if o.Total != 51 {
		t.Fatalf("total should be computed server-side = 51, got %v", o.Total)
	}
	if rec := call(t, h, "POST", "/api/orders", tok, map[string]any{"shipping_address": "x", "items": []map[string]any{{"product_id": 9999, "qty": 1}}}); rec.Code != 400 {
		t.Fatalf("unknown product should 400, got %d", rec.Code)
	}
	if rec := call(t, h, "POST", "/api/orders", tok, map[string]any{"shipping_address": "x", "items": []map[string]any{{"product_id": p.ID, "qty": 0}}}); rec.Code != 400 {
		t.Fatalf("qty 0 should 400, got %d", rec.Code)
	}
	if rec := call(t, h, "GET", "/api/orders", tok, nil); rec.Code != 200 || !strings.Contains(rec.Body.String(), "Ring") {
		t.Fatalf("my orders: %d %s", rec.Code, rec.Body.String())
	}
}

func TestMemberOrdersAreIsolated(t *testing.T) {
	_, st, h := newTestApp(t)
	p, _ := st.CreateProduct(t.Context(), ProductInput{Title: "Lamp", Price: 5, Category: "home"})
	a := register(t, h, "a@shop.test")
	b := register(t, h, "b@shop.test")
	call(t, h, "POST", "/api/orders", a, map[string]any{"shipping_address": "x", "items": []map[string]any{{"product_id": p.ID, "qty": 1}}})

	rec := call(t, h, "GET", "/api/orders", b, nil)
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("member B must not see A's orders: %s", rec.Body.String())
	}
}

func TestAdminPermissions(t *testing.T) {
	_, _, h := newTestApp(t)
	member := register(t, h, "m@shop.test")
	admin := tokenFrom(t, call(t, h, "POST", "/api/auth/login", "", map[string]string{"email": "admin@shop.test", "password": "admin-pass-123"}))

	adminPaths := []struct{ method, path string }{
		{"GET", "/api/admin/stats"}, {"GET", "/api/admin/orders"}, {"GET", "/api/admin/users"},
		{"POST", "/api/admin/products"}, {"POST", "/api/admin/sync"},
	}
	for _, ap := range adminPaths {
		if rec := call(t, h, ap.method, ap.path, "", nil); rec.Code != 401 {
			t.Errorf("guest %s %s should be 401, got %d", ap.method, ap.path, rec.Code)
		}
		if rec := call(t, h, ap.method, ap.path, member, map[string]any{}); rec.Code != 403 {
			t.Errorf("member %s %s should be 403, got %d", ap.method, ap.path, rec.Code)
		}
	}
	if rec := call(t, h, "GET", "/api/admin/stats", admin, nil); rec.Code != 200 {
		t.Fatalf("admin stats: %d", rec.Code)
	}
}

func TestAdminProductCRUDAndOrderStatus(t *testing.T) {
	_, _, h := newTestApp(t)
	admin := tokenFrom(t, call(t, h, "POST", "/api/auth/login", "", map[string]string{"email": "admin@shop.test", "password": "admin-pass-123"}))

	if rec := call(t, h, "POST", "/api/admin/products", admin, map[string]any{"title": "", "price": 1, "category": "x"}); rec.Code != 400 {
		t.Fatalf("empty title should 400, got %d", rec.Code)
	}
	rec := call(t, h, "POST", "/api/admin/products", admin, map[string]any{"title": "Mug", "price": 7, "category": "Home", "image": "https://img.test/m.png"})
	if rec.Code != 201 {
		t.Fatalf("create product: %d %s", rec.Code, rec.Body.String())
	}
	var p Product
	json.Unmarshal(rec.Body.Bytes(), &p)
	if p.Category != "home" {
		t.Fatalf("category should be normalized to lowercase, got %q", p.Category)
	}
	if rec := call(t, h, "PUT", "/api/admin/products/"+itoa(p.ID), admin, map[string]any{"title": "Big Mug", "price": 9, "category": "home"}); rec.Code != 200 || !strings.Contains(rec.Body.String(), "Big Mug") {
		t.Fatalf("update product: %d %s", rec.Code, rec.Body.String())
	}

	member := register(t, h, "m@shop.test")
	orec := call(t, h, "POST", "/api/orders", member, map[string]any{"shipping_address": "x", "items": []map[string]any{{"product_id": p.ID, "qty": 1}}})
	var o Order
	json.Unmarshal(orec.Body.Bytes(), &o)
	if rec := call(t, h, "PATCH", "/api/admin/orders/"+itoa(o.ID), admin, map[string]string{"status": "bogus"}); rec.Code != 400 {
		t.Fatalf("bad status should 400, got %d", rec.Code)
	}
	if rec := call(t, h, "PATCH", "/api/admin/orders/"+itoa(o.ID), admin, map[string]string{"status": "shipped"}); rec.Code != 200 || !strings.Contains(rec.Body.String(), "shipped") {
		t.Fatalf("update status: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(t, h, "DELETE", "/api/admin/products/"+itoa(p.ID), admin, nil); rec.Code != 204 {
		t.Fatalf("delete product: %d", rec.Code)
	}
	if rec := call(t, h, "DELETE", "/api/admin/products/"+itoa(p.ID), admin, nil); rec.Code != 404 {
		t.Fatalf("second delete should 404, got %d", rec.Code)
	}
}

func TestSyncFromFakeStore(t *testing.T) {
	app, st, h := newTestApp(t)
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/products" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`[{"id":1,"title":"Fjallraven Backpack","price":109.95,"description":"d","category":"men's clothing","image":"https://i.test/1.jpg","rating":{"rate":3.9,"count":120}},
		{"id":2,"title":"Slim Fit Tee","price":22.3,"description":"d","category":"men's clothing","image":"https://i.test/2.jpg","rating":{"rate":4.1,"count":259}}]`))
	}))
	defer fake.Close()
	app.cfg.FakeStoreURL = fake.URL

	admin := tokenFrom(t, call(t, h, "POST", "/api/auth/login", "", map[string]string{"email": "admin@shop.test", "password": "admin-pass-123"}))
	for i := 0; i < 2; i++ { // sync ซ้ำต้องไม่เกิดสินค้าซ้ำ
		rec := call(t, h, "POST", "/api/admin/sync", admin, nil)
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"imported":2`) {
			t.Fatalf("sync: %d %s", rec.Code, rec.Body.String())
		}
	}
	if n, _ := st.CountProducts(t.Context()); n != 2 {
		t.Fatalf("expected 2 products after repeated sync, got %d", n)
	}
}

func TestCORS(t *testing.T) {
	_, _, h := newTestApp(t)
	req := httptest.NewRequest("OPTIONS", "/api/products", nil)
	req.Header.Set("Origin", "https://my-preview-abc.vercel.app")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 204 || rec.Header().Get("Access-Control-Allow-Origin") != "https://my-preview-abc.vercel.app" {
		t.Fatalf("wildcard origin should be allowed: %d %v", rec.Code, rec.Header())
	}
	req = httptest.NewRequest("GET", "/api/products", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("unknown origin must not be allowed")
	}
}

func itoa(n int64) string { return strings.TrimSpace(strings.Join([]string{formatInt(n)}, "")) }

func formatInt(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
