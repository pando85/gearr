import {
  FETCH_JOBS_REQUEST,
  FETCH_JOBS_SUCCESS,
  FETCH_JOBS_FAILURE,
  DELETE_JOB_REQUEST,
  DELETE_JOB_SUCCESS,
  DELETE_JOB_FAILURE,
  CREATE_JOB_REQUEST,
  CREATE_JOB_FAILURE,
  RELOAD_JOBS as RESET_JOBS,
  FetchJobsRequestAction,
  FetchJobsSuccessAction,
  FetchJobsFailureAction,
  DeleteJobRequestAction,
  DeleteJobSuccessAction,
  DeleteJobFailureAction,
  CreateJobRequestAction,
  CreateJobFailureAction,
  ResetJobsAction,
  UpdateJobAction,
  UPDATE_JOB,
} from './JobActionsTypes';

import { Job, JobUpdateNotification } from '../model';



export const fetchJobsRequest = (): FetchJobsRequestAction => ({
  type: FETCH_JOBS_REQUEST,
});

export const fetchJobsSuccess = (jobs: Job[]): FetchJobsSuccessAction => ({
  type: FETCH_JOBS_SUCCESS,
  payload: jobs,
});

export const fetchJobsFailure = (error: string): FetchJobsFailureAction => ({
  type: FETCH_JOBS_FAILURE,
  error,
});

export const deleteJobRequest = (): DeleteJobRequestAction => ({
  type: DELETE_JOB_REQUEST,
});

export const deleteJobSuccess = (jobId: string): DeleteJobSuccessAction => ({
  type: DELETE_JOB_SUCCESS,
  payload: jobId,
});

export const deleteJobFailure = (error: string): DeleteJobFailureAction => ({
  type: DELETE_JOB_FAILURE,
  error,
});

export const createJobRequest = (): CreateJobRequestAction => ({
  type: CREATE_JOB_REQUEST,
});

export const createJobFailure = (error: string): CreateJobFailureAction => ({
  type: CREATE_JOB_FAILURE,
  error,
});

export const updateJob = (JobUpdateNotification: JobUpdateNotification): UpdateJobAction => ({
  type: UPDATE_JOB,
  payload: JobUpdateNotification,
});

export const resetJobs = (): ResetJobsAction => ({
  type: RESET_JOBS,
});
