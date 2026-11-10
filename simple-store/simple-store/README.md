# Simple Store

ร้านค้าออนไลน์แบบแยก Frontend / Backend / Database ใช้ข้อมูลสินค้าตัวอย่างจาก [fakestoreapi.com](https://fakestoreapi.com)

| ส่วน | เทคโนโลยี | Deploy ที่ |
|---|---|---|
| Frontend | React + Vite + Fetch API (ไม่มี library เสริม นอกจาก React) | **Vercel** |
| Backend | Go (stdlib ล้วน — ไม่มี dependency ภายนอก) | **Render** (Docker, เชื่อม GitHub Repo) |
| Database | PostgreSQL | **Supabase** |

```
Browser ──fetch──▶ Go API (Render) ──REST/RPC (service_role key)──▶ Supabase Postgres
                        │
                        └── นำเข้าสินค้าจาก fakestoreapi.com (Admin กดปุ่ม หรือ seed ตอนเริ่มระบบ)
```

> **fakestoreapi.com ไม่ต้องใช้ API Key** — เป็น API สาธารณะ Backend เรียกอ่านตรงๆ ได้เลย
> Backend ดึงสินค้ามาเก็บในฐานข้อมูลของเราเอง เพื่อให้ Admin เพิ่ม/แก้/ลบสินค้าและสั่งซื้อจริงได้ (fakestoreapi เองบันทึกข้อมูลจริงไม่ได้)
> ถ้ามีคีย์/URL ของตัวเอง ตั้ง `FAKESTORE_URL` ได้

## สิทธิ์การใช้งาน

| | Guest | Member | Admin |
|---|:-:|:-:|:-:|
| ดูสินค้า / ค้นหา / กรองหมวดหมู่ / ใส่ตะกร้า | ✅ | ✅ | ✅ |
| สมัคร / เข้าสู่ระบบ | ✅ | – | – |
| สั่งซื้อ, ดูประวัติสั่งซื้อ, แก้โปรไฟล์, เปลี่ยนรหัสผ่าน, ออกจากระบบ | ❌ | ✅ | ✅ |
| จัดการสินค้า (เพิ่ม/แก้/ลบ), นำเข้าสินค้า, เปลี่ยนสถานะคำสั่งซื้อ, ดูสมาชิก, สถิติ | ❌ | ❌ | ✅ |

สิทธิ์ถูกตรวจ **ที่ Backend ทุก request** (JWT + role) — ฝั่ง React ซ่อนเมนูเพื่อ UX เท่านั้น

## API

| Method | Path | สิทธิ์ |
|---|---|---|
| GET | `/health` | Guest |
| GET | `/api/products?category=&q=&limit=&offset=` · `/api/products/{id}` · `/api/categories` | Guest |
| POST | `/api/auth/register` · `/api/auth/login` | Guest |
| GET / PUT | `/api/me` (ดู / แก้ชื่อ) | Member |
| POST | `/api/me/password` (เปลี่ยนรหัสผ่าน) | Member |
| POST / GET | `/api/orders` (สั่งซื้อ / ประวัติของตัวเอง) | Member |
| GET | `/api/admin/stats` · `/api/admin/orders` · `/api/admin/users` | Admin |
| POST / PUT / DELETE | `/api/admin/products[/{id}]` | Admin |
| PATCH | `/api/admin/orders/{id}` (status: pending/paid/shipped/cancelled) | Admin |
| POST | `/api/admin/sync` (นำเข้าสินค้าจาก fakestoreapi.com) | Admin |

## รันบนเครื่อง

ต้องมี Go 1.24+, Node 20+ และโปรเจกต์ Supabase

```bash
# 1) ฐานข้อมูล: Supabase → SQL Editor → วางไฟล์ supabase/schema.sql → Run

# 2) Backend
cd backend
cp .env.example .env        # แก้ค่าให้ครบ
set -a; source .env; set +a
go run .                    # http://localhost:8080

# 3) Frontend (อีก terminal)
cd frontend
cp .env.example .env
npm install
npm run dev                 # http://localhost:5173
```

ตอนเริ่ม Backend จะ (1) สร้างบัญชี Admin จาก `ADMIN_EMAIL`/`ADMIN_PASSWORD` ถ้ายังไม่มี และ (2) ดึงสินค้าจาก fakestoreapi.com ถ้าตาราง `products` ยังว่าง

## Deploy

ทำตามลำดับนี้ (Backend ต้องรู้ URL ของ Frontend และกลับกัน จึงวน 1 รอบ)

### 1) Supabase (Database)
1. สร้างโปรเจกต์ที่ supabase.com
2. **SQL Editor** → วางทั้งไฟล์ `supabase/schema.sql` → **Run** (สร้างตาราง, เปิด RLS, ฟังก์ชัน `create_order` / `admin_stats`)
3. **Project Settings → API** จดค่า `Project URL` และ `service_role` key (หรือ Secret key) — **เก็บเป็นความลับ ใช้เฉพาะใน Backend**

### 2) Render (Backend)
1. push โค้ดทั้งโฟลเดอร์ขึ้น GitHub
2. Render → **New → Blueprint** → เลือก Repo (อ่าน `render.yaml` ให้อัตโนมัติ) หรือ New Web Service → Docker, Dockerfile path `./backend/Dockerfile`, context `./backend`
3. ตั้ง Environment Variables: `SUPABASE_URL`, `SUPABASE_SERVICE_KEY`, `ADMIN_EMAIL`, `ADMIN_PASSWORD`, `ALLOWED_ORIGINS` (ใส่หลังได้ URL Vercel), `JWT_SECRET` (Render สุ่มให้)
4. Deploy เสร็จเปิด `https://<ชื่อ>.onrender.com/health` ต้องได้ `{"status":"ok"}` — Render ทำ auto-deploy ทุกครั้งที่ push

### 3) Vercel (Frontend)
1. Vercel → **Add New Project** → Import Repo เดียวกัน
2. **Root Directory = `frontend`** (Framework: Vite จะถูกตรวจจับอัตโนมัติ)
3. Environment Variable: `VITE_API_URL=https://<ชื่อ>.onrender.com`
4. Deploy — `vercel.json` ทำ SPA rewrite ให้ (refresh หน้า `/cart` ฯลฯ ได้)

### 4) กลับไปที่ Render
ตั้ง `ALLOWED_ORIGINS=https://<โปรเจกต์>.vercel.app,https://*.vercel.app` (CORS; `*.vercel.app` ครอบคลุม preview deploy) แล้ว redeploy

> Render แพ็กเกจ Free จะ "หลับ" เมื่อไม่มีคนใช้ — request แรกอาจช้า ~30–60 วินาที (หน้าเว็บจะแสดงสถานะกำลังโหลด)

## ทดสอบ

```bash
cd backend  && go test ./...      # 10 เทสต์: auth, สิทธิ์ Guest/Member/Admin, สั่งซื้อ, โปรไฟล์/รหัสผ่าน, sync, CORS
cd frontend && npm install && npm test   # vitest + React Testing Library
```

## หมายเหตุด้านความปลอดภัย / ข้อจำกัด

- รหัสผ่านเก็บแบบ PBKDF2-SHA256 (600,000 รอบ + salt) ; JWT HS256 อายุ 12 ชั่วโมง
- ราคาในคำสั่งซื้อคำนวณที่ฐานข้อมูล (`create_order`) ไม่เชื่อราคาจาก client
- ตารางทั้งหมดเปิด RLS และไม่มี policy → anon key เข้าตรงๆ ไม่ได้ ต้องผ่าน Backend เท่านั้น
- Token เก็บใน `localStorage` (ง่ายสำหรับ prototype) — ถ้าจะขึ้น production จริงควรพิจารณา httpOnly cookie
- ยังไม่มี rate limit ที่ `/api/auth/*`, ไม่มีระบบชำระเงินจริง (สถานะ "ชำระแล้ว" Admin เป็นคนเปลี่ยน), ไม่มี reset password ทางอีเมล
- Backend ใช้ Go stdlib ล้วนและคุยกับ Supabase ผ่าน REST (PostgREST) — ถ้าต้องการ SQL ตรงให้เปลี่ยน `SupabaseStore` เป็นตัวที่ใช้ `pgx` ได้ เพราะทุกอย่างอยู่หลัง interface `Store`
