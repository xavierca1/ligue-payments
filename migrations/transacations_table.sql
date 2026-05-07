create table public.subscriptions (
  id uuid not null default extensions.uuid_generate_v4 (),
  customer_id uuid not null,
  product_id uuid not null,
  amount integer not null,
  status public.sub_status null default 'PENDING'::sub_status,
  interval public.sub_interval null default 'MONTHLY'::sub_interval,
  next_billing_date date not null,
  payment_method_id character varying(100) null,
  created_at timestamp with time zone null default now(),
  updated_at timestamp with time zone null default now(),
  plan_id uuid null,
  payment_method character varying(20) null,
  constraint subscriptions_pkey primary key (id),
  constraint subscriptions_customer_id_fkey foreign KEY (customer_id) references customers (id),
  constraint subscriptions_plan_id_fkey foreign KEY (plan_id) references plans (id),
  constraint subscriptions_product_id_fkey foreign KEY (product_id) references products (id)
) TABLESPACE pg_default;

create index IF not exists idx_subs_next_billing on public.subscriptions using btree (status, next_billing_date) TABLESPACE pg_default;

create index IF not exists idx_subscriptions_expiration on public.subscriptions using btree (status, payment_method, created_at) TABLESPACE pg_default
where
  (
    (status = 'PENDING'::sub_status)
    and ((payment_method)::text = 'PIX'::text)
  );