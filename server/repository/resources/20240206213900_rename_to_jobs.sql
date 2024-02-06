-- Rename videos table to jobs
ALTER TABLE
    videos RENAME TO jobs;

-- Rename video_status table to job_status
ALTER TABLE
    video_status RENAME TO job_status;

-- Rename video_events table to job_events
ALTER TABLE
    video_events RENAME TO job_events;

-- Rename columns in job_events table
ALTER TABLE
    job_events RENAME COLUMN video_id TO job_id;

ALTER TABLE
    job_events RENAME COLUMN video_event_id TO job_event_id;

-- Rename columns in job_status table
ALTER TABLE
    job_status RENAME COLUMN video_id TO job_id;

ALTER TABLE
    job_status RENAME COLUMN video_event_id TO job_event_id;

ALTER TABLE
    job_status DROP CONSTRAINT video_status_pkey,
ADD
    CONSTRAINT job_status_pkey PRIMARY KEY (job_id);

-- Rename function fn_video_status_update to fn_job_status_update if exists
ALTER FUNCTION fn_video_status_update(
    p_video_id varchar,
    p_video_event_id integer,
    p_worker_name varchar,
    p_event_time timestamp,
    p_event_type varchar,
    p_notification_type varchar,
    p_status varchar,
    p_message text
) RENAME TO fn_job_status_update;

-- Rename function fn_trigger_video_status_update to fn_trigger_job_status_update if exists
ALTER FUNCTION fn_trigger_video_status_update() RENAME TO fn_trigger_job_status_update;

-- Drop trigger event_insert_video_status_update if exists
DROP TRIGGER IF EXISTS event_insert_video_status_update ON video_events;

-- Create trigger event_insert_job_status_update if not exists
CREATE TRIGGER event_insert_job_status_update
AFTER
INSERT
    ON job_events FOR EACH ROW EXECUTE FUNCTION fn_trigger_job_status_update();
