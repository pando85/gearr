DELETE FROM video_events
WHERE video_id NOT IN (SELECT id FROM videos);

ALTER TABLE video_events
DROP CONSTRAINT video_events_video_id_fkey;

ALTER TABLE video_events
ADD CONSTRAINT fk_video_events_videos
FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE;

-- Update video_status table
ALTER TABLE video_status
ADD CONSTRAINT fk_video_status_videos
FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE;
