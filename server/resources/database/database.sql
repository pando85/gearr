CREATE TABLE IF NOT EXISTS videos
(
    id varchar(255)  primary key,
    source_path    text not null,
    destination_path text not null,
    duration int
);

CREATE TABLE IF NOT EXISTS video_events(
    video_id varchar(255)  not null,
    video_event_id int not null,
    worker_name varchar(255) not null,
    event_time timestamp  not null,
    event_type varchar(50) not null,
    notification_type varchar(50) not null,
    status  varchar(20) not null,
    message text,
    primary key (video_id,video_event_id),
    foreign KEY (video_id) REFERENCES videos(id)
);

CREATE TABLE IF NOT EXISTS workers
(
   name varchar(100) primary key not null,
   ip varchar(100) not null,
   queue_name varchar(255) not null,
   last_seen timestamp not null
);

CREATE TABLE IF NOT EXISTS video_status (
                                video_id          varchar(255) not null,
                                video_event_id    integer      not null,
                                video_path        text not null,
                                worker_name       varchar(255) not null,
                                event_time        timestamp    not null,
                                event_type        varchar(50)  not null,
                                notification_type varchar(50)  not null,
                                status            varchar(20)  not null,
                                message           text,
                                constraint video_status_pkey
                                    primary key (video_id)
);

--Function to insert update on video_status
create or replace function fn_video_status_update(p_video_id varchar, p_video_event_id integer,
                                                  p_worker_name varchar, p_event_time timestamp, p_event_type varchar, p_notification_type varchar, p_status varchar, p_message text) returns void
    security definer
    language plpgsql as $$
declare
    p_video_path varchar;
begin
    select v.source_path into p_video_path from videos v where v.id=p_video_id;
    insert into video_status(video_id, video_event_id, video_path,worker_name, event_time, event_type, notification_type, status, message)
    values (p_video_id, p_video_event_id,p_video_path, p_worker_name, p_event_time, p_event_type, p_notification_type,
            p_status, p_message)
    on conflict on constraint video_status_pkey
        do update set video_event_id=p_video_event_id, video_path=p_video_path,worker_name=p_worker_name,
                      event_time=p_event_time, event_type=p_event_type,
                      notification_type=p_event_time, status=p_status, message=p_message;
end;
$$;

--trigger function for video_status_update
create or replace function fn_trigger_video_status_update() returns trigger
    security definer
    language plpgsql
as $$
begin
    perform fn_video_status_update(new.video_id, new.video_event_id,
                                   new.worker_name,new.event_time,new.event_type,new.notification_type,
                                   new.status,new.message);
    return new;
end;
$$;
--trigger video_events
create trigger event_insert_video_status_update after insert on video_events
    for each row
execute procedure fn_trigger_video_status_update();


--To Reload Everything!!
--do language plpgsql $$
--    declare
--        e record;
--        i integer:=1;
--    begin
--        for e in (select * from video_events order by event_time asc) loop
--                perform fn_video_status_update(e.video_id, e.video_event_id,
--                                               e.worker_name,e.event_time,e.event_type,e.notification_type,
--                                               e.status,e.message);
--                i:=i+1;
--                IF MOD(i, 200) = 0 THEN
--                    COMMIT;
--                END IF;
--            end loop;
--    end;
--$$;


