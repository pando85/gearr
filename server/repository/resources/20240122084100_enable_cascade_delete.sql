-- Only run this migration if old tables exist (upgrade from old schema)
-- This migration is idempotent and safe to run on fresh databases

DO $$
BEGIN
    -- Check if video_events table exists (old schema)
    IF EXISTS (SELECT FROM pg_tables WHERE tablename = 'video_events') THEN
        -- Clean up orphaned records
        DELETE FROM video_events
        WHERE video_id NOT IN (SELECT id FROM videos);

        -- Drop old constraint if it exists
        IF EXISTS (
            SELECT 1 FROM information_schema.table_constraints 
            WHERE constraint_name = 'video_events_video_id_fkey' 
            AND table_name = 'video_events'
        ) THEN
            ALTER TABLE video_events
            DROP CONSTRAINT video_events_video_id_fkey;
        END IF;

        -- Add new cascade delete constraint
        ALTER TABLE video_events
        ADD CONSTRAINT fk_video_events_videos
        FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE;
    END IF;

    -- Check if video_status table exists (old schema)
    IF EXISTS (SELECT FROM pg_tables WHERE tablename = 'video_status') THEN
        -- Add cascade delete constraint if it doesn't exist
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints 
            WHERE constraint_name = 'fk_video_status_videos' 
            AND table_name = 'video_status'
        ) THEN
            ALTER TABLE video_status
            ADD CONSTRAINT fk_video_status_videos
            FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE;
        END IF;
    END IF;
END $$;