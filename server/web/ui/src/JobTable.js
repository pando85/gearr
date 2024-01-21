// JobTable.js
import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { Table, TableBody, TableCell, TableHead, TableRow, CircularProgress, Typography, Button } from '@mui/material';
import { Info, QuestionMark, Task, VideoSettings } from '@mui/icons-material';

import './JobTable.css';

const JobTable = ({ token, setShowJobTable }) => {
  const [jobs, setJobs] = useState([]);
  const [selectedJob, setSelectedJob] = useState(null);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [fetchedDetails, setFetchedDetails] = useState(new Set());

  useEffect(() => {
    const fetchJobs = async () => {
      try {
        setLoading(true);
        const response = await axios.get('/api/v1/job/', {
          params: { token, page },
        });

        setJobs((prevJobs) => [...prevJobs, ...response.data]);
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
      if (
        window.innerHeight + window.scrollY >=
        document.body.offsetHeight - 100
      ) {
        setPage((prevPage) => prevPage + 1);
      }
    };

    window.addEventListener('scroll', handleScroll);

    return () => {
      window.removeEventListener('scroll', handleScroll);
    };
  }, []);

  useEffect(() => {
    const fetchJobDetails = async (jobId) => {
      if (!fetchedDetails.has(jobId)) {
        try {
          const response = await axios.get(`/api/v1/job/`, {
            params: { token, uuid: jobId },
          });
          const enrichedJob = {
            ...jobs.find((job) => job.id === jobId),
            sourcePath: response.data.sourcePath,
            destinationPath: response.data.destinationPath,
            status: response.data.status,
            status_message: response.data.status_message,
          };
          setJobs((prevJobs) =>
            prevJobs.map((job) =>
              job.id === jobId ? enrichedJob : job
            )
          );
          setFetchedDetails((prevSet) => new Set(prevSet.add(jobId)));
        } catch (error) {
          console.error(`Error fetching details for job ${jobId}:`, error);
        }
      }
    };

    // Fetch details for each job when they are rendered in the table
    jobs.forEach((job) => fetchJobDetails(job.id));
  }, [token, jobs, fetchedDetails]);

  const handleRowClick = (jobId) => {
    setSelectedJob(jobs.find((job) => job.id === jobId));
  };

  const getStatusColor = (status) => {
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
    <div className="jobTableContainer">
      <Table className="jobTable">
        <TableHead>
          <TableRow>
            <TableCell className="tableHeader"> <span title="Source"><Task/></span></TableCell>
            <TableCell className="tableHeader"><span title="Destionation"><VideoSettings/></span></TableCell>
            <TableCell className="tableHeader"><span title="Status"><QuestionMark/></span></TableCell>
            <TableCell className="tableHeader"><span title="Message"><Info/></span></TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {jobs.map((job) => (
            <TableRow
              key={job.id}
              onClick={() => handleRowClick(job.id)}
              className="tableRow"
            >
              <TableCell>{job.sourcePath}</TableCell>
              <TableCell>{job.destinationPath}</TableCell>
              <TableCell>
                <Button
                  variant="contained"
                  style={{
                    backgroundColor: getStatusColor(job.status),
                    color: '#282c34',
                  }}
                >
                  {job.status}
                </Button>
              </TableCell>
              <TableCell>{job.status_message}</TableCell>
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
