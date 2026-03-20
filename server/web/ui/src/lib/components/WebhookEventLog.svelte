<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { authStore, toastStore } from '$lib/stores';
  import { fetchWebhookEvents } from '$lib/api';
  import type { WebhookEvent } from '$lib/webhook-model';
  import { formatDateShort, formatDateDetailed } from '$lib/utils';
  import IconSearch from '$lib/components/icons/IconSearch.svelte';
  import IconRefresh from '$lib/components/icons/IconRefresh.svelte';
  import IconArrowUp from '$lib/components/icons/IconArrowUp.svelte';
  import IconArrowDown from '$lib/components/icons/IconArrowDown.svelte';
  import IconInfoOutline from '$lib/components/icons/IconInfoOutline.svelte';
  import IconCheckCircle from '$lib/components/icons/IconCheckCircle.svelte';
  import IconError from '$lib/components/icons/IconError.svelte';
  import IconClose from '$lib/components/icons/IconClose.svelte';

  let events = $state<WebhookEvent[]>([]);
  let loading = $state(true);
  let sourceFilter = $state('');
  let eventTypeFilter = $state('');
  let statusFilter = $state('');
  let searchFilter = $state('');
  let selectedEvent = $state<WebhookEvent | null>(null);
  let expandedPayload = $state(false);

  const sourceOptions = ['radarr', 'sonarr'];
  const eventTypeOptions = ['download', 'grab', 'rename', 'test', 'movie_delete'];
  const statusOptions = ['success', 'failed', 'skipped'];

  onMount(async () => {
    const token = authStore.getToken();
    if (!token) {
      goto('/');
      return;
    }
    await loadEvents();
  });

  async function loadEvents() {
    const token = authStore.getToken();
    if (!token) return;

    loading = true;
    try {
      events = await fetchWebhookEvents(
        token,
        sourceFilter || undefined,
        eventTypeFilter || undefined,
        statusFilter || undefined
      );
    } catch (error) {
      toastStore.error('Failed to load webhook events');
    } finally {
      loading = false;
    }
  }

  async function handleRefresh() {
    await loadEvents();
    toastStore.info('Refreshed webhook events');
  }

  function clearFilters() {
    sourceFilter = '';
    eventTypeFilter = '';
    statusFilter = '';
    searchFilter = '';
    loadEvents();
  }

  const filteredEvents = $derived(() => {
    if (!searchFilter) return events;
    return events.filter(
      (event) =>
        event.file_path?.toLowerCase().includes(searchFilter.toLowerCase()) ||
        event.message?.toLowerCase().includes(searchFilter.toLowerCase()) ||
        event.source?.toLowerCase().includes(searchFilter.toLowerCase())
    );
  });

  function getStatusIcon(status: string) {
    switch (status) {
      case 'success':
        return IconCheckCircle;
      case 'failed':
        return IconError;
      default:
        return null;
    }
  }

  function getStatusClass(status: string) {
    switch (status) {
      case 'success':
        return 'status-success';
      case 'failed':
        return 'status-failed';
      case 'skipped':
        return 'status-skipped';
      default:
        return '';
    }
  }

  function truncatePath(path: string, maxLength: number = 40) {
    if (!path) return '-';
    const fileName = path.split('/').pop() || path;
    if (fileName.length > maxLength) {
      return fileName.substring(0, maxLength) + '...';
    }
    return fileName;
  }

  function formatPayload(payload: string) {
    if (!payload) return null;
    try {
      return JSON.stringify(JSON.parse(payload), null, 2);
    } catch {
      return payload;
    }
  }
</script>

<div class="webhook-page">
  <div class="webhook-header">
    <h1 class="webhook-title">Webhook Events</h1>
    <div class="webhook-actions">
      <div class="webhook-search">
        <IconSearch class="webhook-search-icon" />
        <input
          class="webhook-search-input"
          type="text"
          placeholder="Search events..."
          bind:value={searchFilter}
        />
      </div>
      <div class="webhook-filters">
        <select class="webhook-filter-select" bind:value={sourceFilter} onchange={loadEvents}>
          <option value="">All Sources</option>
          {#each sourceOptions as source}
            <option value={source}>{source}</option>
          {/each}
        </select>
        <select class="webhook-filter-select" bind:value={eventTypeFilter} onchange={loadEvents}>
          <option value="">All Events</option>
          {#each eventTypeOptions as eventType}
            <option value={eventType}>{eventType}</option>
          {/each}
        </select>
        <select class="webhook-filter-select" bind:value={statusFilter} onchange={loadEvents}>
          <option value="">All Status</option>
          {#each statusOptions as status}
            <option value={status}>{status}</option>
          {/each}
        </select>
        <button class="webhook-clear-btn" onclick={clearFilters} title="Clear Filters">
          <IconClose class="w-4 h-4" />
        </button>
        <button class="webhook-refresh-btn" onclick={handleRefresh} title="Refresh">
          <IconRefresh class="w-5 h-5" />
        </button>
      </div>
    </div>
  </div>

  <div class="webhook-table-wrapper">
    <div class="webhook-thead">
      <div class="webhook-th webhook-th-timestamp">Timestamp</div>
      <div class="webhook-th webhook-th-source">Source</div>
      <div class="webhook-th webhook-th-event">Event</div>
      <div class="webhook-th webhook-th-file">File</div>
      <div class="webhook-th webhook-th-status">Status</div>
      <div class="webhook-th webhook-th-actions">Actions</div>
    </div>

    {#if loading}
      <div class="webhook-loading">
        <p>Loading...</p>
      </div>
    {:else if filteredEvents().length > 0}
      <div class="webhook-tbody">
        {#each filteredEvents() as event}
          <div class="webhook-tr" onclick={() => selectedEvent = event}>
            <div class="webhook-td webhook-th-timestamp" title={formatDateDetailed(event.created_at)}>
              <span class="webhook-date">{formatDateShort(event.created_at)}</span>
            </div>
            <div class="webhook-td webhook-th-source">
              <span class="webhook-source-badge">{event.source}</span>
            </div>
            <div class="webhook-td webhook-th-event">
              <span class="webhook-event-type">{event.event_type}</span>
            </div>
            <div class="webhook-td webhook-th-file" title={event.file_path}>
              <span class="webhook-file-path">{truncatePath(event.file_path)}</span>
            </div>
            <div class="webhook-td webhook-th-status">
              <span class="webhook-status-badge {getStatusClass(event.status)}">{event.status}</span>
            </div>
            <div class="webhook-td webhook-th-actions">
              <button
                class="webhook-action-btn"
                onclick={(e) => { e.stopPropagation(); selectedEvent = event; }}
                title="Details"
              >
                <IconInfoOutline class="w-4 h-4" />
              </button>
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <div class="webhook-empty-state">
        <p class="webhook-empty-state-text">No webhook events found</p>
      </div>
    {/if}
  </div>

  {#if selectedEvent}
    <div class="webhook-details-overlay" onclick={() => selectedEvent = null}>
      <div class="webhook-details-card" onclick={(e) => e.stopPropagation()}>
        <div class="webhook-details-header">
          <span class="webhook-details-title">Event Details</span>
          <button class="webhook-details-close" onclick={() => selectedEvent = null}>
            &times;
          </button>
        </div>
        <div class="webhook-details-body">
          <div class="webhook-details-row">
            <span class="webhook-details-label">ID</span>
            <span class="webhook-details-value">{selectedEvent.id}</span>
          </div>
          <div class="webhook-details-row">
            <span class="webhook-details-label">Source</span>
            <span class="webhook-details-value">{selectedEvent.source}</span>
          </div>
          <div class="webhook-details-row">
            <span class="webhook-details-label">Event Type</span>
            <span class="webhook-details-value">{selectedEvent.event_type}</span>
          </div>
          <div class="webhook-details-row">
            <span class="webhook-details-label">Status</span>
            <span class="webhook-details-value">
              <span class="webhook-status-badge {getStatusClass(selectedEvent.status)}">
                {selectedEvent.status}
              </span>
            </span>
          </div>
          <div class="webhook-details-row">
            <span class="webhook-details-label">File Path</span>
            <span class="webhook-details-value">{selectedEvent.file_path || '-'}</span>
          </div>
          <div class="webhook-details-row">
            <span class="webhook-details-label">Message</span>
            <span class="webhook-details-value">{selectedEvent.message || '-'}</span>
          </div>
          {#if selectedEvent.job_id}
            <div class="webhook-details-row">
              <span class="webhook-details-label">Job ID</span>
              <span class="webhook-details-value">{selectedEvent.job_id}</span>
            </div>
          {/if}
          <div class="webhook-details-row">
            <span class="webhook-details-label">Timestamp</span>
            <span class="webhook-details-value">{formatDateDetailed(selectedEvent.created_at)}</span>
          </div>
          {#if selectedEvent.error_details}
            <div class="webhook-details-row webhook-details-error">
              <span class="webhook-details-label">Error</span>
              <span class="webhook-details-value">{selectedEvent.error_details}</span>
            </div>
          {/if}
          {#if selectedEvent.payload}
            <div class="webhook-details-payload">
              <div class="webhook-details-payload-header">
                <span class="webhook-details-label">Payload</span>
                <button
                  class="webhook-payload-toggle"
                  onclick={() => expandedPayload = !expandedPayload}
                >
                  {expandedPayload ? 'Collapse' : 'Expand'}
                </button>
              </div>
              <pre class="webhook-payload-content {expandedPayload ? 'expanded' : ''}">
{formatPayload(selectedEvent.payload)}</pre>
            </div>
          {/if}
        </div>
        <div class="webhook-details-actions">
          <button class="btn btn-secondary" onclick={() => selectedEvent = null}>
            Close
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .webhook-page {
    padding: var(--spacing-lg);
    max-width: 1400px;
    margin: 0 auto;
  }

  .webhook-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--spacing-lg);
    margin-bottom: var(--spacing-lg);
    flex-wrap: wrap;
  }

  .webhook-title {
    font-size: var(--font-size-3xl);
    font-weight: var(--font-weight-bold);
    color: var(--text-primary);
  }

  .webhook-actions {
    display: flex;
    align-items: center;
    gap: var(--spacing-md);
    flex-wrap: wrap;
  }

  .webhook-search {
    position: relative;
  }

  .webhook-search-icon {
    position: absolute;
    left: 0.75rem;
    top: 50%;
    transform: translateY(-50%);
    width: 1.25rem;
    height: 1.25rem;
    color: var(--text-muted);
  }

  .webhook-search-input {
    width: 200px;
    padding: 0.5rem 0.75rem 0.5rem 2.5rem;
    font-size: var(--font-size-sm);
    color: var(--text-primary);
    background-color: var(--bg-input);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-md);
  }

  .webhook-search-input:focus {
    outline: none;
    border-color: var(--color-primary);
  }

  .webhook-filters {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
  }

  .webhook-filter-select {
    padding: 0.5rem 0.75rem;
    font-size: var(--font-size-sm);
    color: var(--text-primary);
    background-color: var(--bg-input);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-md);
  }

  .webhook-refresh-btn,
  .webhook-clear-btn {
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

  .webhook-refresh-btn:hover,
  .webhook-clear-btn:hover {
    background-color: var(--bg-hover);
    color: var(--text-primary);
  }

  .webhook-table-wrapper {
    background-color: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-lg);
    overflow: hidden;
  }

  .webhook-thead {
    display: flex;
    background-color: var(--bg-secondary);
    border-bottom: 1px solid var(--border-color);
  }

  .webhook-th {
    padding: var(--spacing-md);
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-medium);
    color: var(--text-secondary);
    text-align: left;
  }

  .webhook-th-timestamp { width: 140px; }
  .webhook-th-source { width: 100px; }
  .webhook-th-event { width: 100px; }
  .webhook-th-file { flex: 1; min-width: 200px; }
  .webhook-th-status { width: 100px; }
  .webhook-th-actions { width: 80px; }

  .webhook-tbody {
    max-height: calc(100vh - 280px);
    overflow-y: auto;
  }

  .webhook-tr {
    display: flex;
    border-bottom: 1px solid var(--border-color-light);
    cursor: pointer;
  }

  .webhook-tr:last-child {
    border-bottom: none;
  }

  .webhook-tr:hover {
    background-color: var(--bg-hover);
  }

  .webhook-td {
    padding: var(--spacing-md);
    font-size: var(--font-size-sm);
    color: var(--text-primary);
    display: flex;
    align-items: center;
  }

  .webhook-date {
    color: var(--text-secondary);
  }

  .webhook-source-badge {
    display: inline-flex;
    align-items: center;
    padding: 0.25rem 0.5rem;
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-medium);
    border-radius: var(--border-radius-full);
    background-color: rgba(59, 130, 246, 0.1);
    color: var(--color-info);
    text-transform: capitalize;
  }

  .webhook-event-type {
    text-transform: capitalize;
  }

  .webhook-file-path {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .webhook-status-badge {
    display: inline-flex;
    align-items: center;
    padding: 0.25rem 0.5rem;
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-medium);
    border-radius: var(--border-radius-full);
    text-transform: capitalize;
  }

  .webhook-status-badge.status-success {
    background-color: rgba(34, 197, 94, 0.1);
    color: var(--color-success);
  }

  .webhook-status-badge.status-failed {
    background-color: rgba(239, 68, 68, 0.1);
    color: var(--color-error);
  }

  .webhook-status-badge.status-skipped {
    background-color: var(--bg-tertiary);
    color: var(--text-secondary);
  }

  .webhook-action-btn {
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

  .webhook-action-btn:hover {
    background-color: var(--bg-tertiary);
    color: var(--text-primary);
  }

  .webhook-loading,
  .webhook-empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: var(--spacing-2xl);
    color: var(--text-muted);
  }

  .webhook-empty-state-text {
    font-size: var(--font-size-sm);
  }

  .webhook-details-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1050;
  }

  .webhook-details-card {
    width: 600px;
    max-width: 90vw;
    max-height: 80vh;
    background-color: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-lg);
    box-shadow: var(--shadow-xl);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .webhook-details-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--spacing-md);
    border-bottom: 1px solid var(--border-color);
  }

  .webhook-details-title {
    font-size: var(--font-size-base);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
  }

  .webhook-details-close {
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

  .webhook-details-close:hover {
    background-color: var(--bg-hover);
    color: var(--text-primary);
  }

  .webhook-details-body {
    padding: var(--spacing-md);
    overflow-y: auto;
    flex: 1;
  }

  .webhook-details-row {
    display: flex;
    margin-bottom: var(--spacing-sm);
  }

  .webhook-details-row:last-child {
    margin-bottom: 0;
  }

  .webhook-details-label {
    width: 100px;
    flex-shrink: 0;
    font-size: var(--font-size-xs);
    color: var(--text-muted);
  }

  .webhook-details-value {
    flex: 1;
    font-size: var(--font-size-sm);
    color: var(--text-primary);
    word-break: break-all;
  }

  .webhook-details-error .webhook-details-value {
    color: var(--color-error);
  }

  .webhook-details-payload {
    margin-top: var(--spacing-md);
    border-top: 1px solid var(--border-color);
    padding-top: var(--spacing-md);
  }

  .webhook-details-payload-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-sm);
  }

  .webhook-payload-toggle {
    font-size: var(--font-size-xs);
    color: var(--color-primary);
    background: none;
    border: none;
    cursor: pointer;
  }

  .webhook-payload-content {
    font-size: var(--font-size-xs);
    background-color: var(--bg-secondary);
    padding: var(--spacing-sm);
    border-radius: var(--border-radius-md);
    overflow-x: auto;
    max-height: 200px;
    white-space: pre-wrap;
    word-break: break-all;
  }

  .webhook-payload-content.expanded {
    max-height: none;
  }

  .webhook-details-actions {
    padding: var(--spacing-md);
    border-top: 1px solid var(--border-color);
  }

  @media (max-width: 768px) {
    .webhook-header {
      flex-direction: column;
    }

    .webhook-actions {
      width: 100%;
    }

    .webhook-search {
      flex: 1;
    }

    .webhook-search-input {
      width: 100%;
    }

    .webhook-filter-select {
      flex: 1;
    }
  }
</style>