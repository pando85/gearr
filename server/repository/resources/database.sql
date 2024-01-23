-- Define videos table
CREATE TABLE IF NOT EXISTS videos (
    id varchar(255) PRIMARY KEY,
    source_path text NOT NULL,
    destination_path text NOT NULL
);

-- Define video_events table
CREATE TABLE IF NOT EXISTS video_events (
    video_id varchar(255) NOT NULL,
    video_event_id int NOT NULL,
    worker_name varchar(255) NOT NULL,
    event_time timestamp NOT NULL,
    event_type varchar(50) NOT NULL,
    notification_type varchar(50) NOT NULL,
    status varchar(20) NOT NULL,
    message text,
    PRIMARY KEY (video_id, video_event_id),
    FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE
);

-- Define workers table
CREATE TABLE IF NOT EXISTS workers (
    name varchar(100) PRIMARY KEY NOT NULL,
    ip varchar(100) NOT NULL,
    queue_name varchar(255) NOT NULL,
    last_seen timestamp NOT NULL
);

-- Define video_status table
CREATE TABLE IF NOT EXISTS video_status (
    video_id varchar(255) NOT NULL,
    video_event_id integer NOT NULL,
    video_path text NOT NULL,
    worker_name varchar(255) NOT NULL,
    event_time timestamp NOT NULL,
    event_type varchar(50) NOT NULL,
    notification_type varchar(50) NOT NULL,
    status varchar(20) NOT NULL,
    message text,
    CONSTRAINT video_status_pkey PRIMARY KEY (video_id),
    FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE
);

-- Function to insert or update video_status
CREATE OR REPLACE FUNCTION fn_video_status_update(
    p_video_id varchar,
    p_video_event_id integer,
    p_worker_name varchar,
    p_event_time timestamp,
    p_event_type varchar,
    p_notification_type varchar,
    p_status varchar,
    p_message text
) RETURNS VOID SECURITY DEFINER LANGUAGE plpgsql AS $$
DECLARE
    p_video_path varchar;
BEGIN
    SELECT v.source_path INTO p_video_path
    FROM videos v
    WHERE v.id = p_video_id;

    INSERT INTO video_status (
        video_id,
        video_event_id,
        video_path,
        worker_name,
        event_time,
        event_type,
        notification_type,
        status,
        message
    )
    VALUES (
        p_video_id,
        p_video_event_id,
        p_video_path,
        p_worker_name,
        p_event_time,
        p_event_type,
        p_notification_type,
        p_status,
        p_message
    )
    ON CONFLICT ON CONSTRAINT video_status_pkey DO UPDATE SET
        video_event_id = p_video_event_id,
        video_path = p_video_path,
        worker_name = p_worker_name,
        event_time = p_event_time,
        event_type = p_event_type,
        notification_type = p_notification_type,
        status = p_status,
        message = p_message;
END;
$$;

-- Trigger function for video_status_update
CREATE OR REPLACE FUNCTION fn_trigger_video_status_update() RETURNS TRIGGER SECURITY DEFINER LANGUAGE plpgsql AS $$
BEGIN
    PERFORM fn_video_status_update(
        NEW.video_id,
        NEW.video_event_id,
        NEW.worker_name,
        NEW.event_time,
        NEW.event_type,
        NEW.notification_type,
        NEW.status,
        NEW.message
    );
    RETURN NEW;
END;
$$;

-- Drop existing trigger if it exists
DROP TRIGGER IF EXISTS event_insert_video_status_update ON video_events;

-- Create trigger for video_events
CREATE TRIGGER event_insert_video_status_update
AFTER INSERT ON video_events
FOR EACH ROW
EXECUTE PROCEDURE fn_trigger_video_status_update();
