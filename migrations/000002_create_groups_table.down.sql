begin;

drop table if exists groups;

alter table users drop constraint users_pkey;

commit;