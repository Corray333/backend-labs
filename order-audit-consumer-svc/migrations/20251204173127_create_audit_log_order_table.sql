-- +goose Up
-- +goose StatementBegin
create table if not exists audit_log_order
(
    id            bigserial                not null primary key,
    order_id      bigint                   not null,
    order_item_id bigint                   not null,
    customer_id   bigint                   not null,
    order_status  text                     not null,
    created_at    timestamp with time zone not null,
    updated_at    timestamp with time zone not null
);

create index if not exists idx_audit_log_order_order_id on audit_log_order (order_id);
create index if not exists idx_audit_log_order_customer_id on audit_log_order (customer_id);
create index if not exists idx_audit_log_order_created_at on audit_log_order (created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists audit_log_order;
-- +goose StatementEnd
