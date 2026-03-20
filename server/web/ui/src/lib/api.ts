import axios from 'axios';
import { jobStore } from './stores';
import { scannerStore } from './stores/scanner';
import { createJob, type Job, type Worker } from './model';
import type { ScannerStatus, LibraryScan } from './stores/scanner-model';
import { createWebhookEvent, type WebhookEvent } from './webhook-model';

export type { Worker };

export async function fetchJobs(token: string): Promise<Job[]> {
  jobStore.setLoading();

  try {
    const response = await axios.get('/api/v1/job/', {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });

    const jobs: Job[] = response.data.map((jobData: Partial<Job>) => createJob(jobData));
    jobStore.setJobs(jobs);
    return jobs;
  } catch (error) {
    console.error('Error fetching jobs:', error);
    jobStore.setError('Failed to fetch jobs.');
    throw error;
  }
}

export async function deleteJob(token: string, jobId: string): Promise<void> {
  jobStore.setLoading();

  try {
    await axios.delete(`/api/v1/job/${jobId}`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
    jobStore.removeJob(jobId);
  } catch (error) {
    console.error(`Error deleting job ${jobId}:`, error);
    jobStore.setError('Failed to delete job.');
    throw error;
  }
}

export async function createJobRequest(token: string, path: string): Promise<void> {
  jobStore.setLoading();

  try {
    await axios.post(
      `/api/v1/job/`,
      {
        source_path: path,
      },
      {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      }
    );
  } catch (error) {
    console.error(`Error creating job with path ${path}:`, error);
    jobStore.setError('Failed to create job.');
    throw error;
  }
}

export async function fetchWorkers(token: string) {
  const response = await axios.get('/api/v1/workers', {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
  return response.data;
}

export async function fetchScannerStatus(token: string): Promise<ScannerStatus> {
  scannerStore.setLoading();
  
  try {
    const response = await axios.get('/api/v1/scanner/status', {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
    scannerStore.setStatus(response.data);
    return response.data;
  } catch (error) {
    console.error('Error fetching scanner status:', error);
    scannerStore.setError('Failed to fetch scanner status.');
    throw error;
  }
}

export async function triggerScan(token: string): Promise<void> {
  try {
    await axios.post('/api/v1/scanner/scan', {}, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
  } catch (error) {
    console.error('Error triggering scan:', error);
    throw error;
  }
}

export async function fetchScanHistory(token: string): Promise<LibraryScan[]> {
  try {
    const response = await axios.get('/api/v1/scanner/history', {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
    scannerStore.setHistory(response.data);
    return response.data;
  } catch (error) {
    console.error('Error fetching scan history:', error);
    throw error;
  }
}

export async function updateJobPriority(token: string, jobId: string, priority: number): Promise<void> {
  try {
    await axios.patch(`/api/v1/job/${jobId}/priority`, 
      { priority },
      {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      }
    );
  } catch (error) {
    console.error(`Error updating job priority ${jobId}:`, error);
    throw error;
  }
}

export async function fetchWebhookEvents(
  token: string,
  source?: string,
  eventType?: string,
  status?: string
): Promise<WebhookEvent[]> {
  try {
    const params = new URLSearchParams();
    if (source) params.append('source', source);
    if (eventType) params.append('event_type', eventType);
    if (status) params.append('status', status);

    const queryString = params.toString();
    const url = `/api/v1/webhook/events${queryString ? `?${queryString}` : ''}`;

    const response = await axios.get(url, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
    return response.data.map((event: Partial<WebhookEvent>) => createWebhookEvent(event));
  } catch (error) {
    console.error('Error fetching webhook events:', error);
    throw error;
  }
}

export async function fetchWebhookEvent(token: string, id: number): Promise<WebhookEvent> {
  try {
    const response = await axios.get(`/api/v1/webhook/events/${id}`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
    return createWebhookEvent(response.data);
  } catch (error) {
    console.error(`Error fetching webhook event ${id}:`, error);
    throw error;
  }
}
