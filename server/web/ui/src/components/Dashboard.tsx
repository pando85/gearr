import React from 'react';
import { Link } from 'react-router-dom';
import { 
  Assignment, 
  HourglassEmpty, 
  CheckCircle, 
  Error, 
  Schedule,
  ArrowForward 
} from '@mui/icons-material';
import { Job } from '../model';
import { formatDateShort, getStatusColor } from '../utils';
import './styles/Dashboard.css';

interface DashboardProps {
  jobs: Job[];
}

const Dashboard: React.FC<DashboardProps> = ({ jobs }) => {
  const stats = {
    total: jobs.length,
    progressing: jobs.filter((j) => j.status === 'progressing').length,
    completed: jobs.filter((j) => j.status === 'completed').length,
    failed: jobs.filter((j) => j.status === 'failed').length,
    queued: jobs.filter((j) => j.status === 'queued').length,
  };

  const recentJobs = [...jobs]
    .sort((a, b) => new Date(b.last_update).getTime() - new Date(a.last_update).getTime())
    .slice(0, 5);

  const statCards = [
    { key: 'total', label: 'Total Jobs', value: stats.total, icon: <Assignment /> },
    { key: 'progressing', label: 'In Progress', value: stats.progressing, icon: <HourglassEmpty /> },
    { key: 'completed', label: 'Completed', value: stats.completed, icon: <CheckCircle /> },
    { key: 'failed', label: 'Failed', value: stats.failed, icon: <Error /> },
    { key: 'queued', label: 'Queued', value: stats.queued, icon: <Schedule /> },
  ];

  const getStatusBadge = (status: string) => {
    const statusColors: Record<string, string> = {
      completed: 'badge-success',
      failed: 'badge-error',
      progressing: 'badge-info',
      queued: 'badge-neutral',
    };
    return statusColors[status] || 'badge-neutral';
  };

  return (
    <div className="dashboard">
      <div className="dashboard-header">
        <h1 className="dashboard-title">Dashboard</h1>
        <p className="dashboard-subtitle">Overview of your video encoding jobs</p>
      </div>

      <div className="dashboard-grid">
        {statCards.map((stat) => (
          <div key={stat.key} className="stat-card">
            <div className="stat-card-header">
              <div className={`stat-card-icon ${stat.key}`}>
                {stat.icon}
              </div>
            </div>
            <div className="stat-card-value">{stat.value}</div>
            <div className="stat-card-label">{stat.label}</div>
          </div>
        ))}
      </div>

      <div className="recent-section">
        <div className="recent-header">
          <h2 className="recent-title">Recent Jobs</h2>
          <Link to="/jobs" className="recent-link">
            View all
            <ArrowForward fontSize="small" />
          </Link>
        </div>

        {recentJobs.length > 0 ? (
          <ul className="recent-list">
            {recentJobs.map((job) => {
              const fileName = job.source_path.split('/').pop() || job.source_path;
              return (
                <li key={job.id} className="recent-item">
                  <div className="recent-item-content">
                    <div className="recent-item-name">{fileName}</div>
                    <div className="recent-item-meta">
                      {formatDateShort(job.last_update)}
                    </div>
                  </div>
                  <div className="recent-item-status">
                    <span className={`badge ${getStatusBadge(job.status)}`}>
                      {job.status}
                    </span>
                  </div>
                </li>
              );
            })}
          </ul>
        ) : (
          <div className="dashboard-empty">
            <Assignment className="dashboard-empty-icon" />
            <p className="dashboard-empty-text">No jobs yet</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default Dashboard;