export interface WebhookEvent {
  id: number;
  source: string;
  event_type: string;
  file_path: string;
  status: string;
  message: string;
  payload: string;
  job_id: string | null;
  created_at: Date;
  error_details: string;
}

export function createWebhookEvent(responseData: Partial<WebhookEvent>): WebhookEvent {
  return {
    id: responseData.id || 0,
    source: responseData.source || '',
    event_type: responseData.event_type || '',
    file_path: responseData.file_path || '',
    status: responseData.status || '',
    message: responseData.message || '',
    payload: responseData.payload || '',
    job_id: responseData.job_id || null,
    created_at: new Date(responseData.created_at || Date.now()),
    error_details: responseData.error_details || '',
  };
}