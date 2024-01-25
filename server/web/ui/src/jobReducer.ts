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
    CREATE_JOB_SUCCESS,
    CREATE_JOB_FAILURE,
    RELOAD_JOBS,
    JobActionTypes
} from './actions/JobActionsTypes';

import { Job} from './model';
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

        case CREATE_JOB_SUCCESS:
            return {
                ...state,
                loading: false,
                jobs: [...state.jobs, ...action.payload],
            };

        case CREATE_JOB_FAILURE:
            return {
                ...state,
                loading: false,
                error: action.error,
            };
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
export type {JobState};
export {initialState};
export default jobReducer;
