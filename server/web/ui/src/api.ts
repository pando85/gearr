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
    createJobSuccess,
    createJobFailure,
} from './actions/JobActions';
import {JobActionTypes} from './actions/JobActionsTypes';

export const fetchJobs = (token: string, setShowJobTable: any) => async (dispatch: Dispatch<JobActionTypes>): Promise<void> => {
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
        dispatch(fetchJobsFailure('Failed to fetch jobs.'));
    }
};


export const deleteJob = (token: string, setShowJobTable: any, jobId: string) => async (dispatch: Dispatch<JobActionTypes>): Promise<void> => {
    dispatch(deleteJobRequest());

    try {
        await axios.delete(`/api/v1/job/${jobId}`, {
            headers: {
                Authorization: `Bearer ${token}`,
            },
        });
        dispatch(deleteJobSuccess(jobId));
        // setJobs as needed
    } catch (error) {
        console.error(`Error deleting job ${jobId}:`, error);
        setShowJobTable(false);
        dispatch(deleteJobFailure('Failed to delete job.'));
    }
};

export const createJob = (token: string, setShowJobTable: any, path: string) => async (dispatch: Dispatch<JobActionTypes>): Promise<void> => {
    dispatch(createJobRequest());

    try {
        const response = await axios.post(
            `/api/v1/job/`,
            {
                SourcePath: path,
            },
            {
                headers: {
                    Authorization: `Bearer ${token}`,
                },
            }
        );
        const newJobs: Job[] = response.data.scheduled;
        dispatch(createJobSuccess(newJobs));
    } catch (error) {
        console.error(`Error creating job with path ${path}:`, error);
        setShowJobTable(false);
        dispatch(createJobFailure('Failed to create job.'));
    }
};

