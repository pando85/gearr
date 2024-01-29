import React, { useContext, useEffect, useRef, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { Job } from './model';
import { fetchJobs, deleteJob, createJob } from './api';
import { RootState } from './store';
import { resetJobs } from './actions/JobActions';

import { FixedSizeList, FixedSizeListProps } from 'react-window';
import {
  Checkbox,
  FormControl as FormControlMui,
  Menu,
  MenuItem,
  InputLabel,
  ListItemIcon,
  Select,
  Typography,
} from '@mui/material';

import { Button, Table } from 'react-bootstrap';
import { Cached, CalendarMonth, Delete, Error, Feed, MoreVert, QuestionMark, Replay, Search, Task, VideoSettings } from '@mui/icons-material';

import './JobTable.css';

interface JobTableProps {
  token: string;
  setShowJobTable: React.Dispatch<React.SetStateAction<boolean>>;
  setErrorText: React.Dispatch<React.SetStateAction<string>>;
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

const JobTable: React.FC<JobTableProps> = ({ token, setShowJobTable, setErrorText }) => {
  const [filteredJobs, setFilteredJobs] = useState<Job[]>([]);
  const [selectedJob, setSelectedJob] = useState<Job | null>(null);
  const [buttonsMenu, setButtonsMenu] = useState<null | HTMLElement>(null); // For menu anchor
  const [nameFilter, setNameFilter] = useState<string>(''); // State for name filter
  const [selectedStatusFilter, setSelectedStatus] = useState<string | string[]>([]);
  const [selectedDateFilter, setSelectedDateFilter] = useState<string>('');
  const [detailsMenuAnchor, setDetailsMenuAnchor] = useState<null | HTMLElement>(null);

  const [isSmallScreen, setIsSmallScreen] = useState(window.innerWidth <= 768);
  const [height, setHeight] = useState(window.innerHeight);

  const dispatch = useDispatch();
  const jobs: Job[] = useSelector((state: RootState) => state.jobs);

  useEffect(() => {
    dispatch(fetchJobs(token, setShowJobTable, setErrorText) as any);
  }, [dispatch, token, setShowJobTable, setErrorText]);

  const handleDeleteJob = (jobId: string) => {
    dispatch(deleteJob(token, setShowJobTable, setErrorText, jobId) as any);
  };

  const handleCreateJob = (path: string) => {
    dispatch(createJob(token, setShowJobTable, setErrorText, path) as any);
  };

  const handleReload = () => {
    dispatch(resetJobs() as any);
    dispatch(fetchJobs(token, setShowJobTable, setErrorText) as any);
  };

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
    setFilteredJobs(filteredJobs);
  }, [selectedStatusFilter, jobs, selectedDateFilter, nameFilter]);

  const reload = () => {
    handleReload();
  };

  const handleClick = (event: React.MouseEvent<HTMLElement>) => {
    setButtonsMenu(event.currentTarget);
  };

  const handleClose = () => {
    setButtonsMenu(null);
  };

  const handleCloseDetailsMenu = () => {
    setDetailsMenuAnchor(null);
  }

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

  const handleRowClick = (job: Job) => {
    setSelectedJob(job);
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

  const VirtualTableContext = React.createContext<{
    top: number
    setTop: (top: number) => void
    header: React.ReactNode
  }>({
    top: 0,
    setTop: (_: number) => { },
    header: <></>,
  });

  function VirtualTable({
    row,
    header,
    ...rest
  }: {
    header?: React.ReactNode
    row: FixedSizeListProps['children']
  } & Omit<FixedSizeListProps, 'children' | 'innerElementType'>) {
    const listRef = useRef<FixedSizeList | null>()
    const [top, setTop] = useState(0)

    return (
      <VirtualTableContext.Provider value={{ top, setTop, header }}>
        <FixedSizeList
          {...rest}
          innerElementType={Inner}
          onItemsRendered={props => {
            const style =
              listRef.current &&
              // @ts-ignore private method access
              listRef.current._getItemStyle(props.overscanStartIndex)
            setTop((style && style.top) || 0)

            // Call the original callback
            rest.onItemsRendered && rest.onItemsRendered(props)
          }}
          ref={el => (listRef.current = el)}
        >
          {row}
        </FixedSizeList>
      </VirtualTableContext.Provider>
    );
  };

  const Inner = React.forwardRef<HTMLDivElement, React.HTMLProps<HTMLDivElement>>(
    function Inner({ children, ...rest }, ref) {
      const { header, top } = useContext(VirtualTableContext)
      return (
        <div {...rest} ref={ref}>
          <Table striped hover responsive style={{ top, position: 'absolute', width: '100%', height: '100%' }}>
            {header}
            <tbody>{children}</tbody>
          </Table>
        </div>
      )
    }
  );

  const Row = ({ index }: { index: number }) => {
    const job = filteredJobs[index];
    if (!job) {
      return null;
    }
    return (
      <tr
        key={job.id}
        onClick={() => handleRowClick(job)}
        className="table-row"
      >
        <td>{renderPath(isSmallScreen, job.source_path)}</td>
        <td className="d-none d-sm-table-cell">{job.destination_path}</td>
        <td style={{ wordBreak: "keep-all" }}>{renderStatusCellContent(job)}</td>
        <td style={{ wordBreak: "keep-all" }} title={formatDateDetailed(job.last_update)}>
          <div className="row-menu">
            {formatDateShort(job.last_update)}
            <Button
              variant="link"
              className="buttons-menu"
              onClick={handleClick}
              size="sm"
            >
              <MoreVert />
            </Button>
            <Menu
              id="buttons-menu"
              className="buttons-menu"
              anchorEl={buttonsMenu}
              keepMounted
              open={Boolean(buttonsMenu)}
              onClose={handleClose}
              anchorOrigin={{
                vertical: 'bottom',
                horizontal: 'right',
              }}
              transformOrigin={{
                vertical: 'top',
                horizontal: 'right',
              }}
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
              onClose={handleCloseDetailsMenu}
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
        </td>
      </tr>
    );
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
          <FormControlMui>
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
          </FormControlMui>
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
          <Button variant="link" className="float-right" style={{ marginLeft: 'auto' }} onClick={reload}>
            <Cached />
          </Button>
        </div>
      </div>
      <div className="job-list">
        <VirtualTable
          height={height}
          width="100%"
          itemCount={filteredJobs.length}
          itemSize={105}
          header={
            <thead>
              <tr>
                <th>
                  <span title="Source">
                    <Task />
                  </span>
                </th>
                <th className="d-none d-sm-table-cell">
                  <span title="Destination">
                    <VideoSettings />
                  </span>
                </th>
                <th>
                  <span title="Status">
                    <QuestionMark />
                  </span>
                </th>
                <th>
                  <span title="LastUpdate">
                    <CalendarMonth />
                  </span>
                </th>
              </tr>
            </thead>
          }
          row={Row}
        />
      </div>
    </div>
  );
};

export default JobTable;
