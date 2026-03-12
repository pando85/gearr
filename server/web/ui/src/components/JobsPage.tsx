import React, { useEffect, useState, useCallback } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { FixedSizeList } from 'react-window';
import {
  Search,
  Refresh,
  ArrowUpward,
  ArrowDownward,
  Delete,
  Replay,
  Info,
  Error,
  Assignment,
} from '@mui/icons-material';
import useWebSocket from 'react-use-websocket';
import { Job, JobUpdateNotification, JobUpdateNotificationClass } from '../model';
import { fetchJobs, deleteJob, createJob } from '../api';
import { RootState } from '../store';
import { STATUS_FILTER_OPTIONS, DATE_FILTER_OPTIONS, formatDateShort, formatDateDetailed, getDateFromFilterOption, sortJobs } from '../utils';
import { updateJob, resetJobs } from '../actions/JobActions';
import { useToast } from './Toast';
import '../styles/JobsPage.css';

interface JobsPageProps {
  token: string;
  setShowJobTable: React.Dispatch<React.SetStateAction<boolean>>;
  setErrorText: React.Dispatch<React.SetStateAction<string>>;
}

const JobsPage: React.FC<JobsPageProps> = ({ token, setShowJobTable, setErrorText }) => {
  const dispatch = useDispatch();
  const jobs: Job[] = useSelector((state: RootState) => state.jobs);
  const toast = useToast();

  const [filteredJobs, setFilteredJobs] = useState<Job[]>([]);
  const [nameFilter, setNameFilter] = useState<string>('');
  const [statusFilter, setStatusFilter] = useState<string[]>([]);
  const [dateFilter, setDateFilter] = useState<string>('');
  const [sortColumn, setSortColumn] = useState<string | null>('last_update');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('desc');
  const [selectedJob, setSelectedJob] = useState<Job | null>(null);
  const [height, setHeight] = useState(window.innerHeight - 200);

  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
  const wsURL = `${protocol}://${window.location.hostname}:${window.location.port}/ws/job?token=${token}`;
  const { lastMessage } = useWebSocket(wsURL);

  const handleReload = useCallback(() => {
    dispatch(resetJobs() as any);
    dispatch(fetchJobs(token, setShowJobTable, setErrorText) as any);
    toast.showToast({ type: 'info', message: 'Refreshing jobs...' });
  }, [dispatch, token, setShowJobTable, setErrorText, toast]);

  const handleDeleteJob = async (jobId: string) => {
    try {
      await dispatch(deleteJob(token, setShowJobTable, setErrorText, jobId) as any);
      toast.showToast({ type: 'success', message: 'Job deleted successfully' });
    } catch (error) {
      toast.showToast({ type: 'error', message: 'Failed to delete job' });
    }
  };

  const handleRecreateJob = async (job: Job) => {
    try {
      await dispatch(deleteJob(token, setShowJobTable, setErrorText, job.id) as any);
      await dispatch(createJob(token, setShowJobTable, setErrorText, job.source_path) as any);
      toast.showToast({ type: 'success', message: 'Job recreated successfully' });
    } catch (error) {
      toast.showToast({ type: 'error', message: 'Failed to recreate job' });
    }
  };

  const handleSort = (column: string) => {
    if (sortColumn === column) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortColumn(column);
      setSortDirection('asc');
    }
  };

  useEffect(() => {
    if (lastMessage !== null) {
      const notification: JobUpdateNotification = new JobUpdateNotificationClass(JSON.parse(lastMessage.data));
      dispatch(updateJob(notification) as any);
    }
  }, [dispatch, lastMessage]);

  useEffect(() => {
    dispatch(fetchJobs(token, setShowJobTable, setErrorText) as any);
  }, [dispatch, token, setShowJobTable, setErrorText]);

  useEffect(() => {
    const handleResize = () => {
      setHeight(window.innerHeight - 200);
    };
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  useEffect(() => {
    let result = jobs;

    if (statusFilter.length > 0) {
      result = result.filter((job) => statusFilter.includes(job.status));
    }

    if (dateFilter) {
      result = result.filter((job) => job.last_update >= getDateFromFilterOption(dateFilter));
    }

    if (nameFilter) {
      result = result.filter((job) => 
        job.source_path?.toLowerCase().includes(nameFilter.toLowerCase())
      );
    }

    setFilteredJobs(sortJobs(sortColumn, sortDirection, result));
  }, [jobs, statusFilter, dateFilter, nameFilter, sortColumn, sortDirection]);

  const getStatusBadgeClass = (status: string) => {
    const classes: Record<string, string> = {
      progressing: 'progressing',
      completed: 'completed',
      failed: 'failed',
      queued: 'queued',
    };
    return classes[status] || 'queued';
  };

  const renderStatus = (job: Job) => {
    if (job.status === 'progressing' && job.status_phase === 'FFMPEG') {
      try {
        const messageObj = JSON.parse(job.status_message);
        const progress = messageObj.progress !== undefined ? parseFloat(messageObj.progress) : 0;
        return (
          <div className="job-progress" title={`${progress.toFixed(2)}%`}>
            <div className="job-progress-bar">
              <div className="job-progress-fill" style={{ width: `${progress}%` }} />
            </div>
          </div>
        );
      } catch {
        return <span className={`job-status-badge ${getStatusBadgeClass(job.status)}`}>{job.status}</span>;
      }
    }

    if (job.status === 'failed') {
      return (
        <div title={job.status_message}>
          <Error className="job-error-icon" />
        </div>
      );
    }

    const displayStatus = job.status_phase !== 'Job' ? job.status_phase.toLowerCase() : job.status;
    return <span className={`job-status-badge ${getStatusBadgeClass(job.status)}`}>{displayStatus}</span>;
  };

  const truncatePath = (path: string, maxLength: number = 40) => {
    const fileName = path.split('/').pop() || path;
    if (fileName.length > maxLength) {
      return fileName.substring(0, maxLength) + '...';
    }
    return fileName;
  };

  const Row = ({ index, style }: { index: number; style: React.CSSProperties }) => {
    const job = filteredJobs[index];
    if (!job) return null;

    return (
      <div className="jobs-tr" style={style}>
        <div className="jobs-td jobs-td-source" title={job.source_path}>
          <span className="job-path">{truncatePath(job.source_path)}</span>
        </div>
        <div className="jobs-td jobs-td-destination" title={job.destination_path}>
          <span className="job-path">{truncatePath(job.destination_path)}</span>
        </div>
        <div className="jobs-td jobs-td-status">{renderStatus(job)}</div>
        <div className="jobs-td jobs-td-date" title={formatDateDetailed(job.last_update)}>
          <span className="job-date">{formatDateShort(job.last_update)}</span>
        </div>
        <div className="jobs-td jobs-td-actions">
          <div className="job-actions">
            <button
              className="job-action-btn"
              onClick={() => setSelectedJob(job)}
              title="Details"
            >
              <Info fontSize="small" />
            </button>
            <button
              className="job-action-btn delete"
              onClick={() => handleDeleteJob(job.id)}
              title="Delete"
            >
              <Delete fontSize="small" />
            </button>
            <button
              className="job-action-btn"
              onClick={() => handleRecreateJob(job)}
              title="Recreate"
            >
              <Replay fontSize="small" />
            </button>
          </div>
        </div>
      </div>
    );
  };

  const SortIcon = ({ column }: { column: string }) => {
    if (sortColumn !== column) return <ArrowDownward className="jobs-sort-icon" />;
    return sortDirection === 'asc' 
      ? <ArrowUpward className="jobs-sort-icon active" />
      : <ArrowDownward className="jobs-sort-icon active" />;
  };

  return (
    <div className="jobs-page">
      <div className="jobs-header">
        <h1 className="jobs-title">Jobs</h1>
        <div className="jobs-actions">
          <div className="jobs-search">
            <Search className="jobs-search-icon" fontSize="small" />
            <input
              className="jobs-search-input"
              type="text"
              placeholder="Search jobs..."
              value={nameFilter}
              onChange={(e) => setNameFilter(e.target.value)}
            />
          </div>
          <div className="jobs-filters">
            <select
              className="jobs-filter-select"
              value={statusFilter.length > 0 ? statusFilter[0] : ''}
              onChange={(e) => setStatusFilter(e.target.value ? [e.target.value] : [])}
            >
              <option value="">All Status</option>
              {STATUS_FILTER_OPTIONS.map((status) => (
                <option key={status} value={status}>{status}</option>
              ))}
            </select>
            <select
              className="jobs-filter-select"
              value={dateFilter}
              onChange={(e) => setDateFilter(e.target.value)}
            >
              {DATE_FILTER_OPTIONS.map((option) => (
                <option key={option} value={option === 'Last update' ? '' : option}>
                  {option}
                </option>
              ))}
            </select>
            <button className="jobs-refresh-btn" onClick={handleReload} title="Refresh">
              <Refresh fontSize="small" />
            </button>
          </div>
        </div>
      </div>

      <div className="jobs-table-wrapper">
        <div className="jobs-thead">
          <div className="jobs-th jobs-th-source sortable" onClick={() => handleSort('source_path')}>
            Source <SortIcon column="source_path" />
          </div>
          <div className="jobs-th jobs-th-destination sortable" onClick={() => handleSort('destination_path')}>
            Destination <SortIcon column="destination_path" />
          </div>
          <div className="jobs-th jobs-th-status sortable" onClick={() => handleSort('status')}>
            Status <SortIcon column="status" />
          </div>
          <div className="jobs-th jobs-th-date sortable" onClick={() => handleSort('last_update')}>
            Date <SortIcon column="last_update" />
          </div>
          <div className="jobs-th jobs-th-actions">Actions</div>
        </div>

        {filteredJobs.length > 0 ? (
          <FixedSizeList
            height={height}
            width="100%"
            itemCount={filteredJobs.length}
            itemSize={56}
          >
            {Row}
          </FixedSizeList>
        ) : (
          <div className="jobs-empty">
            <Assignment className="jobs-empty-icon" />
            <p className="jobs-empty-text">No jobs found</p>
          </div>
        )}
      </div>

      {selectedJob && (
        <div className="jobs-details-card">
          <div className="jobs-details-header">
            <span className="jobs-details-title">Job Details</span>
            <button className="jobs-details-close" onClick={() => setSelectedJob(null)}>
              ×
            </button>
          </div>
          <div className="jobs-details-body">
            <div className="jobs-details-row">
              <span className="jobs-details-label">ID</span>
              <span className="jobs-details-value">{selectedJob.id}</span>
            </div>
            <div className="jobs-details-row">
              <span className="jobs-details-label">Source</span>
              <span className="jobs-details-value">{selectedJob.source_path}</span>
            </div>
            <div className="jobs-details-row">
              <span className="jobs-details-label">Destination</span>
              <span className="jobs-details-value">{selectedJob.destination_path}</span>
            </div>
            <div className="jobs-details-row">
              <span className="jobs-details-label">Status</span>
              <span className="jobs-details-value">{selectedJob.status}</span>
            </div>
            <div className="jobs-details-row">
              <span className="jobs-details-label">Phase</span>
              <span className="jobs-details-value">{selectedJob.status_phase}</span>
            </div>
            <div className="jobs-details-row">
              <span className="jobs-details-label">Message</span>
              <span className="jobs-details-value">{selectedJob.status_message}</span>
            </div>
          </div>
          <div className="jobs-details-actions">
            <button className="btn btn-secondary" onClick={() => setSelectedJob(null)}>
              Close
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default JobsPage;