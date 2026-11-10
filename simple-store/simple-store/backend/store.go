package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Store คือชั้นข้อมูลทั้งหมดของ Backend — ผูกกับ Supabase ผ่าน SupabaseStore
// และมี in-memory เวอร์ชันใน test
type Store interface {
	ListProducts(ctx context.Context, category, q string, limit, offset int) ([]Product, error)
	GetProduct(ctx context.Context, id int64) (*Product, error)
	Categories(ctx context.Context) ([]string, error)
	CountProducts(ctx context.Context) (int, error)
	CreateProduct(ctx context.Context, in ProductInput) (*Product, error)
	UpdateProduct(ctx context.Context, id int64, in ProductInput) (*Product, error)
	DeleteProduct(ctx context.Context, id int64) error
	UpsertSourceProducts(ctx context.Context, ps []Product) error

	CreateUser(ctx context.Context, u User) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	ListUsers(ctx context.Context) ([]User, error)
	UpdateUserName(ctx context.Context, id, name string) (*User, error)
	UpdateUserPassword(ctx context.Context, id, passwordHash string) error

	CreateOrder(ctx context.Context, userID, address string, lines []CartLine) (*Order, error)
	ListOrdersByUser(ctx context.Context, userID string) ([]Order, error)
	ListOrders(ctx context.Context) ([]Order, error)
	UpdateOrderStatus(ctx context.Context, id int64, status string) (*Order, error)

	Stats(ctx context.Context) (*Stats, error)
}

// ---------- Supabase REST (PostgREST) client ----------

type SupabaseStore struct {
	base string
	key  string
	hc   *http.Client
}

type SupaError struct {
	Status int
	Body   string
}

func (e *SupaError) Error() string { return fmt.Sprintf("supabase %d: %s", e.Status, e.Body) }

func NewSupabaseStore(baseURL, serviceKey string) *SupabaseStore {
	return &SupabaseStore{base: baseURL, key: serviceKey, hc: &http.Client{Timeout: 20 * time.Second}}
}

func (s *SupabaseStore) do(ctx context.Context, method, path string, q url.Values, body any, prefer string, out any) error {
	u := s.base + "/rest/v1/" + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("apikey", s.key)
	// คีย์แบบ JWT (service_role) ส่งเป็น Bearer ได้; คีย์รูปแบบใหม่ (sb_secret_...) ใช้ header apikey อย่างเดียว
	if strings.HasPrefix(s.key, "eyJ") {
		req.Header.Set("Authorization", "Bearer "+s.key)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if prefer != "" {
		req.Header.Set("Prefer", prefer)
	}
	resp, err := s.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if resp.StatusCode >= 300 {
		return &SupaError{Status: resp.StatusCode, Body: string(data)}
	}
	if out != nil && len(bytes.TrimSpace(data)) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

func eq(v string) string { return "eq." + v }

// ตัดอักขระที่มีความหมายใน PostgREST filter ออกจากคำค้นหา
func sanitizeSearch(q string) string {
	r := strings.NewReplacer(",", " ", "(", " ", ")", " ", "*", " ", "\\", " ", "\"", " ", ".", " ")
	return strings.TrimSpace(r.Replace(q))
}

// ---------- Products ----------

func (s *SupabaseStore) ListProducts(ctx context.Context, category, q string, limit, offset int) ([]Product, error) {
	v := url.Values{"select": {"*"}, "order": {"id.asc"}, "limit": {strconv.Itoa(limit)}, "offset": {strconv.Itoa(offset)}}
	if category != "" {
		v.Set("category", eq(category))
	}
	if q = sanitizeSearch(q); q != "" {
		v.Set("or", fmt.Sprintf("(title.ilike.*%s*,description.ilike.*%s*)", q, q))
	}
	out := []Product{}
	return out, s.do(ctx, "GET", "products", v, nil, "", &out)
}

func (s *SupabaseStore) GetProduct(ctx context.Context, id int64) (*Product, error) {
	var out []Product
	err := s.do(ctx, "GET", "products", url.Values{"select": {"*"}, "id": {eq(strconv.FormatInt(id, 10))}}, nil, "", &out)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return &out[0], nil
}

func (s *SupabaseStore) Categories(ctx context.Context) ([]string, error) {
	var rows []struct {
		Category string `json:"category"`
	}
	if err := s.do(ctx, "GET", "products", url.Values{"select": {"category"}, "limit": {"1000"}}, nil, "", &rows); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	out := []string{}
	for _, r := range rows {
		if !seen[r.Category] {
			seen[r.Category] = true
			out = append(out, r.Category)
		}
	}
	sortStrings(out)
	return out, nil
}

func (s *SupabaseStore) CountProducts(ctx context.Context) (int, error) {
	st, err := s.Stats(ctx)
	if err != nil {
		return 0, err
	}
	return st.Products, nil
}

func (s *SupabaseStore) CreateProduct(ctx context.Context, in ProductInput) (*Product, error) {
	var out []Product
	if err := s.do(ctx, "POST", "products", nil, in, "return=representation", &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty response creating product")
	}
	return &out[0], nil
}

func (s *SupabaseStore) UpdateProduct(ctx context.Context, id int64, in ProductInput) (*Product, error) {
	var out []Product
	err := s.do(ctx, "PATCH", "products", url.Values{"id": {eq(strconv.FormatInt(id, 10))}}, in, "return=representation", &out)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return &out[0], nil
}

func (s *SupabaseStore) DeleteProduct(ctx context.Context, id int64) error {
	var out []Product
	err := s.do(ctx, "DELETE", "products", url.Values{"id": {eq(strconv.FormatInt(id, 10))}}, nil, "return=representation", &out)
	if err != nil {
		return err
	}
	if len(out) == 0 {
		return ErrNotFound
	}
	return nil
}

// UpsertSourceProducts อัปเดตสินค้าที่มี source_id ตรงกัน หรือเพิ่มใหม่ถ้ายังไม่มี
func (s *SupabaseStore) UpsertSourceProducts(ctx context.Context, ps []Product) error {
	if len(ps) == 0 {
		return nil
	}
	return s.do(ctx, "POST", "products", url.Values{"on_conflict": {"source_id"}}, ps, "resolution=merge-duplicates,return=minimal", nil)
}

// ---------- Users ----------

func (s *SupabaseStore) CreateUser(ctx context.Context, u User) (*User, error) {
	var out []User
	err := s.do(ctx, "POST", "users", nil, u, "return=representation", &out)
	if err != nil {
		if se, ok := err.(*SupaError); ok && se.Status == http.StatusConflict {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty response creating user")
	}
	return &out[0], nil
}

func (s *SupabaseStore) getUser(ctx context.Context, col, val string) (*User, error) {
	var out []User
	if err := s.do(ctx, "GET", "users", url.Values{"select": {"*"}, col: {eq(val)}}, nil, "", &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return &out[0], nil
}

func (s *SupabaseStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.getUser(ctx, "email", email)
}

func (s *SupabaseStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	return s.getUser(ctx, "id", id)
}

func (s *SupabaseStore) ListUsers(ctx context.Context) ([]User, error) {
	out := []User{}
	return out, s.do(ctx, "GET", "users", url.Values{"select": {"id,email,name,role,created_at"}, "order": {"created_at.desc"}, "limit": {"500"}}, nil, "", &out)
}

func (s *SupabaseStore) UpdateUserName(ctx context.Context, id, name string) (*User, error) {
	var out []User
	err := s.do(ctx, "PATCH", "users", url.Values{"id": {eq(id)}}, map[string]string{"name": name}, "return=representation", &out)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return &out[0], nil
}

func (s *SupabaseStore) UpdateUserPassword(ctx context.Context, id, passwordHash string) error {
	var out []User
	err := s.do(ctx, "PATCH", "users", url.Values{"id": {eq(id)}}, map[string]string{"password_hash": passwordHash}, "return=representation", &out)
	if err != nil {
		return err
	}
	if len(out) == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- Orders ----------

func (s *SupabaseStore) CreateOrder(ctx context.Context, userID, address string, lines []CartLine) (*Order, error) {
	var out Order
	err := s.do(ctx, "POST", "rpc/create_order", nil,
		map[string]any{"p_user_id": userID, "p_address": address, "p_items": lines}, "", &out)
	if err != nil {
		if se, ok := err.(*SupaError); ok && strings.Contains(se.Body, "product_not_found") {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (s *SupabaseStore) ListOrdersByUser(ctx context.Context, userID string) ([]Order, error) {
	out := []Order{}
	return out, s.do(ctx, "GET", "orders", url.Values{
		"select": {"*,order_items(*)"}, "user_id": {eq(userID)}, "order": {"created_at.desc"}, "limit": {"200"},
	}, nil, "", &out)
}

func (s *SupabaseStore) ListOrders(ctx context.Context) ([]Order, error) {
	out := []Order{}
	return out, s.do(ctx, "GET", "orders", url.Values{
		"select": {"*,order_items(*),users(email,name)"}, "order": {"created_at.desc"}, "limit": {"500"},
	}, nil, "", &out)
}

func (s *SupabaseStore) UpdateOrderStatus(ctx context.Context, id int64, status string) (*Order, error) {
	var out []Order
	err := s.do(ctx, "PATCH", "orders", url.Values{"id": {eq(strconv.FormatInt(id, 10))}, "select": {"*,order_items(*)"}},
		map[string]string{"status": status}, "return=representation", &out)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return &out[0], nil
}

func (s *SupabaseStore) Stats(ctx context.Context) (*Stats, error) {
	var out Stats
	if err := s.do(ctx, "POST", "rpc/admin_stats", nil, map[string]any{}, "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
