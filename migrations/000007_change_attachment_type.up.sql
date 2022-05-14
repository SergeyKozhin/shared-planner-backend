alter table events alter column attachments type jsonb using '[]'::jsonb;
