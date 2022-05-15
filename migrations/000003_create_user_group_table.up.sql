create table if not exists user_group (
    id bigserial primary key,
    user_id bigint not null references users(id),
    group_id bigint not null references groups(id),
    color text not null,
    notify bool not null
)