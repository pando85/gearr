import React, { useState, useEffect } from 'react';
import axios from 'axios';
import {
  Devices,
  MemoryStick,
  Schedule,
  Link as LinkIcon,
  MoreVert,
} from '@mui/icons-material';

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
    const interval = setInterval(fetchWorkers, 30000);
    return () => clearInterval(interval);
  }, [token, setShowJobTable, setErrorText]);

  const isOnline = (lastSeen: string) => {
    if (!lastSeen) return false;
    const lastSeenDate = new Date(lastSeen);
    const now = new Date();
    const diffMinutes = (now.getTime() - lastSeenDate.getTime()) / 60000;
    return diffMinutes < 5;
  };

  const formatLastSeen = (lastSeen: string) => {
    if (!lastSeen) return 'Never';
    const date = new Date(lastSeen);
    const now = new Date();
    const diffMinutes = Math.floor((now.getTime() - date.getTime()) / 60000);
    const diffHours = Math.floor(diffMinutes / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffMinutes < 1) return 'Just now';
    if (diffMinutes < 60) return `${diffMinutes}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  };

  if (loading) {
    return (
      <div className="workers-page">
        <div className="workers-header">
          <h1 className="workers-title">
            <Devices className="workers-title-icon" />
            Workers
          </h1>
        </div>
        <div className="workers-loading">
          {[1, 2, 3].map((i) => (
            <div key={i} className="worker-skeleton">
              <div className="worker-skeleton-header">
                <div className="worker-skeleton-icon skeleton" style={{ width: 40, height: 40 }} />
                <div style={{ flex: 1 }}>
                  <div className="worker-skeleton-text skeleton" style={{ width: '60%', marginBottom: 8 }} />
                  <div className="worker-skeleton-text skeleton" style={{ width: '40%' }} />
                </div>
              </div>
              <div className="worker-skeleton-line skeleton" style={{ width: '80%' }} />
              <div className="worker-skeleton-line skeleton" style={{ width: '60%' }} />
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="workers-page">
      <div className="workers-header">
        <h1 className="workers-title">
          <Devices className="workers-title-icon" />
          Workers
          <span className="workers-count">{workers.length}</span>
        </h1>
      </div>

      {workers.length > 0 ? (
        <div className="workers-grid">
          {workers.map((worker) => {
            const online = isOnline(worker.last_seen);
            return (
              <div key={worker.id} className="worker-card">
                <div className="worker-card-header">
                  <div className="worker-info">
                    <div className="worker-icon">
                      <MemoryStick />
                    </div>
                    <div>
                      <div className="worker-name">{worker.name}</div>
                      <div className="worker-id">{worker.id.slice(0, 8)}...</div>
                    </div>
                  </div>
                  <div className={`worker-status ${online ? 'online' : 'offline'}`}>
                    <span className="worker-status-dot" />
                    {online ? 'Online' : 'Offline'}
                  </div>
                </div>

                <div className="worker-details">
                  <div className="worker-detail">
                    <div className="worker-detail-label">
                      <LinkIcon />
                      Queue
                    </div>
                    <div className="worker-detail-value">{worker.queue_name}</div>
                  </div>
                  <div className="worker-detail">
                    <div className="worker-detail-label">
                      <Schedule />
                      Last Seen
                    </div>
                    <div className="worker-detail-value">
                      {formatLastSeen(worker.last_seen)}
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        <div className="workers-empty">
          <Devices className="workers-empty-icon" />
          <h3 className="workers-empty-title">No workers found</h3>
          <p className="workers-empty-text">
            Workers will appear here when they connect to the server
          </p>
        </div>
      )}
    </div>
  );
};

export default WorkersPage;