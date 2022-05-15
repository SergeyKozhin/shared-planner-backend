begin;

alter table users add primary key (id);

create table if not exists groups
(
    id         bigserial primary key,
    name       text   not null,
    creator_id bigint not null references users (id)
);

commit;