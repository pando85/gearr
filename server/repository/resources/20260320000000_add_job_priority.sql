-- Add priority column to jobs table for smart prioritization feature
-- Priority is an integer where higher values indicate higher priority
-- Default priority is 0 (normal priority)

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS priority INTEGER NOT NULL DEFAULT 0;

-- Add index for priority-based sorting queries
CREATE INDEX IF NOT EXISTS idx_jobs_priority ON jobs(priority DESC);