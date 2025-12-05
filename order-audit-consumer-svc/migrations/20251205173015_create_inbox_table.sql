-- +goose Up
-- +goose StatementBegin
create table if not exists inbox (
    id bigserial not null primary key,
    message_id text not null unique,
    queue_name text not null,
    routing_key text not null,
    payload jsonb not null,
    content_type text not null default 'application/json',
    retry_count integer not null default 0,
    max_retries integer not null default 5,
    last_error text,
    created_at timestamp with time zone not null,
    updated_at timestamp with time zone not null,
    next_retry_at timestamp with time zone not null,
    delivery_tag bigint not null
);

create index if not exists idx_inbox_next_retry on inbox (next_retry_at) where retry_count < max_retries;
create index if not exists idx_inbox_created_at on inbox (created_at);
create index if not exists idx_inbox_message_id on inbox (message_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists inbox;
-- +goose StatementEnd
