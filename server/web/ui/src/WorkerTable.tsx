import React, { useState, useEffect } from 'react';
import axios from 'axios';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  CircularProgress,
} from '@mui/material';

interface Worker {
  name: string;
  id: string;
  queue_name: string;
  last_seen: string;
}

interface WorkerTableProps {
  token: string;
  setShowJobTable: React.Dispatch<React.SetStateAction<boolean>>;
  setErrorText: React.Dispatch<React.SetStateAction<string>>;
}

const WorkersTable: React.FC<WorkerTableProps> = ({ token, setShowJobTable, setErrorText }) => {
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [loading, setLoading] = useState<boolean>(false);

  useEffect(() => {
    const fetchWorkers = async () => {
      try {
        setLoading(true);
        const response = await axios.get('/api/v1/workers',
          {
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
  }, [token, setShowJobTable]);

  return (
    <div>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>Name</TableCell>
            <TableCell>ID</TableCell>
            <TableCell>Queue Name</TableCell>
            <TableCell>Last Seen</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {workers.map((worker) => (
            <TableRow key={worker.id}>
              <TableCell>{worker.name}</TableCell>
              <TableCell>{worker.id}</TableCell>
              <TableCell>{worker.queue_name}</TableCell>
              <TableCell>{worker.last_seen}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      {loading && <CircularProgress />}
    </div>
  );
};

export default WorkersTable;
