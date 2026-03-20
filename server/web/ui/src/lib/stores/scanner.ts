import { writable, derived, get } from 'svelte/store';
import type { ScannerStatus, ScannerNotification, LibraryScan } from './scanner-model';

export interface ScannerState {
  status: ScannerStatus | null;
  history: LibraryScan[];
  isLoading: boolean;
  error: string | null;
  isScanning: boolean;
}

function createScannerStore() {
  const { subscribe, set, update } = writable<ScannerState>({
    status: null,
    history: [],
    isLoading: false,
    error: null,
    isScanning: false
  });

  let ws: WebSocket | null = null;
  let reconnectAttempts = 0;
  const maxReconnectAttempts = 5;

  const handleNotification = (notification: ScannerNotification) => {
    update(s => {
      const isScanning = notification.status === 'running';
      return { 
        ...s, 
        isScanning,
        status: s.status ? {
          ...s.status,
          is_scanning: isScanning,
          last_scan: notification.status === 'completed' ? {
            id: notification.scan_id,
            started_at: new Date().toISOString(),
            status: 'completed',
            files_found: notification.files_found,
            files_queued: notification.files_queued,
            files_skipped_size: 0,
            files_skipped_codec: 0,
            files_skipped_exists: notification.files_skipped
          } : s.status.last_scan
        } : null
      };
    });
  };

  const connectWebSocket = (token: string) => {
    if (ws) {
      ws.close();
    }
    
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws/scanner?token=${token}`;
    
    ws = new WebSocket(wsUrl);
    
    ws.onopen = () => {
      reconnectAttempts = 0;
      console.log('Scanner WebSocket connected');
    };
    
    ws.onmessage = (event) => {
      try {
        const notification: ScannerNotification = JSON.parse(event.data);
        handleNotification(notification);
      } catch (e) {
        console.error('Failed to parse scanner notification:', e);
      }
    };
    
    ws.onerror = (error) => {
      console.error('Scanner WebSocket error:', error);
    };
    
    ws.onclose = () => {
      console.log('Scanner WebSocket closed');
      if (reconnectAttempts < maxReconnectAttempts) {
        reconnectAttempts++;
        setTimeout(() => {
          connectWebSocket(token);
        }, 1000 * reconnectAttempts);
      }
    };
  };

  const disconnectWebSocket = () => {
    if (ws) {
      ws.close();
      ws = null;
    }
  };

  return {
    subscribe,
    
    setLoading: () => update(s => ({ ...s, isLoading: true, error: null })),
    
    setError: (error: string) => update(s => ({ ...s, isLoading: false, error })),
    
    setStatus: (status: ScannerStatus) => update(s => ({ 
      ...s, 
      status, 
      isLoading: false,
      isScanning: status.is_scanning 
    })),
    
    setHistory: (history: LibraryScan[]) => update(s => ({ ...s, history })),
    
    handleNotification,
    connectWebSocket,
    disconnectWebSocket
  };
}

export const scannerStore = createScannerStore();

export const scannerEnabled = derived(scannerStore, $s => $s.status?.enabled ?? false);
export const scannerIsScanning = derived(scannerStore, $s => $s.isScanning);
export const lastScan = derived(scannerStore, $s => $s.status?.last_scan ?? null);
export const scanHistory = derived(scannerStore, $s => $s.history);