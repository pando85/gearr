import { connect } from 'react-redux';
import { ThunkDispatch } from 'redux-thunk';
import { Action } from 'redux';
import JobTable from './JobTable';
import { Job } from './model';
import {
    fetchJobsSuccess,
    deleteJobSuccess,
} from './actions/JobActions';

interface DispatchProps {
    fetchJobsSuccess: (jobs: Job[]) => void;
    deleteJobSuccess: (jobId: string) => void;
}

const mapDispatchToProps = (
    dispatch: ThunkDispatch<{}, {}, Action>
): DispatchProps => ({
    fetchJobsSuccess: (jobs) => dispatch(fetchJobsSuccess(jobs)),
    deleteJobSuccess: (jobId) => dispatch(deleteJobSuccess(jobId)),
});

export default connect(null, mapDispatchToProps)(JobTable);
