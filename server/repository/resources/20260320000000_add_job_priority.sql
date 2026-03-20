-- Add priority column to jobs table
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS priority INTEGER NOT NULL DEFAULT 5;

-- Create index for priority-based queries
CREATE INDEX IF NOT EXISTS idx_jobs_priority ON jobs(priority DESC);