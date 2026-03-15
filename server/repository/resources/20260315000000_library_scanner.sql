CREATE TABLE IF NOT EXISTS library_scans (
    id varchar(255) PRIMARY KEY,
    started_at timestamp NOT NULL,
    completed_at timestamp,
    status varchar(20) NOT NULL,
    files_found integer DEFAULT 0,
    files_queued integer DEFAULT 0,
    files_skipped_size integer DEFAULT 0,
    files_skipped_codec integer DEFAULT 0,
    files_skipped_exists integer DEFAULT 0,
    error_message text
);

CREATE TABLE IF NOT EXISTS scanned_files (
    id varchar(255) PRIMARY KEY,
    file_path text NOT NULL UNIQUE,
    file_size bigint NOT NULL,
    codec varchar(50),
    last_scanned_at timestamp NOT NULL,
    queued boolean DEFAULT FALSE,
    scan_id varchar(255) REFERENCES library_scans(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_scanned_files_path ON scanned_files(file_path);
CREATE INDEX IF NOT EXISTS idx_scanned_files_codec ON scanned_files(codec);
CREATE INDEX IF NOT EXISTS idx_scanned_files_queued ON scanned_files(queued);