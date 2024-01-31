// jobReducer.ts
import { Reducer } from 'redux';
import {
    FETCH_JOBS_REQUEST,
    FETCH_JOBS_SUCCESS,
    FETCH_JOBS_FAILURE,
    DELETE_JOB_REQUEST,
    DELETE_JOB_SUCCESS,
    DELETE_JOB_FAILURE,
    CREATE_JOB_REQUEST,
    CREATE_JOB_FAILURE,
    UPDATE_JOB,
    RELOAD_JOBS,
    JobActionTypes
} from './actions/JobActionsTypes';

import { Job, JobClass } from './model';
interface JobState {
    jobs: Job[];
    loading: boolean;
}

const initialState: JobState = {
    jobs: [],
    loading: true,
};

const jobReducer: Reducer<JobState, JobActionTypes> = (state = initialState, action) => {
    switch (action.type) {
        case FETCH_JOBS_REQUEST:
            return {
                ...state,
                loading: true,
                error: null,
            };

        case FETCH_JOBS_SUCCESS:
            return {
                ...state,
                loading: false,
                jobs: [...action.payload],
            };

        case FETCH_JOBS_FAILURE:
            return {
                ...state,
                loading: false,
                error: action.error,
            };
        case DELETE_JOB_REQUEST:
            return {
                ...state,
                loading: true,
                error: null,
            };

        case DELETE_JOB_SUCCESS:
            return {
                ...state,
                loading: false,
                jobs: state.jobs.filter((job) => job.id !== action.payload),
            };

        case DELETE_JOB_FAILURE:
            return {
                ...state,
                loading: false,
                error: action.error,
            };

        case CREATE_JOB_REQUEST:
            return {
                ...state,
                loading: true,
                error: null,
            };

        case CREATE_JOB_FAILURE:
            return {
                ...state,
                loading: false,
                error: action.error,
            };
        case UPDATE_JOB:
            const updatedJobIndex = state.jobs.findIndex((job) => job.id === action.payload.id);

            if (updatedJobIndex !== -1) {
                state.jobs[updatedJobIndex].status = action.payload.status;
                state.jobs[updatedJobIndex].status_message = action.payload.message;
                state.jobs[updatedJobIndex].last_update = action.payload.event_time;
                const updatedJobs = [...state.jobs];
                return {
                    ...state,
                    loading: false,
                    jobs: updatedJobs,
                };
            } else {
                const newJob: Job = new JobClass({
                    id: action.payload.id,
                    source_path: action.payload.source_path,
                    destination_path: action.payload.destination_path,
                });
                return {
                    ...state,
                    loading: false,
                    jobs: [...state.jobs, newJob],
                };
            }
        case RELOAD_JOBS:
            return {
                ...state,
                loading: true,
                jobs: [],
            };
        default:
            return state;
    }
};
export type { JobState };
export { initialState };
export default jobReducer;
