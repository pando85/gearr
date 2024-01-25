import React, { useState, useEffect } from 'react';
import {
  Button,
  CircularProgress,
  Checkbox,
  FormControl,
  InputLabel,
  Menu,
  MenuItem,
  ListItemIcon,
  Select,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import {
  Cached,
  CalendarMonth,
  Delete,
  Error,
  Feed,
  MoreVert,
  QuestionMark,
  Replay,
  Search,
  Task,
  VideoSettings,
} from '@mui/icons-material';

import './JobTable.css';
import { useDispatch, useSelector } from 'react-redux';
import { Job } from './model';
import { fetchJobs, deleteJob, createJob } from './api';
import { RootState } from './store';
import { resetJobs } from './actions/JobActions';

interface JobTableProps {
  token: string;
  setShowJobTable: React.Dispatch<React.SetStateAction<boolean>>;
}

const formatDate = (date: Date, options: Intl.DateTimeFormatOptions): string => {
  if (date == null) {
    return '';
  }

  try {
    return new Intl.DateTimeFormat(navigator.language, options).format(date);
  } catch (error) {
    console.error('Error formatting date:', error);
    return '';
  }
};

const formatDateDetailed = (date: Date): string => {
  const options: Intl.DateTimeFormatOptions = {
    timeStyle: 'long',
  };
  return formatDate(date, options);
};

const formatDateShort = (date: Date): string => {
  const options: Intl.DateTimeFormatOptions = {
    dateStyle: 'short',
  };
  const formatedDate = formatDate(date, options)
  return formatedDate;
};

const getDateFromFilterOption = (filterOption: string) => {
  const currentDate = new Date();

  switch (filterOption) {
    case 'Last 30 minutes':
      return new Date(currentDate.getTime() - 30 * 60 * 1000);

    case 'Last 3 hours':
      return new Date(currentDate.getTime() - 3 * 60 * 60 * 1000);

    case 'Last 6 hours':
      return new Date(currentDate.getTime() - 6 * 60 * 60 * 1000);

    case 'Last 24 hours':
      return new Date(currentDate.getTime() - 24 * 60 * 60 * 1000);

    case 'Last 2 days':
      return new Date(currentDate.getTime() - 2 * 24 * 60 * 60 * 1000);

    case 'Last 7 days':
      return new Date(currentDate.getTime() - 7 * 24 * 60 * 60 * 1000);

    case 'Last 30 days':
      return new Date(currentDate.getTime() - 30 * 24 * 60 * 60 * 1000);

    default:
      return new Date(0);
  }
}

const renderPath = (isSmallScreen: boolean, path: string) => {
  if (isSmallScreen) {
    const shortPath = path.split('/').pop();
    return shortPath ? shortPath : path;
  } else {
    return path;
  }
};

const JobTable: React.FC<JobTableProps> = ({ token, setShowJobTable }) => {
  const [filteredJobs, setFilteredJobs] = useState<Job[]>([]);
  const [selectedJob, setSelectedJob] = useState<Job | null>(null);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null); // For menu anchor
  const [nameFilter, setNameFilter] = useState<string>(''); // State for name filter
  const [selectedStatusFilter, setSelectedStatus] = useState<string | string[]>([]);
  const [selectedDateFilter, setSelectedDateFilter] = useState<string>('');
  const [detailsMenuAnchor, setDetailsMenuAnchor] = useState<null | HTMLElement>(null);

  const [isSmallScreen, setIsSmallScreen] = useState(window.innerWidth <= 768);

  const dispatch = useDispatch();
  const jobs: Job[] = useSelector((state: RootState) => state.jobs);
  // TODO: show sign in on error
  // setShowJobTable(false)
  const error = useSelector((state: RootState) => state.error);
  const loading = useSelector((state: RootState) => state.loading);

  useEffect(() => {
    dispatch(fetchJobs(token) as any);
  }, [dispatch]);

  const handleDeleteJob = (jobId: string) => {
    dispatch(deleteJob(token, jobId) as any);
  };

  const handleCreateJob = (path: string) => {
    dispatch(createJob(token, path) as any);
  };

  const handleReload = () => {
    dispatch(resetJobs() as any);
    dispatch(fetchJobs(token) as any);
  };

  useEffect(() => {
    const handleResize = () => {
      setIsSmallScreen(window.innerWidth <= 768);
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
    setFilteredJobs(filteredJobs);
  });

  const reload = () => {
    handleReload();
  };

  const handleClick = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const handleMenuOptionClick = (job: Job | null, option: string) => {
    if (job !== null) {
      if (['delete', 'recreate'].includes(option)) {
        handleDeleteJob(job.id);
      };
      handleClose();
      if (option === 'recreate') {
        handleCreateJob(job.source_path);
      }
    }
  };

  const handleRowClick = (jobId: string) => {
    setSelectedJob(jobs.find((job) => job.id === jobId) || null);
  };

  const handleNameFilterChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setNameFilter(event.target.value);
  };

  const handleDetailedViewClick = (event: React.MouseEvent<HTMLElement>) => {
    event.stopPropagation();
    setDetailsMenuAnchor(event.currentTarget);
  };

  const statusFilterOptions = [
    'progressing',
    'queued',
    'completed',
    'failed',
  ];

  const dateFilterOptions = [
    'Last update',
    'Last 30 minutes',
    'Last 3 hours',
    'Last 6 hours',
    'Last 24 hours',
    'Last 2 days',
    'Last 7 days',
    'Last 30 days',
  ];

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'completed':
        return 'green';
      case 'failed':
        return 'red';
      default:
        return 'grey';
    }
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
                  <div
                    className="progress-bar"
                    style={{ width: `${progress}%` }}
                  />
                </div>
              );
            }
          } catch (error) {
            return (
              <div className="error-icon" title={job.status_message}>
                <Error />
              </div>
            );
          }
        })()
      ) : (
        <span />
      );
    } else if (job.status === 'failed') {
      return (
        <div className="error-icon" title={job.status_message}>
          <Error />
        </div>
      );
    } else {
      return (
        <Button
          variant="contained"
          style={{
            backgroundColor: getStatusColor(job.status),
          }}
        >
          {job.status}
        </Button>
      );
    }
  };

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
              {statusFilterOptions.map((option) => (
                <MenuItem value={option}>
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
            {dateFilterOptions.map((option) => (
              <MenuItem key={option} value={option}>
                {option}
              </MenuItem>
            ))}
          </Select>
          <Button className="float-right" style={{ marginLeft: 'auto' }} onClick={reload}>
            <Cached />
          </Button>
        </div>
      </div>
      <div className="flex-top-bar padder mb-4 mb-sm-0" />
      <div className="job-list">
        <Table className="job-table">
          <TableHead>
            <TableRow>
              <TableCell>
                <span title="Source">
                  <Task />
                </span>
              </TableCell>
              <TableCell className="d-none d-sm-table-cell">
                <span title="Destination">
                  <VideoSettings />
                </span>
              </TableCell>
              <TableCell>
                <span title="Status">
                  <QuestionMark />
                </span>
              </TableCell>
              <TableCell>
                <span title="LastUpdate">
                  <CalendarMonth />
                </span>
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredJobs.map((job) => (
              <TableRow
                key={job.id}
                onClick={() => handleRowClick(job.id)}
                className="table-row"
              >
                <TableCell>
                  {renderPath(isSmallScreen, job.source_path)}
                </TableCell>
                <TableCell className="d-none d-sm-table-cell">
                  {job.destination_path}
                </TableCell>
                <TableCell className="d-none d-sm-table-cell">
                  {renderStatusCellContent(job)}
                </TableCell>
                <TableCell title={formatDateDetailed(job.last_update)}>
                  <div className="row-menu">
                    {formatDateShort(job.last_update)}
                    <Button
                      className="simple-menu"
                      aria-controls="simple-menu"
                      aria-haspopup="true"
                      onClick={handleClick}
                      size="small"
                    >
                      <MoreVert />
                    </Button>
                    <Menu
                      id="simple-menu"
                      className="simple-menu"
                      anchorEl={anchorEl}
                      keepMounted
                      open={Boolean(anchorEl)}
                      onClose={handleClose}
                    >
                      <MenuItem title="Details" onClick={(event) => handleDetailedViewClick(event)}>
                        <Feed />
                      </MenuItem>
                      <MenuItem title="Delete" onClick={() => handleMenuOptionClick(selectedJob, 'delete')}>
                        <Delete />
                      </MenuItem>
                      <MenuItem title="Recreate" onClick={() => handleMenuOptionClick(selectedJob, 'recreate')}>
                        <Replay />
                      </MenuItem>
                    </Menu>
                    <Menu
                      id="details-menu"
                      className="details-menu"
                      anchorEl={detailsMenuAnchor}
                      keepMounted
                      open={Boolean(detailsMenuAnchor)}
                      onClose={() => setDetailsMenuAnchor(null)}
                    >
                      {selectedJob && [
                        <MenuItem key="job-details">
                          <Typography variant="h5" gutterBottom>
                            Job Details
                          </Typography>
                        </MenuItem>,
                        <MenuItem key="job-id">
                          <Typography>ID: {selectedJob.id}</Typography>
                        </MenuItem>,
                        <MenuItem key="job-source">
                          <Typography>Source: {selectedJob.source_path}</Typography>
                        </MenuItem>,
                        <MenuItem key="job-destination">
                          <Typography>Destination: {selectedJob.destination_path}</Typography>
                        </MenuItem>,
                        <MenuItem key="job-status">
                          <Typography>Status: {selectedJob.status}</Typography>
                        </MenuItem>,
                        <MenuItem key="job-message">
                          <Typography>Message: {selectedJob.status_message}</Typography>
                        </MenuItem>,
                      ]}
                    </Menu>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>

        {loading && <CircularProgress />}
      </div>
    </div>
  );
};

export default JobTable;
