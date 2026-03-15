-- Define jobs table
CREATE TABLE IF NOT EXISTS jobs (
    id varchar(255) PRIMARY KEY,
    source_path text NOT NULL,
    destination_path text NOT NULL
);

-- Define job_events table
CREATE TABLE IF NOT EXISTS job_events (
    job_id varchar(255) NOT NULL,
    job_event_id int NOT NULL,
    worker_name varchar(255) NOT NULL,
    event_time timestamp NOT NULL,
    event_type varchar(50) NOT NULL,
    notification_type varchar(50) NOT NULL,
    status varchar(20) NOT NULL,
    message text,
    PRIMARY KEY (job_id, job_event_id),
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

-- Define workers table
CREATE TABLE IF NOT EXISTS workers (
    name varchar(100) PRIMARY KEY NOT NULL,
    ip varchar(100) NOT NULL,
    queue_name varchar(255) NOT NULL,
    last_seen timestamp NOT NULL
);

-- Define job_status table
CREATE TABLE IF NOT EXISTS job_status (
    job_id varchar(255) NOT NULL,
    job_event_id integer NOT NULL,
    video_path text NOT NULL,
    worker_name varchar(255) NOT NULL,
    event_time timestamp NOT NULL,
    event_type varchar(50) NOT NULL,
    notification_type varchar(50) NOT NULL,
    status varchar(20) NOT NULL,
    message text,
    CONSTRAINT job_status_pkey PRIMARY KEY (job_id),
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

-- Function to insert or update job_status
CREATE
OR REPLACE FUNCTION fn_job_status_update(
    p_job_id varchar,
    p_job_event_id integer,
    p_worker_name varchar,
    p_event_time timestamp,
    p_event_type varchar,
    p_notification_type varchar,
    p_status varchar,
    p_message text
) RETURNS VOID SECURITY DEFINER LANGUAGE plpgsql AS $$ DECLARE p_video_path varchar;

BEGIN
SELECT
    v.source_path INTO p_video_path
FROM
    jobs v
WHERE
    v.id = p_job_id;

INSERT INTO
    job_status (
        job_id,
        job_event_id,
        video_path,
        worker_name,
        event_time,
        event_type,
        notification_type,
        status,
        message
    )
VALUES
    (
        p_job_id,
        p_job_event_id,
        p_video_path,
        p_worker_name,
        p_event_time,
        p_event_type,
        p_notification_type,
        p_status,
        p_message
    ) ON CONFLICT ON CONSTRAINT job_status_pkey DO
UPDATE
SET
    job_event_id = p_job_event_id,
    video_path = p_video_path,
    worker_name = p_worker_name,
    event_time = p_event_time,
    event_type = p_event_type,
    notification_type = p_notification_type,
    status = p_status,
    message = p_message;

END;

$$;

-- Trigger function for job_status_update
CREATE
OR REPLACE FUNCTION fn_trigger_job_status_update() RETURNS TRIGGER SECURITY DEFINER LANGUAGE plpgsql AS $$ BEGIN PERFORM fn_job_status_update(
    NEW.job_id,
    NEW.job_event_id,
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
DROP TRIGGER IF EXISTS event_insert_job_status_update ON job_events;

-- Create trigger for job_events
CREATE TRIGGER event_insert_job_status_update
AFTER
INSERT
    ON job_events FOR EACH ROW EXECUTE PROCEDURE fn_trigger_job_status_update();

-- Queue tables for PostgreSQL-based message broker
CREATE TABLE IF NOT EXISTS encode_queue (
    id SERIAL PRIMARY KEY,
    job_id varchar(255) NOT NULL,
    download_url text NOT NULL,
    upload_url text NOT NULL,
    checksum_url text NOT NULL,
    event_id int NOT NULL,
    created_at timestamp NOT NULL DEFAULT NOW(),
    locked_at timestamp,
    locked_by varchar(255),
    status varchar(20) NOT NULL DEFAULT 'pending'
);

CREATE INDEX IF NOT EXISTS idx_encode_queue_pending ON encode_queue (status, created_at) 
    WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS pgs_queue (
    id SERIAL PRIMARY KEY,
    job_id varchar(255) NOT NULL,
    pgs_id int NOT NULL,
    pgs_data bytea NOT NULL,
    pgs_language varchar(10) NOT NULL,
    reply_to_queue varchar(255) NOT NULL,
    created_at timestamp NOT NULL DEFAULT NOW(),
    locked_at timestamp,
    locked_by varchar(255),
    status varchar(20) NOT NULL DEFAULT 'pending'
);

CREATE INDEX IF NOT EXISTS idx_pgs_queue_pending ON pgs_queue (status, created_at) 
    WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS pgs_responses (
    id SERIAL PRIMARY KEY,
    job_id varchar(255) NOT NULL,
    pgs_id int NOT NULL,
    srt_data bytea,
    error text,
    reply_to_queue varchar(255) NOT NULL,
    created_at timestamp NOT NULL DEFAULT NOW(),
    consumed boolean NOT NULL DEFAULT false,
    consumed_at timestamp
);

CREATE INDEX IF NOT EXISTS idx_pgs_responses_pending ON pgs_responses (reply_to_queue, consumed, created_at) 
    WHERE consumed = false;

CREATE TABLE IF NOT EXISTS task_event_queue (
    id SERIAL PRIMARY KEY,
    job_id varchar(255) NOT NULL,
    event_id int NOT NULL,
    event_type varchar(50) NOT NULL,
    worker_name varchar(255) NOT NULL,
    worker_queue varchar(255) NOT NULL,
    event_time timestamp NOT NULL,
    ip varchar(100),
    notification_type varchar(50) NOT NULL,
    status varchar(20) NOT NULL,
    message text,
    created_at timestamp NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_task_event_pending ON task_event_queue (created_at);

CREATE TABLE IF NOT EXISTS job_actions (
    id SERIAL PRIMARY KEY,
    job_id varchar(255) NOT NULL,
    worker_name varchar(255) NOT NULL,
    action varchar(50) NOT NULL,
    created_at timestamp NOT NULL DEFAULT NOW(),
    consumed boolean NOT NULL DEFAULT false,
    consumed_at timestamp
);

CREATE INDEX IF NOT EXISTS idx_job_actions_pending ON job_actions (worker_name, consumed, created_at) 
    WHERE consumed = false;

CREATE TABLE IF NOT EXISTS file_processing (
    id SERIAL PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    detected_at TIMESTAMP NOT NULL,
    source VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    message TEXT,
    job_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_file_processing_time ON file_processing(detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_file_processing_source ON file_processing(source);
CREATE INDEX IF NOT EXISTS idx_file_processing_status ON file_processing(status);
