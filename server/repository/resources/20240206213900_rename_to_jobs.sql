-- Rename old tables to new naming convention (idempotent)
-- Only runs if old tables exist (upgrade from old schema)

DO $$
BEGIN
    -- Rename videos table to jobs if it exists
    IF EXISTS (SELECT FROM pg_tables WHERE tablename = 'videos') THEN
        ALTER TABLE videos RENAME TO jobs;
    END IF;

    -- Rename video_status table to job_status if it exists
    IF EXISTS (SELECT FROM pg_tables WHERE tablename = 'video_status') THEN
        ALTER TABLE video_status RENAME TO job_status;
        
        -- Rename columns if they have old names
        IF EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'job_status' AND column_name = 'video_id'
        ) THEN
            ALTER TABLE job_status RENAME COLUMN video_id TO job_id;
        END IF;
        
        IF EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'job_status' AND column_name = 'video_event_id'
        ) THEN
            ALTER TABLE job_status RENAME COLUMN video_event_id TO job_event_id;
        END IF;
        
        -- Update primary key constraint name
        IF EXISTS (
            SELECT 1 FROM information_schema.table_constraints 
            WHERE constraint_name = 'video_status_pkey' AND table_name = 'job_status'
        ) THEN
            ALTER TABLE job_status DROP CONSTRAINT video_status_pkey;
            IF NOT EXISTS (
                SELECT 1 FROM information_schema.table_constraints 
                WHERE constraint_name = 'job_status_pkey' AND table_name = 'job_status'
            ) THEN
                ALTER TABLE job_status ADD CONSTRAINT job_status_pkey PRIMARY KEY (job_id);
            END IF;
        END IF;
    END IF;

    -- Rename video_events table to job_events if it exists
    IF EXISTS (SELECT FROM pg_tables WHERE tablename = 'video_events') THEN
        ALTER TABLE video_events RENAME TO job_events;
        
        -- Rename columns if they have old names
        IF EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'job_events' AND column_name = 'video_id'
        ) THEN
            ALTER TABLE job_events RENAME COLUMN video_id TO job_id;
        END IF;
        
        IF EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'job_events' AND column_name = 'video_event_id'
        ) THEN
            ALTER TABLE job_events RENAME COLUMN video_event_id TO job_event_id;
        END IF;
    END IF;
    
    -- Drop old trigger and functions if they exist
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'event_insert_video_status_update') THEN
        DROP TRIGGER IF EXISTS event_insert_video_status_update ON job_events;
    END IF;
    
    IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'fn_trigger_video_status_update') THEN
        DROP FUNCTION IF EXISTS fn_trigger_video_status_update;
    END IF;
    
    IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'fn_video_status_update') THEN
        DROP FUNCTION IF EXISTS fn_video_status_update;
    END IF;
END $$;