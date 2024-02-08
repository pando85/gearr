import React, { useEffect, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { FixedSizeList } from 'react-window';
import { Button, Card, Dropdown } from 'react-bootstrap';
import {
  ArrowDownward,
  ArrowUpward,
  Cached,
  CalendarMonth,
  Delete,
  Error,
  Feed,
  QuestionMark,
  Replay,
  Search,
  Task,
  VideoSettings
} from '@mui/icons-material';
import {
  Checkbox,
  FormControl,
  MenuItem,
  InputLabel,
  ListItemIcon,
  Select
} from '@mui/material';
import useWebSocket from 'react-use-websocket';
import { Job, JobUpdateNotification, JobUpdateNotificationClass } from './model';
import { fetchJobs, deleteJob, createJob } from './api';
import { RootState } from './store';
import { STATUS_FILTER_OPTIONS, DATE_FILTER_OPTIONS, formatDateShort, formatDateDetailed, getDateFromFilterOption, getStatusColor, renderPath, sortJobs } from './utils';
import { updateJob, resetJobs } from './actions/JobActions';
import './JobTable.css';

interface JobTableProps {
  token: string;
  setShowJobTable: React.Dispatch<React.SetStateAction<boolean>>;
  setErrorText: React.Dispatch<React.SetStateAction<string>>;
}

const JobTable: React.FC<JobTableProps> = ({ token, setShowJobTable, setErrorText }) => {
  // States
  const [filteredJobs, setFilteredJobs] = useState<Job[]>([]);
  const [nameFilter, setNameFilter] = useState<string>('');
  const [selectedStatusFilter, setSelectedStatus] = useState<string | string[]>([]);
  const [selectedDateFilter, setSelectedDateFilter] = useState<string>('');
  const [selectedJobIndex, setSelectedJobIndex] = useState<number | null>(null);
  const [isSmallScreen, setIsSmallScreen] = useState(window.innerWidth <= 768);
  const [height, setHeight] = useState(window.innerHeight);
  const [sortColumn, setSortColumn] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');

  // Redux
  const dispatch = useDispatch();
  const jobs: Job[] = useSelector((state: RootState) => state.jobs);

  // WebSocket
  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  const wsURL = `${protocol}://${window.location.hostname}:${window.location.port}/ws/job?token=${token}`;
  const { lastMessage } = useWebSocket(wsURL);

  // Handlers
  const toggleSortDirection = () => {
    setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
  };

  const handleSort = (column: string) => {
    if (sortColumn === column) {
      toggleSortDirection();
    } else {
      setSortColumn(column);
      setSortDirection('asc');
    }
  };

  const handleDeleteJob = async (jobId: string) => {
    await dispatch(deleteJob(token, setShowJobTable, setErrorText, jobId) as any);
  };

  const handleCreateJob = async (path: string) => {
    await dispatch(createJob(token, setShowJobTable, setErrorText, path) as any);
  };

  const handleReload = () => {
    dispatch(resetJobs() as any);
    dispatch(fetchJobs(token, setShowJobTable, setErrorText) as any);
  };

  const handleMenuOptionClick = async (job: Job | null, option: string) => {
    if (job !== null) {
      if (['delete', 'recreate'].includes(option)) {
        await handleDeleteJob(job.id);
      };
      if (option === 'recreate') {
        await handleCreateJob(job.source_path);
      }
    }
  };

  const handleNameFilterChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setNameFilter(event.target.value);
  };

  const handleButtonClick = (index: number) => {
    setSelectedJobIndex(index);
  };

  const handleDropdownItemClick = () => {
    setSelectedJobIndex(null);
  };

  const handleDropdownClick = (e: React.MouseEvent<HTMLElement>, job: Job) => {
    e.stopPropagation();
  };

  const renderStatusCellContent = (job: Job) => {
    if (job.status === 'progressing') {
      return job.status_message ? (
        (() => {
          try {
            const messageObj = JSON.parse(job.status_message);
            if (messageObj.progress !== undefined) {
              const progress = parseFloat(messageObj.progress);
              return (
                <div className="progress" title={`${progress.toFixed(2)}%`}>
                  <div className="progress-bar" style={{ width: `${progress}%` }} />
                </div>
              );
            }
          } catch (_) {
          }
        })()
      ) : (
        <span />
      );
    } else if (job.status === 'failed') {
      return (
        <div title={job.status_message}>
          <Error className="error-icon" />
        </div>
      );
    } else {
      return (
        <Button
          variant="contained"
          style={{ backgroundColor: getStatusColor(job.status) }}
        >
          {job.status}
        </Button>
      );
    }
  };

  // Effects
  useEffect(() => {
    if (lastMessage !== null) {
      const JobUpdateNotification: JobUpdateNotification = new JobUpdateNotificationClass(JSON.parse(lastMessage.data));
      dispatch(updateJob(JobUpdateNotification) as any);
    }
  }, [dispatch, lastMessage]);

  useEffect(() => {
    dispatch(fetchJobs(token, setShowJobTable, setErrorText) as any);
  }, [dispatch, token, setShowJobTable, setErrorText]);

  useEffect(() => {
    const handleResize = () => {
      setIsSmallScreen(window.innerWidth <= 768);
      setHeight(window.innerHeight);
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  useEffect(() => {
    const statusFilteredJobs = selectedStatusFilter.length > 0
      ? jobs.filter((job) => selectedStatusFilter.includes(job.status))
      : jobs;

    const dateFilteredJobs = selectedDateFilter ? statusFilteredJobs.filter(
      (job) => job.last_update >= getDateFromFilterOption(selectedDateFilter))
      : statusFilteredJobs

    const filteredJobs = nameFilter
      ? dateFilteredJobs.filter((job) => job.source_path ? job.source_path.toLowerCase().includes(nameFilter.toLowerCase()) : false)
      : dateFilteredJobs;
    setFilteredJobs(sortJobs(sortColumn, sortDirection, filteredJobs));
  }, [selectedStatusFilter, jobs, selectedDateFilter, nameFilter, sortDirection, sortColumn]);

  // Components
  const Row = ({ index, style }: { index: number, style: React.CSSProperties }) => {
    const job = filteredJobs[index];
    if (!job) {
      return null;
    }
    return (
      <div className="tr row" style={{ ...style }}>
        <div className="td col">{renderPath(isSmallScreen, job.source_path)}</div>
        <div className="td col d-none d-sm-flex">{renderPath(false, job.destination_path)}</div>
        {selectedJobIndex === index && (
          <Card style={{ width: '18rem', position: 'absolute', top: '50px', right: '20px' }}>
            <Card.Body>
              <Card.Title>Job Details</Card.Title>
              <Card.Text>
                <p>ID: {job.id}</p>
                <p>Source: {job.source_path}</p>
                <p>Destination: {job.destination_path}</p>
                <p>Status: {job.status}</p>
                <p>Message: {job.status_message}</p>
              </Card.Text>
              <Button variant="secondary" onClick={() => handleDropdownItemClick()}>Close</Button>
            </Card.Body>
          </Card>
        )}
        <div className="td col row-status" style={{ wordBreak: "keep-all" }}>{renderStatusCellContent(job)}</div>
        <div className="td col" style={{ wordBreak: "keep-all" }} title={formatDateDetailed(job.last_update)}>
          <div className="row-menu">
            <div className="row-date">
              {formatDateShort(job.last_update)}
            </div>
            <Dropdown onClick={(event) => handleDropdownClick(event, job)}>
              <Dropdown.Toggle variant="link" className="buttons-menu" size="sm" id={`dropdown-basic-${index}`} />
              <Dropdown.Menu>
                <Dropdown.Item title="Details" onClick={() => handleButtonClick(index)}>
                  <Feed />
                </Dropdown.Item>
                <Dropdown.Item title="Delete" onClick={() => handleMenuOptionClick(job, 'delete')}>
                  <Delete />
                </Dropdown.Item>
                <Dropdown.Item title="Recreate" onClick={() => handleMenuOptionClick(job, 'recreate')}>
                  <Replay />
                </Dropdown.Item>
              </Dropdown.Menu>
            </Dropdown>
          </div>
        </div>
      </div >
    );
  };

  interface FixScrollBottomProps {
    style: React.CSSProperties;
    children?: React.ReactNode;
  }

  const FixScrollBottom: React.FC<FixScrollBottomProps> = ({ style, children }) => {
    return <div style={{ ...style, marginTop: '189px' }}>{children}</div>;
  };

  const ArrowIcon = ({ active, direction }: { active: boolean; direction: 'asc' | 'desc' }) => (
    <span>
      {active && (
        <span>{direction === 'asc' ? <ArrowUpward /> : <ArrowDownward />}</span>
      )}
    </span>
  );

  // JSX
  return (
    <div className="content-wrapper">
      <div className="row flex-top-bar">
        <div className="actions">
          <div className="select select search-wrapper">
            <div className="job-list search">
              <Search />
              <input
                className="search-input"
                type="text"
                placeholder="Search jobs..."
                value={nameFilter}
                onChange={handleNameFilterChange}
              />
            </div>
          </div>
        </div>
        <div className="tools">
          <FormControl>
            <InputLabel id="filter-status-select-label">Status</InputLabel>
            <Select
              multiple
              labelId="filter-status-select-label"
              value={selectedStatusFilter}
              onChange={(event) => setSelectedStatus(event.target.value)}
              displayEmpty
              renderValue={(selected) => Array.isArray(selected) ? selected.join(', ') : selected}
            >
              {STATUS_FILTER_OPTIONS.map((option) => (
                <MenuItem value={option} key={option}>
                  <ListItemIcon>
                    <Checkbox checked={selectedStatusFilter.includes(option)} />
                  </ListItemIcon>
                  {option}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <Select
            value={selectedDateFilter ? selectedDateFilter : 'Last update'}
            onChange={(event) => setSelectedDateFilter(event.target.value)}
            displayEmpty
          >
            {DATE_FILTER_OPTIONS.map((option) => (
              <MenuItem key={option} value={option}>
                {option}
              </MenuItem>
            ))}
          </Select>
          <Button variant="link" className="float-right" style={{ marginLeft: 'auto' }} onClick={handleReload}>
            <Cached />
          </Button>
        </div>
      </div>
      <div className="job-table">
        <div className="thead">
          <div className="tr row">
            <div className="th col" onClick={() => handleSort('source_path')}>
              <span title="Source">
                <Task />
                {sortColumn === 'source_path' && (
                  <span>{sortDirection === 'asc' ? <ArrowUpward /> : <ArrowDownward />}</span>
                )}
              </span>
            </div>
            <div className="th col d-none d-sm-flex" onClick={() => handleSort('destination_path')}>
              <span title="Destination">
                <VideoSettings />
                <ArrowIcon active={sortColumn === 'destination_path'} direction={sortDirection} />
              </span>
            </div>
            <div className="th col" onClick={() => handleSort('status')}>
              <span title="Status">
                <QuestionMark />
                <ArrowIcon active={sortColumn === 'status'} direction={sortDirection} />
              </span>
            </div>
            <div className="th col" onClick={() => handleSort('last_update')}>
              <span title="LastUpdate">
                <CalendarMonth />
                <ArrowIcon active={sortColumn === 'last_update'} direction={sortDirection} />
              </span>
            </div>
          </div>
        </div>
        <FixedSizeList
          height={height}
          width="100%"
          innerElementType={FixScrollBottom}
          itemCount={filteredJobs.length}
          overscanCount={20}
          itemSize={63}
        >
          {Row}
        </FixedSizeList>
      </div>
    </div >
  );
};

export default JobTable;
