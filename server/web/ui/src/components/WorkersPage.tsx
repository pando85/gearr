import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { People, Schedule, Storage, Dns } from '@mui/icons-material';
import './styles/WorkersPage.css';

interface Worker {
  name: string;
  id: string;
  queue_name: string;
  last_seen: string;
}

interface WorkersPageProps {
  token: string;
  setShowJobTable: React.Dispatch<React.SetStateAction<boolean>>;
  setErrorText: React.Dispatch<React.SetStateAction<string>>;
}

const WorkersPage: React.FC<WorkersPageProps> = ({ token, setShowJobTable, setErrorText }) => {
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    const fetchWorkers = async () => {
      try {
        setLoading(true);
        const response = await axios.get('/api/v1/workers', {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        });
        setWorkers(response.data);
      } catch (error) {
        console.error('Error fetching workers:', error);
        setShowJobTable(false);
        if (error instanceof Error) {
          setErrorText(error.message);
        }
      } finally {
        setLoading(false);
      }
    };

    fetchWorkers();
  }, [token, setShowJobTable, setErrorText]);

  const getInitials = (name: string) => {
    return name
      .split('-')
      .map((part) => part[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  };

  const formatLastSeen = (lastSeen: string) => {
    const date = new Date(lastSeen);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    return `${diffDays}d ago`;
  };

  if (loading) {
    return (
      <div className="workers-page">
        <div className="workers-header">
          <h1 className="workers-title">Workers</h1>
        </div>
        <div className="workers-loading">
          <div className="workers-spinner" />
        </div>
      </div>
    );
  }

  return (
    <div className="workers-page">
      <div className="workers-header">
        <h1 className="workers-title">Workers ({workers.length})</h1>
      </div>

      {workers.length > 0 ? (
        <div className="workers-grid">
          {workers.map((worker) => (
            <div key={worker.id} className="worker-card">
              <div className="worker-card-header">
                <div className="worker-avatar">{getInitials(worker.name)}</div>
                <div className="worker-info">
                  <div className="worker-name">{worker.name}</div>
                  <div className="worker-id">{worker.id.slice(0, 8)}...</div>
                </div>
                <div className="worker-status">
                  <span className="worker-status-dot" />
                  Online
                </div>
              </div>
              <div className="worker-card-body">
                <div className="worker-detail">
                  <span className="worker-detail-label">
                    <Dns fontSize="small" />
                    Queue
                  </span>
                  <span className="worker-detail-value">{worker.queue_name}</span>
                </div>
                <div className="worker-detail">
                  <span className="worker-detail-label">
                    <Schedule fontSize="small" />
                    Last Seen
                  </span>
                  <span className="worker-detail-value">{formatLastSeen(worker.last_seen)}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="workers-empty">
          <People className="workers-empty-icon" />
          <p className="workers-empty-text">No workers available</p>
        </div>
      )}
    </div>
  );
};

export default WorkersPage;