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
    primary key (video_id,video_event_id)
);

CREATE TABLE IF NOT EXISTS workers
(
   name varchar(100) primary key not null,
   ip varchar(100) not null,
   queue_name varchar(255) not null,
   last_seen timestamp not null
);


