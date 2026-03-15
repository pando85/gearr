<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { goto } from '$app/navigation';
  import { jobStore, authStore, toastStore } from '$lib/stores';
  import { fetchJobs, deleteJob, createJobRequest } from '$lib/api';
  import { createJobUpdateNotification, type Job } from '$lib/model';
  import { STATUS_FILTER_OPTIONS, DATE_FILTER_OPTIONS, formatDateShort, formatDateDetailed, getDateFromFilterOption, sortJobs } from '$lib/utils';
  import IconSearch from '$lib/components/icons/IconSearch.svelte';
  import IconRefresh from '$lib/components/icons/IconRefresh.svelte';
  import IconArrowUp from '$lib/components/icons/IconArrowUp.svelte';
  import IconArrowDown from '$lib/components/icons/IconArrowDown.svelte';
  import IconDelete from '$lib/components/icons/IconDelete.svelte';
  import IconReplay from '$lib/components/icons/IconReplay.svelte';
  import IconInfoOutline from '$lib/components/icons/IconInfoOutline.svelte';
  import IconErrorOutline from '$lib/components/icons/IconErrorOutline.svelte';
  import IconAssignment from '$lib/components/icons/IconAssignment.svelte';

  let nameFilter = $state('');
  let statusFilter = $state<string[]>([]);
  let dateFilter = $state('');
  let sortColumn = $state<string | null>('last_update');
  let sortDirection = $state<'asc' | 'desc'>('desc');
  let selectedJob = $state<Job | null>(null);
  let ws: WebSocket | null = null;

  onMount(async () => {
    const token = authStore.getToken();
    if (!token) {
      goto('/');
      return;
    }

    try {
      await fetchJobs(token);
    } catch (error) {
      authStore.logout();
      goto('/');
    }

    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const wsURL = `${protocol}://${window.location.hostname}:${window.location.port}/ws/job?token=${token}`;
    
    ws = new WebSocket(wsURL);
    ws.onmessage = (event) => {
      const notification = createJobUpdateNotification(JSON.parse(event.data));
      jobStore.updateJob(notification);
    };
  });

  onDestroy(() => {
    if (ws) {
      ws.close();
    }
  });

  async function handleReload() {
    const token = authStore.getToken();
    if (!token) return;
    
    jobStore.reset();
    try {
      await fetchJobs(token);
      toastStore.info('Refreshing jobs...');
    } catch (error) {
      toastStore.error('Failed to refresh jobs');
    }
  }

  async function handleDeleteJob(jobId: string) {
    const token = authStore.getToken();
    if (!token) return;
    
    try {
      await deleteJob(token, jobId);
      toastStore.success('Job deleted successfully');
    } catch (error) {
      toastStore.error('Failed to delete job');
    }
  }

  async function handleRecreateJob(job: Job) {
    const token = authStore.getToken();
    if (!token) return;
    
    try {
      await deleteJob(token, job.id);
      await createJobRequest(token, job.source_path);
      toastStore.success('Job recreated successfully');
    } catch (error) {
      toastStore.error('Failed to recreate job');
    }
  }

  function handleSort(column: string) {
    if (sortColumn === column) {
      sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
    } else {
      sortColumn = column;
      sortDirection = 'asc';
    }
  }

  const filteredJobs = $derived(() => {
    let result = $jobStore.jobs;

    if (statusFilter.length > 0) {
      result = result.filter((job) => statusFilter.includes(job.status));
    }

    if (dateFilter) {
      result = result.filter((job) => job.last_update >= getDateFromFilterOption(dateFilter));
    }

    if (nameFilter) {
      result = result.filter((job) =>
        job.source_path?.toLowerCase().includes(nameFilter.toLowerCase())
      );
    }

    return sortJobs(sortColumn, sortDirection, result);
  });

  function getStatusBadgeClass(status: string) {
    const classes: Record<string, string> = {
      progressing: 'progressing',
      completed: 'completed',
      failed: 'failed',
      queued: 'queued',
    };
    return classes[status] || 'queued';
  }

  function renderStatus(job: Job) {
    if (job.status === 'progressing' && job.status_phase === 'FFMPEG') {
      try {
        const messageObj = JSON.parse(job.status_message);
        const progress = messageObj.progress !== undefined ? parseFloat(messageObj.progress) : 0;
        return {
          type: 'progress' as const,
          progress,
          title: `${progress.toFixed(2)}%`,
        };
      } catch {
        return { type: 'badge' as const, text: job.status };
      }
    }

    if (job.status === 'failed') {
      return { type: 'error' as const, message: job.status_message };
    }

    const displayStatus = job.status_phase !== 'Job' ? job.status_phase.toLowerCase() : job.status;
    return { type: 'badge' as const, text: displayStatus };
  }

  function truncatePath(path: string, maxLength: number = 40) {
    const fileName = path.split('/').pop() || path;
    if (fileName.length > maxLength) {
      return fileName.substring(0, maxLength) + '...';
    }
    return fileName;
  }
</script>

<div class="jobs-page">
  <div class="jobs-header">
    <h1 class="jobs-title">Jobs</h1>
    <div class="jobs-actions">
      <div class="jobs-search">
        <IconSearch class="jobs-search-icon" />
        <input
          class="jobs-search-input"
          type="text"
          placeholder="Search jobs..."
          bind:value={nameFilter}
        />
      </div>
      <div class="jobs-filters">
        <select
          class="jobs-filter-select"
          value={statusFilter.length > 0 ? statusFilter[0] : ''}
          onchange={(e) => statusFilter = e.currentTarget.value ? [e.currentTarget.value] : []}
        >
          <option value="">All Status</option>
          {#each STATUS_FILTER_OPTIONS as status}
            <option value={status}>{status}</option>
          {/each}
        </select>
        <select
          class="jobs-filter-select"
          bind:value={dateFilter}
        >
          {#each DATE_FILTER_OPTIONS as option}
            <option value={option === 'Last update' ? '' : option}>
              {option}
            </option>
          {/each}
        </select>
        <button class="jobs-refresh-btn" onclick={handleReload} title="Refresh">
          <IconRefresh class="w-5 h-5" />
        </button>
      </div>
    </div>
  </div>

  <div class="jobs-table-wrapper">
    <div class="jobs-thead">
      <button class="jobs-th jobs-th-source sortable" onclick={() => handleSort('source_path')}>
        Source
        {#if sortColumn === 'source_path'}
          <IconArrowUp class="jobs-sort-icon active" />
        {:else}
          <IconArrowDown class="jobs-sort-icon" />
        {/if}
      </button>
      <button class="jobs-th jobs-th-destination sortable" onclick={() => handleSort('destination_path')}>
        Destination
        {#if sortColumn === 'destination_path'}
          <IconArrowUp class="jobs-sort-icon active" />
        {:else}
          <IconArrowDown class="jobs-sort-icon" />
        {/if}
      </button>
      <button class="jobs-th jobs-th-status sortable" onclick={() => handleSort('status')}>
        Status
        {#if sortColumn === 'status'}
          <IconArrowUp class="jobs-sort-icon active" />
        {:else}
          <IconArrowDown class="jobs-sort-icon" />
        {/if}
      </button>
      <button class="jobs-th jobs-th-date sortable" onclick={() => handleSort('last_update')}>
        Date
        {#if sortColumn === 'last_update'}
          <IconArrowUp class="jobs-sort-icon active" />
        {:else}
          <IconArrowDown class="jobs-sort-icon" />
        {/if}
      </button>
      <div class="jobs-th jobs-th-actions">Actions</div>
    </div>

    {#if filteredJobs().length > 0}
      <div class="jobs-tbody">
        {#each filteredJobs() as job}
          {@const status = renderStatus(job)}
          <div class="jobs-tr">
            <div class="jobs-td jobs-td-source" title={job.source_path}>
              <span class="job-path">{truncatePath(job.source_path)}</span>
            </div>
            <div class="jobs-td jobs-td-destination" title={job.destination_path}>
              <span class="job-path">{truncatePath(job.destination_path)}</span>
            </div>
            <div class="jobs-td jobs-td-status">
              {#if status.type === 'progress'}
                <div class="job-progress" title={status.title}>
                  <div class="job-progress-bar">
                    <div class="job-progress-fill" style="width: {status.progress}%"></div>
                  </div>
                </div>
              {:else if status.type === 'error'}
                <div title={status.message}>
                  <IconErrorOutline class="job-error-icon" />
                </div>
              {:else}
                <span class="job-status-badge {getStatusBadgeClass(job.status)}">{status.text}</span>
              {/if}
            </div>
            <div class="jobs-td jobs-td-date" title={formatDateDetailed(job.last_update)}>
              <span class="job-date">{formatDateShort(job.last_update)}</span>
            </div>
            <div class="jobs-td jobs-td-actions">
              <div class="job-actions">
                <button
                  class="job-action-btn"
                  onclick={() => selectedJob = job}
                  title="Details"
                >
                  <IconInfoOutline class="w-4 h-4" />
                </button>
                <button
                  class="job-action-btn delete"
                  onclick={() => handleDeleteJob(job.id)}
                  title="Delete"
                >
                  <IconDelete class="w-4 h-4" />
                </button>
                <button
                  class="job-action-btn"
                  onclick={() => handleRecreateJob(job)}
                  title="Recreate"
                >
                  <IconReplay class="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <div class="jobs-empty">
        <IconAssignment class="jobs-empty-icon" />
        <p class="jobs-empty-text">No jobs found</p>
      </div>
    {/if}
  </div>

  {#if selectedJob}
    <div class="jobs-details-card">
      <div class="jobs-details-header">
        <span class="jobs-details-title">Job Details</span>
        <button class="jobs-details-close" onclick={() => selectedJob = null}>
          &times;
        </button>
      </div>
      <div class="jobs-details-body">
        <div class="jobs-details-row">
          <span class="jobs-details-label">ID</span>
          <span class="jobs-details-value">{selectedJob.id}</span>
        </div>
        <div class="jobs-details-row">
          <span class="jobs-details-label">Source</span>
          <span class="jobs-details-value">{selectedJob.source_path}</span>
        </div>
        <div class="jobs-details-row">
          <span class="jobs-details-label">Destination</span>
          <span class="jobs-details-value">{selectedJob.destination_path}</span>
        </div>
        <div class="jobs-details-row">
          <span class="jobs-details-label">Status</span>
          <span class="jobs-details-value">{selectedJob.status}</span>
        </div>
        <div class="jobs-details-row">
          <span class="jobs-details-label">Phase</span>
          <span class="jobs-details-value">{selectedJob.status_phase}</span>
        </div>
        <div class="jobs-details-row">
          <span class="jobs-details-label">Message</span>
          <span class="jobs-details-value">{selectedJob.status_message}</span>
        </div>
      </div>
      <div class="jobs-details-actions">
        <button class="btn btn-secondary" onclick={() => selectedJob = null}>
          Close
        </button>
      </div>
    </div>
  {/if}
</div>

<style>
  .jobs-page {
    padding: var(--spacing-lg);
    max-width: 1400px;
    margin: 0 auto;
  }

  .jobs-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--spacing-lg);
    margin-bottom: var(--spacing-lg);
    flex-wrap: wrap;
  }

  .jobs-title {
    font-size: var(--font-size-3xl);
    font-weight: var(--font-weight-bold);
    color: var(--text-primary);
  }

  .jobs-actions {
    display: flex;
    align-items: center;
    gap: var(--spacing-md);
    flex-wrap: wrap;
  }

  .jobs-search {
    position: relative;
  }

  .jobs-search-icon {
    position: absolute;
    left: 0.75rem;
    top: 50%;
    transform: translateY(-50%);
    width: 1.25rem;
    height: 1.25rem;
    color: var(--text-muted);
  }

  .jobs-search-input {
    width: 200px;
    padding: 0.5rem 0.75rem 0.5rem 2.5rem;
    font-size: var(--font-size-sm);
    color: var(--text-primary);
    background-color: var(--bg-input);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-md);
  }

  .jobs-search-input:focus {
    outline: none;
    border-color: var(--color-primary);
  }

  .jobs-filters {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
  }

  .jobs-filter-select {
    padding: 0.5rem 0.75rem;
    font-size: var(--font-size-sm);
    color: var(--text-primary);
    background-color: var(--bg-input);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-md);
  }

  .jobs-refresh-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2.25rem;
    height: 2.25rem;
    border: 1px solid var(--border-color);
    background-color: var(--bg-input);
    color: var(--text-secondary);
    border-radius: var(--border-radius-md);
    cursor: pointer;
  }

  .jobs-refresh-btn:hover {
    background-color: var(--bg-hover);
    color: var(--text-primary);
  }

  .jobs-table-wrapper {
    background-color: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-lg);
    overflow: hidden;
  }

  .jobs-thead {
    display: flex;
    background-color: var(--bg-secondary);
    border-bottom: 1px solid var(--border-color);
  }

  .jobs-th {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    padding: var(--spacing-md);
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-medium);
    color: var(--text-secondary);
    text-align: left;
    border: none;
    background: none;
    cursor: default;
  }

  .jobs-th.sortable {
    cursor: pointer;
  }

  .jobs-th.sortable:hover {
    color: var(--text-primary);
  }

  .jobs-th-source { flex: 1; min-width: 200px; }
  .jobs-th-destination { flex: 1; min-width: 200px; }
  .jobs-th-status { width: 150px; }
  .jobs-th-date { width: 120px; }
  .jobs-th-actions { width: 120px; }

  .jobs-sort-icon {
    width: 1rem;
    height: 1rem;
    color: var(--text-muted);
  }

  .jobs-sort-icon.active {
    color: var(--color-primary);
  }

  .jobs-tbody {
    max-height: calc(100vh - 280px);
    overflow-y: auto;
  }

  .jobs-tr {
    display: flex;
    border-bottom: 1px solid var(--border-color-light);
  }

  .jobs-tr:last-child {
    border-bottom: none;
  }

  .jobs-tr:hover {
    background-color: var(--bg-hover);
  }

  .jobs-td {
    padding: var(--spacing-md);
    font-size: var(--font-size-sm);
    color: var(--text-primary);
    display: flex;
    align-items: center;
  }

  .jobs-td-source { flex: 1; min-width: 200px; }
  .jobs-td-destination { flex: 1; min-width: 200px; }
  .jobs-td-status { width: 150px; }
  .jobs-td-date { width: 120px; }
  .jobs-td-actions { width: 120px; }

  .job-path {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .job-status-badge {
    display: inline-flex;
    align-items: center;
    padding: 0.25rem 0.5rem;
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-medium);
    border-radius: var(--border-radius-full);
  }

  .job-status-badge.progressing {
    background-color: rgba(59, 130, 246, 0.1);
    color: var(--color-info);
  }

  .job-status-badge.completed {
    background-color: rgba(34, 197, 94, 0.1);
    color: var(--color-success);
  }

  .job-status-badge.failed {
    background-color: rgba(239, 68, 68, 0.1);
    color: var(--color-error);
  }

  .job-status-badge.queued {
    background-color: var(--bg-tertiary);
    color: var(--text-secondary);
  }

  .job-progress {
    width: 100%;
  }

  .job-progress-bar {
    width: 100%;
    height: 6px;
    background-color: var(--bg-tertiary);
    border-radius: var(--border-radius-full);
    overflow: hidden;
  }

  .job-progress-fill {
    height: 100%;
    background-color: var(--color-primary);
    border-radius: var(--border-radius-full);
    transition: width 0.3s ease;
  }

  .job-error-icon {
    width: 1.25rem;
    height: 1.25rem;
    color: var(--color-error);
  }

  .job-actions {
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .job-action-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2rem;
    height: 2rem;
    border: none;
    background: none;
    color: var(--text-muted);
    border-radius: var(--border-radius-md);
    cursor: pointer;
  }

  .job-action-btn:hover {
    background-color: var(--bg-tertiary);
    color: var(--text-primary);
  }

  .job-action-btn.delete:hover {
    color: var(--color-error);
  }

  .jobs-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: var(--spacing-2xl);
    color: var(--text-muted);
  }

  .jobs-empty-icon {
    width: 3rem;
    height: 3rem;
    margin-bottom: var(--spacing-md);
  }

  .jobs-empty-text {
    font-size: var(--font-size-sm);
  }

  .jobs-details-card {
    position: fixed;
    bottom: var(--spacing-lg);
    right: var(--spacing-lg);
    width: 320px;
    background-color: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-lg);
    box-shadow: var(--shadow-xl);
    z-index: 1050;
    animation: slideUp var(--transition-normal);
  }

  .jobs-details-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--spacing-md);
    border-bottom: 1px solid var(--border-color);
  }

  .jobs-details-title {
    font-size: var(--font-size-base);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
  }

  .jobs-details-close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.5rem;
    height: 1.5rem;
    border: none;
    background: none;
    color: var(--text-muted);
    font-size: 1.25rem;
    cursor: pointer;
    border-radius: var(--border-radius-sm);
  }

  .jobs-details-close:hover {
    background-color: var(--bg-hover);
    color: var(--text-primary);
  }

  .jobs-details-body {
    padding: var(--spacing-md);
  }

  .jobs-details-row {
    display: flex;
    margin-bottom: var(--spacing-sm);
  }

  .jobs-details-row:last-child {
    margin-bottom: 0;
  }

  .jobs-details-label {
    width: 80px;
    flex-shrink: 0;
    font-size: var(--font-size-xs);
    color: var(--text-muted);
  }

  .jobs-details-value {
    flex: 1;
    font-size: var(--font-size-sm);
    color: var(--text-primary);
    word-break: break-all;
  }

  .jobs-details-actions {
    padding: var(--spacing-md);
    border-top: 1px solid var(--border-color);
  }

  @media (max-width: 768px) {
    .jobs-header {
      flex-direction: column;
    }

    .jobs-actions {
      width: 100%;
    }

    .jobs-search {
      flex: 1;
    }

    .jobs-search-input {
      width: 100%;
    }

    .jobs-filter-select {
      flex: 1;
    }
  }
</style>