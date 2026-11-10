package main

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrEmailTaken      = errors.New("email already registered")
	ErrProductNotFound = errors.New("product not found")
)

type Product struct {
	ID          int64   `json:"id,omitempty"`
	SourceID    *int    `json:"source_id,omitempty"`
	Title       string  `json:"title"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Image       string  `json:"image"`
	RatingRate  float64 `json:"rating_rate"`
	RatingCount int     `json:"rating_count"`
}

// ProductInput คือข้อมูลที่ Admin ส่งมาตอนสร้าง/แก้ไขสินค้า
type ProductInput struct {
	Title       string  `json:"title"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Image       string  `json:"image"`
}

type User struct {
	ID           string `json:"id,omitempty"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	PasswordHash string `json:"password_hash,omitempty"`
	Role         string `json:"role,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// UserView คือข้อมูลผู้ใช้ที่ส่งกลับให้ client (ไม่มี password_hash)
type UserView struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at,omitempty"`
}

func (u User) View() UserView {
	return UserView{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role, CreatedAt: u.CreatedAt}
}

type OrderItem struct {
	ID        int64   `json:"id,omitempty"`
	OrderID   int64   `json:"order_id,omitempty"`
	ProductID *int64  `json:"product_id"`
	Title     string  `json:"title"`
	Price     float64 `json:"price"`
	Qty       int     `json:"qty"`
}

type OrderUser struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type Order struct {
	ID              int64       `json:"id"`
	UserID          string      `json:"user_id"`
	Total           float64     `json:"total"`
	Status          string      `json:"status"`
	ShippingAddress string      `json:"shipping_address"`
	CreatedAt       string      `json:"created_at"`
	Items           []OrderItem `json:"order_items"`
	User            *OrderUser  `json:"users,omitempty"`
}

type CartLine struct {
	ProductID int64 `json:"product_id"`
	Qty       int   `json:"qty"`
}

type Stats struct {
	Products int     `json:"products"`
	Users    int     `json:"users"`
	Orders   int     `json:"orders"`
	Revenue  float64 `json:"revenue"`
}

var validStatuses = map[string]bool{"pending": true, "paid": true, "shipped": true, "cancelled": true}
