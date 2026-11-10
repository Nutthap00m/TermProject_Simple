package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	// Guest — ไม่ต้อง login
	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("GET /api/products", a.listProducts)
	mux.HandleFunc("GET /api/products/{id}", a.getProduct)
	mux.HandleFunc("GET /api/categories", a.categories)
	mux.HandleFunc("POST /api/auth/register", a.register)
	mux.HandleFunc("POST /api/auth/login", a.login)

	// Member — ต้อง login
	mux.Handle("GET /api/me", a.member(a.me))
	mux.Handle("PUT /api/me", a.member(a.updateMe))
	mux.Handle("POST /api/me/password", a.member(a.changePassword))
	mux.Handle("POST /api/orders", a.member(a.createOrder))
	mux.Handle("GET /api/orders", a.member(a.myOrders))

	// Admin — ต้อง login และ role = admin (ตรวจที่ Backend ทุก request)
	mux.Handle("GET /api/admin/stats", a.admin(a.adminStats))
	mux.Handle("POST /api/admin/products", a.admin(a.adminCreateProduct))
	mux.Handle("PUT /api/admin/products/{id}", a.admin(a.adminUpdateProduct))
	mux.Handle("DELETE /api/admin/products/{id}", a.admin(a.adminDeleteProduct))
	mux.Handle("POST /api/admin/sync", a.admin(a.adminSync))
	mux.Handle("GET /api/admin/orders", a.admin(a.adminOrders))
	mux.Handle("PATCH /api/admin/orders/{id}", a.admin(a.adminUpdateOrder))
	mux.Handle("GET /api/admin/users", a.admin(a.adminUsers))

	return a.cors(mux)
}

func (a *App) originAllowed(origin string) bool {
	for _, o := range a.cfg.AllowedOrigins {
		if o == "*" || o == origin {
			return true
		}
		// รองรับ wildcard แบบ https://*.vercel.app
		if i := strings.Index(o, "*."); i >= 0 {
			if strings.HasPrefix(origin, o[:i]) && strings.HasSuffix(origin, o[i+1:]) {
				return true
			}
		}
	}
	return false
}

func (a *App) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && a.originAllowed(origin) {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Vary", "Origin")
			h.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Authorization,Content-Type")
			h.Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	app := &App{
		cfg:   cfg,
		store: NewSupabaseStore(cfg.SupabaseURL, cfg.SupabaseKey),
		hc:    &http.Client{Timeout: 20 * time.Second},
	}

	// งานตอนเริ่มระบบ (ไม่ทำให้เซิร์ฟเวอร์ล้มถ้าล้มเหลว แต่ log ไว้)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := app.seedAdmin(ctx); err != nil {
			log.Printf("seed admin: %v", err)
		}
		if err := app.seedProducts(ctx); err != nil {
			log.Printf("seed products: %v", err)
		}
	}()

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
