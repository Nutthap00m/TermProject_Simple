package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"sort"
	"strconv"
	"strings"
	"time"
)

type App struct {
	cfg   Config
	store Store
	hc    *http.Client
}

func sortStrings(s []string) { sort.Strings(s) }

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func serverErr(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeErr(w, http.StatusInternalServerError, "internal server error")
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}

// ---------- Guest (public) ----------

func (a *App) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) listProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 100
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}
	ps, err := a.store.ListProducts(r.Context(), q.Get("category"), q.Get("q"), limit, offset)
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ps)
}

func (a *App) getProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid product id")
		return
	}
	p, err := a.store.GetProduct(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusNotFound, "product not found")
		return
	} else if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (a *App) categories(w http.ResponseWriter, r *http.Request) {
	cs, err := a.store.Categories(r.Context())
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

// ---------- Auth ----------

type authResponse struct {
	Token string   `json:"token"`
	User  UserView `json:"user"`
}

func (a *App) issue(w http.ResponseWriter, status int, u *User) {
	tok, err := SignJWT([]byte(a.cfg.JWTSecret), Claims{Sub: u.ID, Email: u.Email, Role: u.Role})
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, status, authResponse{Token: tok, User: u.View()})
}

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Name, Password string }
	if !decode(w, r, &in) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	name := strings.TrimSpace(in.Name)
	if _, err := mail.ParseAddress(email); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid email")
		return
	}
	if name == "" || len(name) > 100 {
		writeErr(w, http.StatusBadRequest, "name is required (max 100 chars)")
		return
	}
	if len(in.Password) < 8 || len(in.Password) > 128 {
		writeErr(w, http.StatusBadRequest, "password must be 8-128 characters")
		return
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		serverErr(w, err)
		return
	}
	u, err := a.store.CreateUser(r.Context(), User{Email: email, Name: name, PasswordHash: hash, Role: "member"})
	if errors.Is(err, ErrEmailTaken) {
		writeErr(w, http.StatusConflict, "email already registered")
		return
	} else if err != nil {
		serverErr(w, err)
		return
	}
	a.issue(w, http.StatusCreated, u)
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if !decode(w, r, &in) {
		return
	}
	u, err := a.store.GetUserByEmail(r.Context(), strings.ToLower(strings.TrimSpace(in.Email)))
	if errors.Is(err, ErrNotFound) || (err == nil && !CheckPassword(in.Password, u.PasswordHash)) {
		writeErr(w, http.StatusUnauthorized, "invalid email or password")
		return
	} else if err != nil {
		serverErr(w, err)
		return
	}
	a.issue(w, http.StatusOK, u)
}

// ---------- Member ----------

func (a *App) me(w http.ResponseWriter, r *http.Request) {
	u, err := a.store.GetUserByID(r.Context(), claimsFrom(r.Context()).Sub)
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusUnauthorized, "account no longer exists")
		return
	} else if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u.View())
}

func (a *App) updateMe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 100 {
		writeErr(w, http.StatusBadRequest, "name is required (max 100 chars)")
		return
	}
	u, err := a.store.UpdateUserName(r.Context(), claimsFrom(r.Context()).Sub, name)
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusUnauthorized, "account no longer exists")
		return
	} else if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u.View())
}

func (a *App) changePassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.NewPassword) < 8 || len(in.NewPassword) > 128 {
		writeErr(w, http.StatusBadRequest, "new password must be 8-128 characters")
		return
	}
	id := claimsFrom(r.Context()).Sub
	u, err := a.store.GetUserByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusUnauthorized, "account no longer exists")
		return
	} else if err != nil {
		serverErr(w, err)
		return
	}
	if !CheckPassword(in.CurrentPassword, u.PasswordHash) {
		writeErr(w, http.StatusBadRequest, "current password is incorrect")
		return
	}
	hash, err := HashPassword(in.NewPassword)
	if err != nil {
		serverErr(w, err)
		return
	}
	if err := a.store.UpdateUserPassword(r.Context(), id, hash); err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "password updated"})
}

func (a *App) createOrder(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Items           []CartLine `json:"items"`
		ShippingAddress string     `json:"shipping_address"`
	}
	if !decode(w, r, &in) {
		return
	}
	addr := strings.TrimSpace(in.ShippingAddress)
	if addr == "" || len(addr) > 500 {
		writeErr(w, http.StatusBadRequest, "shipping_address is required (max 500 chars)")
		return
	}
	if len(in.Items) == 0 || len(in.Items) > 50 {
		writeErr(w, http.StatusBadRequest, "items must contain 1-50 lines")
		return
	}
	// รวมรายการซ้ำ และตรวจจำนวน
	merged := map[int64]int{}
	order := []int64{}
	for _, l := range in.Items {
		if l.ProductID < 1 || l.Qty < 1 || l.Qty > 99 {
			writeErr(w, http.StatusBadRequest, "each item needs product_id >= 1 and qty 1-99")
			return
		}
		if _, seen := merged[l.ProductID]; !seen {
			order = append(order, l.ProductID)
		}
		merged[l.ProductID] += l.Qty
	}
	lines := make([]CartLine, 0, len(order))
	for _, id := range order {
		if merged[id] > 99 {
			writeErr(w, http.StatusBadRequest, "qty per product must be at most 99")
			return
		}
		lines = append(lines, CartLine{ProductID: id, Qty: merged[id]})
	}
	o, err := a.store.CreateOrder(r.Context(), claimsFrom(r.Context()).Sub, addr, lines)
	if errors.Is(err, ErrProductNotFound) {
		writeErr(w, http.StatusBadRequest, "one or more products no longer exist")
		return
	} else if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

func (a *App) myOrders(w http.ResponseWriter, r *http.Request) {
	os, err := a.store.ListOrdersByUser(r.Context(), claimsFrom(r.Context()).Sub)
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, os)
}

// ---------- Admin ----------

func (a *App) adminStats(w http.ResponseWriter, r *http.Request) {
	st, err := a.store.Stats(r.Context())
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func validateProduct(in *ProductInput) string {
	in.Title = strings.TrimSpace(in.Title)
	in.Category = strings.TrimSpace(strings.ToLower(in.Category))
	in.Image = strings.TrimSpace(in.Image)
	switch {
	case in.Title == "" || len(in.Title) > 200:
		return "title is required (max 200 chars)"
	case in.Category == "" || len(in.Category) > 60:
		return "category is required (max 60 chars)"
	case in.Price < 0 || in.Price > 10_000_000:
		return "price must be between 0 and 10,000,000"
	case len(in.Description) > 5000:
		return "description too long (max 5000 chars)"
	case in.Image != "" && !strings.HasPrefix(in.Image, "https://") && !strings.HasPrefix(in.Image, "http://"):
		return "image must be an http(s) URL"
	}
	return ""
}

func (a *App) adminCreateProduct(w http.ResponseWriter, r *http.Request) {
	var in ProductInput
	if !decode(w, r, &in) {
		return
	}
	if msg := validateProduct(&in); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	p, err := a.store.CreateProduct(r.Context(), in)
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (a *App) adminUpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid product id")
		return
	}
	var in ProductInput
	if !decode(w, r, &in) {
		return
	}
	if msg := validateProduct(&in); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	p, err := a.store.UpdateProduct(r.Context(), id, in)
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusNotFound, "product not found")
		return
	} else if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (a *App) adminDeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid product id")
		return
	}
	if err := a.store.DeleteProduct(r.Context(), id); errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusNotFound, "product not found")
		return
	} else if err != nil {
		serverErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) adminOrders(w http.ResponseWriter, r *http.Request) {
	os, err := a.store.ListOrders(r.Context())
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, os)
}

func (a *App) adminUpdateOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid order id")
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !validStatuses[in.Status] {
		writeErr(w, http.StatusBadRequest, "status must be pending, paid, shipped or cancelled")
		return
	}
	o, err := a.store.UpdateOrderStatus(r.Context(), id, in.Status)
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusNotFound, "order not found")
		return
	} else if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (a *App) adminUsers(w http.ResponseWriter, r *http.Request) {
	us, err := a.store.ListUsers(r.Context())
	if err != nil {
		serverErr(w, err)
		return
	}
	views := make([]UserView, len(us))
	for i, u := range us {
		views[i] = u.View()
	}
	writeJSON(w, http.StatusOK, views)
}

// ---------- Import จาก fakestoreapi.com ----------

type fakeProduct struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Image       string  `json:"image"`
	Rating      struct {
		Rate  float64 `json:"rate"`
		Count int     `json:"count"`
	} `json:"rating"`
}

// SyncFromFakeStore ดึงสินค้าจาก fakestoreapi.com (ไม่ต้องใช้ API Key) แล้ว upsert ลง Supabase
func (a *App) SyncFromFakeStore(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.FakeStoreURL+"/products", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := a.hc.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("fakestoreapi returned %d", resp.StatusCode)
	}
	var fps []fakeProduct
	if err := json.NewDecoder(io.LimitReader(resp.Body, 5<<20)).Decode(&fps); err != nil {
		return 0, fmt.Errorf("decode fakestoreapi response: %w", err)
	}
	ps := make([]Product, 0, len(fps))
	for _, f := range fps {
		id := f.ID
		ps = append(ps, Product{
			SourceID: &id, Title: f.Title, Price: f.Price, Description: f.Description,
			Category: strings.ToLower(f.Category), Image: f.Image,
			RatingRate: f.Rating.Rate, RatingCount: f.Rating.Count,
		})
	}
	if err := a.store.UpsertSourceProducts(ctx, ps); err != nil {
		return 0, err
	}
	return len(ps), nil
}

func (a *App) adminSync(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	n, err := a.SyncFromFakeStore(ctx)
	if err != nil {
		log.Printf("sync failed: %v", err)
		writeErr(w, http.StatusBadGateway, "could not import products from fakestoreapi.com")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"imported": n})
}

// ---------- Startup tasks ----------

func (a *App) seedAdmin(ctx context.Context) error {
	if a.cfg.AdminEmail == "" || a.cfg.AdminPassword == "" {
		return nil
	}
	if _, err := a.store.GetUserByEmail(ctx, a.cfg.AdminEmail); err == nil {
		return nil
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	hash, err := HashPassword(a.cfg.AdminPassword)
	if err != nil {
		return err
	}
	_, err = a.store.CreateUser(ctx, User{Email: a.cfg.AdminEmail, Name: "Admin", PasswordHash: hash, Role: "admin"})
	if err == nil {
		log.Printf("created admin account %s", a.cfg.AdminEmail)
	}
	return err
}

func (a *App) seedProducts(ctx context.Context) error {
	if !a.cfg.SeedOnStart {
		return nil
	}
	n, err := a.store.CountProducts(ctx)
	if err != nil || n > 0 {
		return err
	}
	cnt, err := a.SyncFromFakeStore(ctx)
	if err == nil {
		log.Printf("seeded %d products from fakestoreapi.com", cnt)
	}
	return err
}
