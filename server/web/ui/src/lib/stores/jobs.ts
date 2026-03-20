import { writable, derived } from 'svelte/store';
import type { Job, JobUpdateNotification } from '$lib/model';
import { createJob } from '$lib/model';

export interface JobState {
  jobs: Job[];
  loading: boolean;
  error: string | null;
}

const initialState: JobState = {
  jobs: [],
  loading: true,
  error: null,
};

function createJobStore() {
  const { subscribe, update } = writable<JobState>(initialState);

  return {
    subscribe,
    setLoading: () => update((state) => ({ ...state, loading: true, error: null })),
    setJobs: (jobs: Job[]) => update((state) => ({ ...state, loading: false, jobs })),
    setError: (error: string) => update((state) => ({ ...state, loading: false, error })),
    removeJob: (jobId: string) =>
      update((state) => ({
        ...state,
        loading: false,
        jobs: state.jobs.filter((job) => job.id !== jobId),
      })),
    updateJob: (notification: JobUpdateNotification) =>
      update((state) => {
        const updatedJobIndex = state.jobs.findIndex((job) => job.id === notification.id);

        if (updatedJobIndex !== -1) {
          const updatedJobs = [...state.jobs];
          updatedJobs[updatedJobIndex] = {
            ...updatedJobs[updatedJobIndex],
            status: notification.status,
            status_phase: notification.status_phase,
            status_message: notification.message,
            last_update: notification.event_time,
          };
          return { ...state, loading: false, jobs: updatedJobs };
        } else {
          const newJob: Job = createJob({
            id: notification.id,
            status: notification.status,
            status_phase: notification.status_phase,
            source_path: notification.source_path,
            destination_path: notification.destination_path,
          });
          return { ...state, loading: false, jobs: [...state.jobs, newJob] };
        }
      }),
    updateJobPriority: (jobId: string, priority: number) =>
      update((state) => {
        const updatedJobIndex = state.jobs.findIndex((job) => job.id === jobId);
        if (updatedJobIndex !== -1) {
          const updatedJobs = [...state.jobs];
          updatedJobs[updatedJobIndex] = {
            ...updatedJobs[updatedJobIndex],
            priority,
          };
          return { ...state, loading: false, jobs: updatedJobs };
        }
        return state;
      }),
    reset: () => update((state) => ({ ...state, loading: true, jobs: [] })),
  };
}

export const jobStore = createJobStore();

export const jobStats = derived(jobStore, ($jobStore) => {
  const jobs = $jobStore.jobs;
  return {
    total: jobs.length,
    progressing: jobs.filter((j) => j.status === 'progressing').length,
    completed: jobs.filter((j) => j.status === 'completed').length,
    failed: jobs.filter((j) => j.status === 'failed').length,
    queued: jobs.filter((j) => j.status === 'queued').length,
  };
});
