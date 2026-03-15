import axios from 'axios';
import { jobStore } from './stores';
import { createJob, type Job, type Worker } from './model';

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
