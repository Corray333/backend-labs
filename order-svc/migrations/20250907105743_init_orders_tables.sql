-- +goose Up
-- +goose StatementBegin
create table if not exists orders (
    id bigserial not null primary key,
    customer_id bigint not null,
    delivery_address text not null,
    total_price_cents bigint not null,
    total_price_currency text not null,
    created_at timestamp with time zone not null,
    updated_at timestamp with time zone not null
);

create index if not exists idx_order_customer_id on orders (customer_id);

create table if not exists order_items (
    id bigserial not null primary key,
    order_id bigint not null,
    product_id bigint not null,
    quantity integer not null,
    product_title text not null,
    product_url text not null,
    price_cents bigint not null,
    price_currency text not null,
    created_at timestamp with time zone not null,
    updated_at timestamp with time zone not null
);

create index if not exists idx_order_item_order_id on order_items(order_id);

create type v1_order as (
    id bigint,
    customer_id bigint,
    delivery_address text,
    total_price_cents bigint,
    total_price_currency text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

create type v1_order_item as (
    id bigint,
    order_id bigint,
    product_id bigint,
    quantity integer,
    product_title text,
    product_url text,
    price_cents bigint,
    price_currency text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop type if exists v1_order_item;
drop type if exists v1_order;
drop table if exists order_items;
drop table if exists orders;
-- +goose StatementEnd
