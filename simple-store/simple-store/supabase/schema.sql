-- Simple Store — Supabase schema
-- วิธีใช้: Supabase Dashboard → SQL Editor → วางทั้งไฟล์นี้แล้วกด Run (รันซ้ำได้ ไม่พัง)

create table if not exists users (
  id            uuid primary key default gen_random_uuid(),
  email         text not null unique,
  name          text not null,
  password_hash text not null,
  role          text not null default 'member' check (role in ('member', 'admin')),
  created_at    timestamptz not null default now()
);

create table if not exists products (
  id           bigint generated always as identity primary key,
  source_id    integer unique,            -- id จาก fakestoreapi.com (null = สินค้าที่ Admin เพิ่มเอง)
  title        text not null,
  price        numeric(12, 2) not null check (price >= 0),
  description  text not null default '',
  category     text not null,
  image        text not null default '',
  rating_rate  numeric(3, 2) not null default 0,
  rating_count integer not null default 0,
  created_at   timestamptz not null default now()
);
create index if not exists products_category_idx on products (category);

create table if not exists orders (
  id               bigint generated always as identity primary key,
  user_id          uuid not null references users (id) on delete cascade,
  total            numeric(12, 2) not null,
  status           text not null default 'pending'
                   check (status in ('pending', 'paid', 'shipped', 'cancelled')),
  shipping_address text not null default '',
  created_at       timestamptz not null default now()
);
create index if not exists orders_user_idx on orders (user_id);

create table if not exists order_items (
  id         bigint generated always as identity primary key,
  order_id   bigint not null references orders (id) on delete cascade,
  product_id bigint references products (id) on delete set null,
  title      text not null,                -- เก็บ snapshot ชื่อ/ราคา ณ เวลาสั่งซื้อ
  price      numeric(12, 2) not null,
  qty        integer not null check (qty > 0)
);
create index if not exists order_items_order_idx on order_items (order_id);

-- Backend ใช้ service_role key (ข้าม RLS) เท่านั้น
-- เปิด RLS แบบไม่มี policy = ปิดไม่ให้ anon key อ่าน/เขียนตรงๆ ได้เลย
alter table users       enable row level security;
alter table products    enable row level security;
alter table orders      enable row level security;
alter table order_items enable row level security;

-- สร้างคำสั่งซื้อแบบ transaction เดียว และคิดราคาจากตาราง products (ไม่เชื่อราคาจาก client)
create or replace function create_order(p_user_id uuid, p_address text, p_items jsonb)
returns jsonb
language plpgsql
security definer
set search_path = public
as $$
declare
  v_order_id bigint;
  v_total    numeric(12, 2);
  v_missing  integer;
begin
  select count(*) into v_missing
  from jsonb_to_recordset(p_items) as i(product_id bigint, qty integer)
  left join products p on p.id = i.product_id
  where p.id is null;

  if v_missing > 0 then
    raise exception 'product_not_found';
  end if;

  select coalesce(sum(p.price * i.qty), 0) into v_total
  from jsonb_to_recordset(p_items) as i(product_id bigint, qty integer)
  join products p on p.id = i.product_id;

  insert into orders (user_id, total, shipping_address)
  values (p_user_id, v_total, coalesce(p_address, ''))
  returning id into v_order_id;

  insert into order_items (order_id, product_id, title, price, qty)
  select v_order_id, p.id, p.title, p.price, i.qty
  from jsonb_to_recordset(p_items) as i(product_id bigint, qty integer)
  join products p on p.id = i.product_id;

  return (
    select to_jsonb(o) || jsonb_build_object(
      'order_items',
      (select coalesce(jsonb_agg(to_jsonb(oi)), '[]'::jsonb) from order_items oi where oi.order_id = o.id)
    )
    from orders o where o.id = v_order_id
  );
end;
$$;

create or replace function admin_stats()
returns jsonb
language sql
security definer
set search_path = public
as $$
  select jsonb_build_object(
    'products', (select count(*) from products),
    'users',    (select count(*) from users),
    'orders',   (select count(*) from orders),
    'revenue',  (select coalesce(sum(total), 0) from orders where status <> 'cancelled')
  );
$$;

revoke all on function create_order(uuid, text, jsonb) from public, anon, authenticated;
revoke all on function admin_stats() from public, anon, authenticated;
grant execute on function create_order(uuid, text, jsonb) to service_role;
grant execute on function admin_stats() to service_role;
