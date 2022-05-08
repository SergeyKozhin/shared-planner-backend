create table if not exists users(
    id bigserial,
    full_name text not null,
    email text not null unique,
    phone_number text not null,
    photo text not null
);
