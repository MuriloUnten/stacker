-- +goose Up
create table if not exists item (
    item_id      integer primary key,
    name         text not null,
    sku          text unique not null,
    kind         text not null check (kind in ('a', 'c')), -- (assembly | component)
    description  text,
    thumbnail_id integer references image(image_id),

    current_version_id integer references item_version(version_id)  -- NULL for components, always
);

create table if not exists inventory (
    item_id   integer primary key references item(item_id),
    uom       text not null check (uom in ('each', 'meter', 'centimeter', 'millimiter', 'gram', 'kilogram', 'meter_sqr')),
    available decimal(18, 6) not null check (available >= 0),
    reserved  decimal(18, 6) not null check (reserved >= 0)
);

-- exclusively assemblies get item_versions (never components)
create table if not exists item_version (
    version_id   integer primary key,
    item_id      integer not null references item(item_id),
    version_code text,
    notes        text,
    status       text not null default 'draft' check (status in ('draft', 'published', 'deprecated')),
    created_at   text not null default (datetime('now')),
    published_at text,

    unique(item_id, version_code)
);

create table if not exists bom_line (
    bom_line_id       integer primary key,
    parent_version_id integer not null references item_version(version_id),
    child_item_id     integer not null references item(item_id),
    child_version_id  integer references item_version(version_id), -- NULL for components, always
    quantity          decimal(18, 6) not null check (quantity > 0),
    position          integer not null check (position >= 0)
);

create table if not exists manufacture_order (
    order_id           integer primary key,
    item_id            integer not null references item(item_id),
    item_version_id    integer not null references item_version(version_id),
    quantity_requested decimal(18, 6) not null check (quantity_requested > 0),
    quantity_completed decimal(18, 6) not null check (quantity_completed >= 0),
    status             text not null check (status in ('pending', 'started', 'done', 'cancelled')),
    created_at         text not null default (datetime('now')),
    started_at         text,
    completed_at       text
);

create table if not exists stock_transaction (
    transaction_id integer primary key,
    item_id        integer references item(item_id),
    order_id       integer references manufacture_order(order_id),
    quantity       decimal(18, 6) not null,
    kind           text not null check (kind in ('consumption', 'production', 'purchase', 'refurbish', 'sold')),
    created_at     text not null default (datetime('now'))
);

create table if not exists tracker (
    tracker_id integer primary key,
    item_id    integer not null references item(item_id),
    threshold  decimal(18, 6) not null check (threshold >= 0)
);

create table if not exists image (
    image_id integer primary key,
    content  blob not null
);

create table if not exists item_reference_image (
    item_reference_image_id integer primary key,
    image_id integer not null references image(image_id),
    item_id  integer not null references item(item_id),
    position integer not null check (position >= 0)
);

-- the following triggers ensure the database proper behavior of components and assemblies

-- +goose StatementBegin
create trigger if not exists item_version_kind_guard_insert
before insert on item_version
for each row
when (select kind from item where item_id = new.item_id) != 'a'
begin
    select raise(abort, 'only assemblies may have item_version rows');
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger if not exists item_version_kind_guard_update
before update on item_version
for each row
when (select kind from item where item_id = new.item_id) != 'a'
begin
    select raise(abort, 'only assemblies may have item_version rows');
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger if not exists bom_line_set_child_kind_insert
before insert on bom_line
for each row
begin
    select raise(abort, 'child_item_id does not exist')
    where not exists (select 1 from item where item_id = new.child_item_id);
end;
-- +goose StatementEnd

-- +goose Down
SELECT 1;
