package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemStore คือ Store แบบ in-memory ใช้ทดสอบ handler โดยไม่ต้องต่อ Supabase
type MemStore struct {
	mu       sync.Mutex
	products []Product
	users    []User
	orders   []Order
	nextPID  int64
	nextOID  int64
}

func NewMemStore() *MemStore { return &MemStore{nextPID: 1, nextOID: 1} }

func (m *MemStore) ListProducts(_ context.Context, category, q string, limit, offset int) ([]Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []Product{}
	for _, p := range m.products {
		if category != "" && p.Category != category {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(p.Title+p.Description), strings.ToLower(q)) {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (m *MemStore) GetProduct(_ context.Context, id int64) (*Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.products {
		if p.ID == id {
			c := p
			return &c, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemStore) Categories(_ context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	seen := map[string]bool{}
	out := []string{}
	for _, p := range m.products {
		if !seen[p.Category] {
			seen[p.Category] = true
			out = append(out, p.Category)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (m *MemStore) CountProducts(_ context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.products), nil
}

func (m *MemStore) CreateProduct(_ context.Context, in ProductInput) (*Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := Product{ID: m.nextPID, Title: in.Title, Price: in.Price, Description: in.Description, Category: in.Category, Image: in.Image}
	m.nextPID++
	m.products = append(m.products, p)
	return &p, nil
}

func (m *MemStore) UpdateProduct(_ context.Context, id int64, in ProductInput) (*Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.products {
		if m.products[i].ID == id {
			p := &m.products[i]
			p.Title, p.Price, p.Description, p.Category, p.Image = in.Title, in.Price, in.Description, in.Category, in.Image
			c := *p
			return &c, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemStore) DeleteProduct(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.products {
		if m.products[i].ID == id {
			m.products = append(m.products[:i], m.products[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (m *MemStore) UpsertSourceProducts(_ context.Context, ps []Product) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, np := range ps {
		found := false
		for i := range m.products {
			if m.products[i].SourceID != nil && np.SourceID != nil && *m.products[i].SourceID == *np.SourceID {
				id := m.products[i].ID
				m.products[i] = np
				m.products[i].ID = id
				found = true
				break
			}
		}
		if !found {
			np.ID = m.nextPID
			m.nextPID++
			m.products = append(m.products, np)
		}
	}
	return nil
}

func (m *MemStore) CreateUser(_ context.Context, u User) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, x := range m.users {
		if x.Email == u.Email {
			return nil, ErrEmailTaken
		}
	}
	u.ID = fmt.Sprintf("user-%d", len(m.users)+1)
	u.CreatedAt = time.Now().Format(time.RFC3339)
	m.users = append(m.users, u)
	return &u, nil
}

func (m *MemStore) GetUserByEmail(_ context.Context, email string) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			c := u
			return &c, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemStore) GetUserByID(_ context.Context, id string) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.ID == id {
			c := u
			return &c, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemStore) ListUsers(_ context.Context) ([]User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]User{}, m.users...), nil
}

func (m *MemStore) UpdateUserName(_ context.Context, id, name string) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.users {
		if m.users[i].ID == id {
			m.users[i].Name = name
			c := m.users[i]
			return &c, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemStore) UpdateUserPassword(_ context.Context, id, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.users {
		if m.users[i].ID == id {
			m.users[i].PasswordHash = hash
			return nil
		}
	}
	return ErrNotFound
}

func (m *MemStore) CreateOrder(_ context.Context, userID, address string, lines []CartLine) (*Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o := Order{ID: m.nextOID, UserID: userID, Status: "pending", ShippingAddress: address, CreatedAt: time.Now().Format(time.RFC3339)}
	for _, l := range lines {
		var prod *Product
		for i := range m.products {
			if m.products[i].ID == l.ProductID {
				prod = &m.products[i]
			}
		}
		if prod == nil {
			return nil, ErrProductNotFound
		}
		pid := prod.ID
		o.Items = append(o.Items, OrderItem{ProductID: &pid, Title: prod.Title, Price: prod.Price, Qty: l.Qty})
		o.Total += prod.Price * float64(l.Qty)
	}
	m.nextOID++
	m.orders = append(m.orders, o)
	return &o, nil
}

func (m *MemStore) ListOrdersByUser(_ context.Context, userID string) ([]Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []Order{}
	for _, o := range m.orders {
		if o.UserID == userID {
			out = append(out, o)
		}
	}
	return out, nil
}

func (m *MemStore) ListOrders(_ context.Context) ([]Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Order{}, m.orders...), nil
}

func (m *MemStore) UpdateOrderStatus(_ context.Context, id int64, status string) (*Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.orders {
		if m.orders[i].ID == id {
			m.orders[i].Status = status
			c := m.orders[i]
			return &c, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemStore) Stats(_ context.Context) (*Stats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := &Stats{Products: len(m.products), Users: len(m.users), Orders: len(m.orders)}
	for _, o := range m.orders {
		if o.Status != "cancelled" {
			st.Revenue += o.Total
		}
	}
	return st, nil
}
