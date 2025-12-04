-- +goose Up
-- +goose StatementBegin
create table if not exists outbox (
    id bigserial not null primary key,
    queue_name text not null,
    exchange_name text not null default '',
    routing_key text not null,
    payload jsonb not null,
    content_type text not null default 'application/json',
    retry_count integer not null default 0,
    max_retries integer not null default 5,
    last_error text,
    created_at timestamp with time zone not null,
    updated_at timestamp with time zone not null,
    next_retry_at timestamp with time zone not null
);

create index if not exists idx_outbox_next_retry on outbox (next_retry_at) where retry_count < max_retries;
create index if not exists idx_outbox_created_at on outbox (created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists outbox;
-- +goose StatementEnd
