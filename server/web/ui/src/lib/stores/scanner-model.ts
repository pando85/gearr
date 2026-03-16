export interface ScannerConfig {
  enabled: boolean;
  interval: number;
  min_file_size: number;
  paths: string[];
  file_extensions: string[];
}

export interface LibraryScan {
  id: string;
  started_at: string;
  completed_at?: string;
  status: 'running' | 'completed' | 'failed';
  files_found: number;
  files_queued: number;
  files_skipped_size: number;
  files_skipped_codec: number;
  files_skipped_exists: number;
  error_message?: string;
}

export interface ScannerStatus {
  enabled: boolean;
  is_scanning: boolean;
  last_scan: LibraryScan | null;
  next_scan_at: string | null;
  config: ScannerConfig;
}

export interface ScannerNotification {
  type: 'scan_started' | 'scan_progress' | 'scan_completed';
  scan_id: string;
  progress: number;
  files_found: number;
  files_queued: number;
  files_skipped: number;
  current_path?: string;
  status: 'running' | 'completed' | 'failed';
  error_message?: string;
}