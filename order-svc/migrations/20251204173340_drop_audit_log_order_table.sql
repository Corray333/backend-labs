-- +goose Up
-- +goose StatementBegin
drop table if exists audit_log_order;
-- +goose StatementEnd

-- +goose Down
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
-- +goose StatementEnd
