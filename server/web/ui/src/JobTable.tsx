// JobTable.tsx
import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { Table, TableBody, TableCell, TableHead, TableRow, CircularProgress, Typography, Button } from '@mui/material';
import { CalendarMonth, Info, QuestionMark, Task, VideoSettings } from '@mui/icons-material';

import './JobTable.css';

interface Job {
  id: string;
  sourcePath: string;
  destinationPath: string;
  status: string;
  status_message: string;
  last_update: Date;
}

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
    timeStyle: "long",
  };
  return formatDate(date, options)
}

const formatDateShort = (date: Date): string => {
  const options: Intl.DateTimeFormatOptions = {
    dateStyle: "short",
  };
  return formatDate(date, options)
}

const JobTable: React.FC<JobTableProps> = ({ token, setShowJobTable }) => {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [selectedJob, setSelectedJob] = useState<Job | null>(null);
  const [page, setPage] = useState<number>(1);
  const [loading, setLoading] = useState<boolean>(false);
  const [fetchedDetails, setFetchedDetails] = useState<Set<string>>(new Set());

  useEffect(() => {
    const fetchJobs = async () => {
      try {
        setLoading(true);
        const response = await axios.get('/api/v1/job/', {
          params: { page },
          headers: {
            Authorization: `Bearer ${token}`,
          },
        });
        const newJobs: Job[] = response.data;
        setJobs((prevJobs) => [...prevJobs, ...newJobs]);
      } catch (error) {
        console.error('Error fetching jobs:', error);
        setShowJobTable(false);
      } finally {
        setLoading(false);
      }
    };

    fetchJobs();
  }, [token, page, setShowJobTable]);

  useEffect(() => {
    const handleScroll = () => {
      if (window.innerHeight + window.scrollY >= document.body.offsetHeight - 100) {
        setPage((prevPage) => prevPage + 1);
      }
    };

    window.addEventListener('scroll', handleScroll);

    return () => {
      window.removeEventListener('scroll', handleScroll);
    };
  }, []);

  useEffect(() => {
    const fetchJobDetails = async (jobId: string) => {
      if (!fetchedDetails.has(jobId)) {
        try {
          const response = await axios.get(`/api/v1/job/${jobId}`, {
            headers: {
              Authorization: `Bearer ${token}`,
            },
          });

          const foundJob = jobs.find((job) => job.id === jobId);

          if (foundJob) {
            const enrichedJob: Job = {
              ...foundJob,
              sourcePath: response.data.sourcePath,
              destinationPath: response.data.destinationPath,
              status: response.data.status,
              status_message: response.data.status_message,
              last_update: new Date(response.data.last_update),
            };

            setJobs((prevJobs) =>
              prevJobs.map((job) => (job.id === jobId ? enrichedJob : job))
            );
          }

          setFetchedDetails((prevSet) => new Set(prevSet.add(jobId)));
        } catch (error) {
          console.error(`Error fetching details for job ${jobId}:`, error);
        }
      }
    };


    // Fetch details for each job when they are rendered in the table
    jobs.forEach((job) => fetchJobDetails(job.id));
  }, [token, jobs, fetchedDetails]);

  const handleRowClick = (jobId: string) => {
    setSelectedJob(jobs.find((job) => job.id === jobId) || null);
  };

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

  return (
    <div className="float-center">
      <Table className="jobTable">
        <TableHead>
          <TableRow>
            <TableCell > <span title="Source"><Task /></span></TableCell>
            <TableCell className="d-none d-sm-table-cell"><span title="Destionation"><VideoSettings /></span></TableCell>
            <TableCell ><span title="Status"><QuestionMark /></span></TableCell>
            <TableCell className="d-none d-sm-table-cell"><span title="Message"><Info /></span></TableCell>
            <TableCell ><span title="LastUpdate"><CalendarMonth /></span></TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {jobs.map((job) => (
            <TableRow
              key={job.id}
              onClick={() => handleRowClick(job.id)}
              className="tableRow"
            >
              <TableCell >{job.sourcePath}</TableCell>
              <TableCell className="d-none d-sm-table-cell">{job.destinationPath}</TableCell>
              <TableCell >
                <Button
                  variant="contained"
                  style={{
                    backgroundColor: getStatusColor(job.status),
                  }}
                >
                  {job.status}
                </Button>
              </TableCell>
              <TableCell className="d-none d-sm-table-cell">{job.status_message}</TableCell>
              <TableCell title={formatDateDetailed(job.last_update)}>{formatDateShort(job.last_update)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      {loading && <CircularProgress />}

      {selectedJob && selectedJob.id && (
        <div>
          <Typography variant="h5" gutterBottom>
            Selected Job Details
          </Typography>
          <Typography>ID: {selectedJob.id}</Typography>
          <Typography>Source : {selectedJob.sourcePath}</Typography>
          <Typography>Destination: {selectedJob.destinationPath}</Typography>
          <Typography>Status: {selectedJob.status}</Typography>
          <Typography>Message: {selectedJob.status_message}</Typography>
        </div>
      )}
    </div>
  );
};

export default JobTable;
