create index concurrently if not exists user_name_email_phone_trgm ON users USING GIST ((full_name || ' ' || email || ' ' || phone_number) gist_trgm_ops);
