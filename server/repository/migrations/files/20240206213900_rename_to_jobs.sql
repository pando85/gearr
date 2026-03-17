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

DROP TRIGGER event_insert_video_status_update ON job_events;

DROP FUNCTION fn_trigger_video_status_update;
DROP FUNCTION fn_video_status_update;
