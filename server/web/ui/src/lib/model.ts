export interface Job {
  id: string;
  source_path: string;
  destination_path: string;
  status: string;
  status_phase: string;
  status_message: string;
  last_update: Date;
  priority: number;
}

export function createJob(responseData: Partial<Job>): Job {
  return {
    id: responseData.id || '',
    source_path: responseData.source_path || '',
    destination_path: responseData.destination_path || '',
    status: responseData.status || 'queued',
    status_phase: responseData.status_phase || '',
    status_message: responseData.status_message || '',
    last_update: new Date(responseData.last_update || Date.now()),
    priority: responseData.priority ?? 1,
  };
}

export interface JobUpdateNotification {
  id: string;
  status: string;
  status_phase: string;
  message: string;
  event_time: Date;
  source_path: string;
  destination_path: string;
}

export function createJobUpdateNotification(
  responseData: Partial<JobUpdateNotification>
): JobUpdateNotification {
  return {
    id: responseData.id || '',
    status: responseData.status || '',
    status_phase: responseData.status_phase || '',
    message: responseData.message || '',
    event_time: new Date(responseData.event_time || Date.now()),
    source_path: responseData.source_path || '',
    destination_path: responseData.destination_path || '',
  };
}

export interface Worker {
  name: string;
  id: string;
  queue_name: string;
  last_seen: string;
}
