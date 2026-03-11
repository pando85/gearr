import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { FixedSizeList } from 'react-window';
import {
  Search,
  FilterList,
  Refresh,
  MoreVert,
  Delete,
  Replay,
  Info,
  ArrowUpward,
  ArrowDownward,
  Work,
} from '@mui/icons-material';
import useWebSocket from 'react-use-websocket';
import { Job, JobUpdateNotification, JobUpdateNotificationClass } from '../model';
import { fetchJobs, deleteJob, createJob } from '../api';
import { RootState } from '../store';
import { STATUS_FILTER_OPTIONS, DATE_FILTER_OPTIONS, getDateFromFilterOption, getStatusColor } from '../utils';
import { updateJob, resetJobs } from '../actions/JobActions';
import { useToast } from './Toast';

interface JobsPageProps {
  token: string;
  setShowJobTable: React.Dispatch<React.SetStateAction<boolean>>;
  setErrorText: React.Dispatch<React.SetStateAction<string>>;
}

interface SortConfig {
  column: string;
  direction: 'asc' | 'desc';
}

const JobsPage: React.FC<JobsPageProps> = ({ token, setShowJobTable, setErrorText }) => {
  const dispatch = useDispatch();
  const toast = useToast();
  const jobs: Job[] = useSelector((state: RootState) => state.jobs);

  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<string[]>([]);
  const [dateFilter, setDateFilter] = useState('');
  const [sortConfig, setSortConfig] = useState<SortConfig>({ column: 'last_update', direction: 'desc' });
  const [selectedJob, setSelectedJob] = useState<Job | null>(null);
  const [height, setHeight] = useState(window.innerHeight);

  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
  const wsURL = `${protocol}://${window.location.hostname}:${window.location.port}/ws/job?token=${token}`;
  const { lastMessage } = useWebSocket(wsURL);

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
    const handleResize = () => setHeight(window.innerHeight);
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  const filteredAndSortedJobs = useMemo(() => {
    let result = [...jobs];

    if (statusFilter.length > 0) {
      result = result.filter((job) => statusFilter.includes(job.status));
    }

    if (dateFilter) {
      const filterDate = getDateFromFilterOption(dateFilter);
      result = result.filter((job) => job.last_update >= filterDate);
    }

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (job) =>
          job.source_path?.toLowerCase().includes(query) ||
          job.destination_path?.toLowerCase().includes(query)
      );
    }

    result.sort((a, b) => {
      const aValue = a[sortConfig.column as keyof Job];
      const bValue = b[sortConfig.column as keyof Job];

      if (sortConfig.column === 'last_update') {
        const aTime = aValue ? new Date(aValue as Date).getTime() : 0;
        const bTime = bValue ? new Date(bValue as Date).getTime() : 0;
        return sortConfig.direction === 'asc' ? aTime - bTime : bTime - aTime;
      }

      if (typeof aValue === 'string' && typeof bValue === 'string') {
        return sortConfig.direction === 'asc'
          ? aValue.localeCompare(bValue)
          : bValue.localeCompare(aValue);
      }

      return 0;
    });

    return result;
  }, [jobs, statusFilter, dateFilter, searchQuery, sortConfig]);

  const handleSort = useCallback((column: string) => {
    setSortConfig((prev) => ({
      column,
      direction: prev.column === column && prev.direction === 'asc' ? 'desc' : 'asc',
    }));
  }, []);

  const handleDeleteJob = useCallback(
    async (jobId: string) => {
      try {
        await dispatch(deleteJob(token, setShowJobTable, setErrorText, jobId) as any);
        toast.success('Job deleted', `Job ${jobId} has been removed`);
      } catch (err) {
        toast.error('Delete failed', 'Could not delete the job');
      }
    },
    [dispatch, token, setShowJobTable, setErrorText, toast]
  );

  const handleRecreateJob = useCallback(
    async (job: Job) => {
      try {
        await dispatch(deleteJob(token, setShowJobTable, setErrorText, job.id) as any);
        await dispatch(createJob(token, setShowJobTable, setErrorText, job.source_path) as any);
        toast.success('Job recreated', `Recreated job from ${job.source_path}`);
      } catch (err) {
        toast.error('Recreate failed', 'Could not recreate the job');
      }
    },
    [dispatch, token, setShowJobTable, setErrorText, toast]
  );

  const handleRefresh = useCallback(() => {
    dispatch(resetJobs() as any);
    dispatch(fetchJobs(token, setShowJobTable, setErrorText) as any);
    toast.info('Refreshing...', 'Loading latest jobs');
  }, [dispatch, token, setShowJobTable, setErrorText, toast]);

  const getFileName = (path: string) => path.split('/').pop() || path;

  const formatDate = (date: Date) => {
    if (!date) return '';
    return new Intl.DateTimeFormat(navigator.language, {
      dateStyle: 'short',
      timeStyle: 'short',
    }).format(new Date(date));
  };

  const getProgress = (job: Job) => {
    if (job.status !== 'progressing' || job.status_phase !== 'FFMPEG') return null;
    try {
      const msg = JSON.parse(job.status_message);
      return msg.progress !== undefined ? parseFloat(msg.progress) : 0;
    } catch {
      return 0;
    }
  };

  const SortIcon = ({ column }: { column: string }) => (
    <span className={`sort-icon ${sortConfig.column === column ? 'active' : ''}`}>
      {sortConfig.column === column ? (
        sortConfig.direction === 'asc' ? (
          <ArrowUpward fontSize="small" />
        ) : (
          <ArrowDownward fontSize="small" />
        )
      ) : (
        <ArrowUpward fontSize="small" />
      )}
    </span>
  );

  const Row = ({ index, style }: { index: number; style: React.CSSProperties }) => {
    const job = filteredAndSortedJobs[index];
    if (!job) return null;

    const progress = getProgress(job);

    return (
      <div style={style}>
        <tr className="job-row">
          <td>
            <div className="job-path" title={job.source_path}>
              {getFileName(job.source_path)}
            </div>
          </td>
          <td className="d-none d-md-table-cell">
            <div className="job-path" title={job.destination_path}>
              {getFileName(job.destination_path)}
            </div>
          </td>
          <td>
            {progress !== null ? (
              <div className="job-progress">
                <div className="job-progress-bar">
                  <div className="job-progress-fill" style={{ width: `${progress}%` }} />
                </div>
                <div className="job-progress-text">{progress.toFixed(1)}%</div>
              </div>
            ) : (
              <span className={`job-status job-status-${job.status}`}>
                {job.status_phase !== 'Job' ? job.status_phase.toLowerCase() : job.status}
              </span>
            )}
          </td>
          <td>
            <div className="job-date">{formatDate(job.last_update)}</div>
          </td>
          <td>
            <div className="job-actions">
              <button
                className="job-action-btn"
                onClick={() => setSelectedJob(job)}
                title="Details"
              >
                <Info fontSize="small" />
              </button>
              <button
                className="job-action-btn danger"
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
          </td>
        </tr>
      </div>
    );
  };

  return (
    <div className="jobs-page">
      <div className="jobs-header">
        <h1 className="jobs-title">
          <Work className="jobs-title-icon" />
          Jobs
          <span className="jobs-count">{filteredAndSortedJobs.length}</span>
        </h1>
      </div>

      <div className="jobs-toolbar">
        <div className="search-box">
          <Search className="search-icon" />
          <input
            className="search-input"
            type="text"
            placeholder="Search jobs..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>

        <div className="filter-group">
          <select
            className="filter-select"
            value={statusFilter.length > 0 ? statusFilter[0] : ''}
            onChange={(e) =>
              setStatusFilter(e.target.value ? [e.target.value] : [])
            }
          >
            <option value="">All Status</option>
            {STATUS_FILTER_OPTIONS.map((status) => (
              <option key={status} value={status}>
                {status}
              </option>
            ))}
          </select>

          <select
            className="filter-select"
            value={dateFilter}
            onChange={(e) => setDateFilter(e.target.value)}
          >
            {DATE_FILTER_OPTIONS.map((option) => (
              <option key={option} value={option === 'Last update' ? '' : option}>
                {option}
              </option>
            ))}
          </select>
        </div>

        <div className="jobs-actions">
          <button className="btn btn-ghost btn-icon" onClick={handleRefresh} title="Refresh">
            <Refresh />
          </button>
        </div>
      </div>

      <div className="jobs-table-container">
        {filteredAndSortedJobs.length > 0 ? (
          <table className="jobs-table">
            <thead>
              <tr>
                <th className="sortable" onClick={() => handleSort('source_path')}>
                  Source
                  <SortIcon column="source_path" />
                </th>
                <th className="sortable d-none d-md-table-cell" onClick={() => handleSort('destination_path')}>
                  Destination
                  <SortIcon column="destination_path" />
                </th>
                <th className="sortable" onClick={() => handleSort('status')}>
                  Status
                  <SortIcon column="status" />
                </th>
                <th className="sortable" onClick={() => handleSort('last_update')}>
                  Updated
                  <SortIcon column="last_update" />
                </th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              <FixedSizeList
                height={height - 300}
                width="100%"
                itemCount={filteredAndSortedJobs.length}
                itemSize={63}
                overscanCount={10}
              >
                {Row}
              </FixedSizeList>
            </tbody>
          </table>
        ) : (
          <div className="jobs-empty">
            <Work className="jobs-empty-icon" />
            <h3 className="jobs-empty-title">No jobs found</h3>
            <p className="jobs-empty-text">
              {searchQuery || statusFilter.length > 0 || dateFilter
                ? 'Try adjusting your filters'
                : 'Jobs will appear here when they are created'}
            </p>
          </div>
        )}
      </div>

      {selectedJob && (
        <div className="job-details-modal" onClick={() => setSelectedJob(null)}>
          <div className="job-details-content" onClick={(e) => e.stopPropagation()}>
            <div className="job-details-header">
              <h3 className="job-details-title">Job Details</h3>
              <button className="job-details-close" onClick={() => setSelectedJob(null)}>
                ×
              </button>
            </div>
            <div className="job-details-body">
              <div className="job-details-row">
                <span className="job-details-label">ID</span>
                <span className="job-details-value">{selectedJob.id}</span>
              </div>
              <div className="job-details-row">
                <span className="job-details-label">Source Path</span>
                <span className="job-details-value">{selectedJob.source_path}</span>
              </div>
              <div className="job-details-row">
                <span className="job-details-label">Destination Path</span>
                <span className="job-details-value">{selectedJob.destination_path}</span>
              </div>
              <div className="job-details-row">
                <span className="job-details-label">Status</span>
                <span className={`badge badge-${
                  selectedJob.status === 'completed' ? 'success' :
                  selectedJob.status === 'failed' ? 'error' :
                  selectedJob.status === 'progressing' ? 'info' : 'neutral'
                }`}>
                  {selectedJob.status}
                </span>
              </div>
              <div className="job-details-row">
                <span className="job-details-label">Phase</span>
                <span className="job-details-value">{selectedJob.status_phase}</span>
              </div>
              <div className="job-details-row">
                <span className="job-details-label">Message</span>
                <span className="job-details-value" style={{ wordBreak: 'break-all' }}>
                  {selectedJob.status_message || 'No message'}
                </span>
              </div>
              <div className="job-details-row">
                <span className="job-details-label">Last Update</span>
                <span className="job-details-value">{formatDate(selectedJob.last_update)}</span>
              </div>
            </div>
            <div className="job-details-footer">
              <button className="btn btn-secondary" onClick={() => setSelectedJob(null)}>
                Close
              </button>
              <button
                className="btn btn-danger"
                onClick={() => {
                  handleDeleteJob(selectedJob.id);
                  setSelectedJob(null);
                }}
              >
                Delete Job
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default JobsPage;