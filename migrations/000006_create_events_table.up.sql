create table if not exists events
(
    id bigserial primary key,
    type int not null,
    title text not null,
    description text not null,
    attachments text[],
    notifications bigint[],
    group_id bigint not null references groups (id),
    all_day bool not null,
    repeat_type int not null,
    start_date timestamptz not null,
    end_date timestamptz,
    duration interval,
    recurrence_rule text not null,
    exceptions timestamptz[]
);
