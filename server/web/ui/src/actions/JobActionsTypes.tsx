import { Job } from '../model';

export const FETCH_JOBS_REQUEST = 'FETCH_JOBS_REQUEST';
export const FETCH_JOBS_SUCCESS = 'FETCH_JOBS_SUCCESS';
export const FETCH_JOBS_FAILURE = 'FETCH_JOBS_FAILURE';

export const DELETE_JOB_REQUEST = 'DELETE_JOB_REQUEST';
export const DELETE_JOB_SUCCESS = 'DELETE_JOB_SUCCESS';
export const DELETE_JOB_FAILURE = 'DELETE_JOB_FAILURE';

export const CREATE_JOB_REQUEST = 'CREATE_JOB_REQUEST';
export const CREATE_JOB_SUCCESS = 'CREATE_JOB_SUCCESS';
export const CREATE_JOB_FAILURE = 'CREATE_JOB_FAILURE';

export const RELOAD_JOBS = 'RELOAD_JOBS';

interface FetchJobsRequestAction {
    type: typeof FETCH_JOBS_REQUEST;
}

interface FetchJobsSuccessAction {
    type: typeof FETCH_JOBS_SUCCESS;
    payload: Job[];
}

interface FetchJobsFailureAction {
    type: typeof FETCH_JOBS_FAILURE;
    error: string;
}

interface DeleteJobRequestAction {
    type: typeof DELETE_JOB_REQUEST;
}

interface DeleteJobSuccessAction {
    type: typeof DELETE_JOB_SUCCESS;
    payload: string; // Job ID
}

interface DeleteJobFailureAction {
    type: typeof DELETE_JOB_FAILURE;
    error: string;
}

interface CreateJobRequestAction {
    type: typeof CREATE_JOB_REQUEST;
}

interface CreateJobSuccessAction {
    type: typeof CREATE_JOB_SUCCESS;
    payload: Job[];
}

interface CreateJobFailureAction {
    type: typeof CREATE_JOB_FAILURE;
    error: string;
}

interface ResetJobsAction {
    type: typeof RELOAD_JOBS;
}

export type JobActionTypes =
    | FetchJobsRequestAction
    | FetchJobsSuccessAction
    | FetchJobsFailureAction
    | DeleteJobRequestAction
    | DeleteJobSuccessAction
    | DeleteJobFailureAction
    | CreateJobRequestAction
    | CreateJobSuccessAction
    | CreateJobFailureAction
    | ResetJobsAction;

export type {
    FetchJobsRequestAction,
    FetchJobsSuccessAction,
    FetchJobsFailureAction,
    DeleteJobRequestAction,
    DeleteJobSuccessAction,
    DeleteJobFailureAction,
    CreateJobRequestAction,
    CreateJobSuccessAction,
    CreateJobFailureAction,
    ResetJobsAction,
}
