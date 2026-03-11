import React, { useMemo } from 'react';
import { useSelector } from 'react-redux';
import { Link } from 'react-router-dom';
import {
  Work,
  CheckCircle,
  Error,
  Schedule,
  Devices,
  TrendingUp,
  ArrowForward,
} from '@mui/icons-material';
import { RootState } from '../store';
import { Job } from '../model';
import { formatDateShort } from '../utils';

interface DashboardProps {
  token: string;
}

const Dashboard: React.FC<DashboardProps> = () => {
  const jobs: Job[] = useSelector((state: RootState) => state.jobs);

  const stats = useMemo(() => {
    const total = jobs.length;
    const completed = jobs.filter((j) => j.status === 'completed').length;
    const progressing = jobs.filter((j) => j.status === 'progressing').length;
    const queued = jobs.filter((j) => j.status === 'queued').length;
    const failed = jobs.filter((j) => j.status === 'failed').length;

    return { total, completed, progressing, queued, failed };
  }, [jobs]);

  const recentJobs = useMemo(() => {
    return [...jobs]
      .sort((a, b) => {
        const dateA = a.last_update ? new Date(a.last_update).getTime() : 0;
        const dateB = b.last_update ? new Date(b.last_update).getTime() : 0;
        return dateB - dateA;
      })
      .slice(0, 5);
  }, [jobs]);

  const getJobFileName = (path: string) => {
    return path.split('/').pop() || path;
  };

  const getRelativeTime = (date: Date) => {
    if (!date) return '';
    const now = new Date();
    const diff = now.getTime() - new Date(date).getTime();
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return 'Just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    return `${days}d ago`;
  };

  return (
    <div className="dashboard-page">
      <div className="dashboard-header">
        <h1 className="dashboard-title">Dashboard</h1>
        <p className="dashboard-subtitle">Overview of your transcoding jobs</p>
      </div>

      <div className="dashboard-stats">
        <div className="stat-card">
          <div className="stat-icon primary">
            <Work />
          </div>
          <div className="stat-content">
            <div className="stat-label">Total Jobs</div>
            <div className="stat-value">{stats.total}</div>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon success">
            <CheckCircle />
          </div>
          <div className="stat-content">
            <div className="stat-label">Completed</div>
            <div className="stat-value">{stats.completed}</div>
            {stats.total > 0 && (
              <div className="stat-change positive">
                <TrendingUp style={{ fontSize: 14 }} />
                {Math.round((stats.completed / stats.total) * 100)}%
              </div>
            )}
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon info">
            <Schedule />
          </div>
          <div className="stat-content">
            <div className="stat-label">In Progress</div>
            <div className="stat-value">{stats.progressing}</div>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon error">
            <Error />
          </div>
          <div className="stat-content">
            <div className="stat-label">Failed</div>
            <div className="stat-value">{stats.failed}</div>
          </div>
        </div>
      </div>

      <div className="dashboard-grid">
        <div className="dashboard-section">
          <div className="dashboard-section-header">
            <h2 className="dashboard-section-title">Recent Jobs</h2>
            <Link to="/jobs" className="dashboard-section-action">
              View all <ArrowForward style={{ fontSize: 12, marginLeft: 4 }} />
            </Link>
          </div>
          <div className="dashboard-section-content">
            {recentJobs.length > 0 ? (
              <div className="recent-jobs-list">
                {recentJobs.map((job) => (
                  <div key={job.id} className="recent-job-item">
                    <div className={`recent-job-status ${job.status}`} />
                    <div className="recent-job-info">
                      <div className="recent-job-name" title={job.source_path}>
                        {getJobFileName(job.source_path)}
                      </div>
                      <div className="recent-job-time">
                        {getRelativeTime(job.last_update)}
                      </div>
                    </div>
                    <span className={`badge badge-${
                      job.status === 'completed' ? 'success' :
                      job.status === 'failed' ? 'error' :
                      job.status === 'progressing' ? 'info' : 'neutral'
                    }`}>
                      {job.status}
                    </span>
                  </div>
                ))}
              </div>
            ) : (
              <div className="dashboard-empty">
                <Work className="dashboard-empty-icon" />
                <p className="dashboard-empty-text">No jobs yet</p>
              </div>
            )}
          </div>
        </div>

        <div className="dashboard-section">
          <div className="dashboard-section-header">
            <h2 className="dashboard-section-title">Queue Status</h2>
          </div>
          <div className="dashboard-section-content">
            <div className="worker-activity-list">
              <div className="worker-activity-item">
                <div className="worker-activity-icon">
                  <Schedule />
                </div>
                <div className="worker-activity-info">
                  <div className="worker-activity-name">Queued Jobs</div>
                  <div className="worker-activity-status">
                    {stats.queued} waiting to be processed
                  </div>
                </div>
              </div>
              <div className="worker-activity-item">
                <div className="worker-activity-icon">
                  <Devices />
                </div>
                <div className="worker-activity-info">
                  <div className="worker-activity-name">Active Workers</div>
                  <div className="worker-activity-status">
                    Processing {stats.progressing} jobs
                  </div>
                </div>
              </div>
              <div className="worker-activity-item">
                <div className="worker-activity-icon">
                  <CheckCircle />
                </div>
                <div className="worker-activity-info">
                  <div className="worker-activity-name">Success Rate</div>
                  <div className="worker-activity-status">
                    {stats.total > 0
                      ? Math.round(((stats.completed) / (stats.total - stats.queued - stats.progressing || 1)) * 100)
                      : 0}% of finished jobs
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;