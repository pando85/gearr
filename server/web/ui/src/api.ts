import axios from 'axios';
import { Dispatch } from 'redux';
import { Job, JobClass } from './model';
import {
    fetchJobsRequest,
    fetchJobsSuccess,
    fetchJobsFailure,
    deleteJobRequest,
    deleteJobSuccess,
    deleteJobFailure,
    createJobRequest,
    createJobFailure,
} from './actions/JobActions';
import { JobActionTypes } from './actions/JobActionsTypes';

export const fetchJobs = (token: string, setShowJobTable: any, setErrorText: any) => async (dispatch: Dispatch<JobActionTypes>): Promise<void> => {
    dispatch(fetchJobsRequest());

    try {
        const response = await axios.get('/api/v1/job/', {
            headers: {
                Authorization: `Bearer ${token}`,
            },
        });

        const newJobs: Job[] = response.data.map((jobData: any) => new JobClass(jobData));
        dispatch(fetchJobsSuccess(newJobs));
    } catch (error) {
        console.error('Error fetching jobs:', error);
        setShowJobTable(false);
        console.log(error as string);
        if (error instanceof Error) {
            setErrorText(error.message);
        }
        dispatch(fetchJobsFailure('Failed to fetch jobs.'));
    }
};


export const deleteJob = (token: string, setShowJobTable: any, setErrorText: any, jobId: string) => async (dispatch: Dispatch<JobActionTypes>): Promise<void> => {
    dispatch(deleteJobRequest());

    try {
        await axios.delete(`/api/v1/job/${jobId}`, {
            headers: {
                Authorization: `Bearer ${token}`,
            },
        });
        dispatch(deleteJobSuccess(jobId));
    } catch (error) {
        console.error(`Error deleting job ${jobId}:`, error);
        setShowJobTable(false);
        if (error instanceof Error) {
            setErrorText(error.message);
        }
        dispatch(deleteJobFailure('Failed to delete job.'));
    }
};

export const createJob = (token: string, setShowJobTable: any, setErrorText: any, path: string) => async (dispatch: Dispatch<JobActionTypes>): Promise<void> => {
    dispatch(createJobRequest());
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
        setShowJobTable(false);
        if (error instanceof Error) {
            setErrorText(error.message);
        }
        dispatch(createJobFailure('Failed to create job.'));
    }
};

