<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores';
  import { fetchWorkers, type Worker } from '$lib/api';
  import IconPeople from '$lib/components/icons/IconPeople.svelte';
  import IconDns from '$lib/components/icons/IconDns.svelte';
  import IconSchedule from '$lib/components/icons/IconSchedule.svelte';

  let workers = $state<Worker[]>([]);
  let loading = $state(true);

  onMount(async () => {
    const token = authStore.getToken();
    if (!token) {
      goto('/');
      return;
    }

    try {
      workers = await fetchWorkers(token);
    } catch (error) {
      authStore.logout();
      goto('/');
    } finally {
      loading = false;
    }
  });

  function getInitials(name: string) {
    return name
      .split('-')
      .map((part) => part[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  }

  function formatLastSeen(lastSeen: string) {
    const date = new Date(lastSeen);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    return `${diffDays}d ago`;
  }
</script>

<div class="workers-page">
  <div class="workers-header">
    <h1 class="workers-title">Workers ({workers.length})</h1>
  </div>

  {#if loading}
    <div class="workers-loading">
      <div class="workers-spinner"></div>
    </div>
  {:else if workers.length > 0}
    <div class="workers-grid">
      {#each workers as worker}
        <div class="worker-card">
          <div class="worker-card-header">
            <div class="worker-avatar">{getInitials(worker.name)}</div>
            <div class="worker-info">
              <div class="worker-name">{worker.name}</div>
              <div class="worker-id">{worker.id.slice(0, 8)}...</div>
            </div>
            <div class="worker-status">
              <span class="worker-status-dot"></span>
              Online
            </div>
          </div>
          <div class="worker-card-body">
            <div class="worker-detail">
              <span class="worker-detail-label">
                <IconDns class="w-4 h-4" />
                Queue
              </span>
              <span class="worker-detail-value">{worker.queue_name}</span>
            </div>
            <div class="worker-detail">
              <span class="worker-detail-label">
                <IconSchedule class="w-4 h-4" />
                Last Seen
              </span>
              <span class="worker-detail-value">{formatLastSeen(worker.last_seen)}</span>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="workers-empty">
      <IconPeople class="workers-empty-icon" />
      <p class="workers-empty-text">No workers available</p>
    </div>
  {/if}
</div>

<style>
  .workers-page {
    padding: var(--spacing-lg);
    max-width: 1400px;
    margin: 0 auto;
  }

  .workers-header {
    margin-bottom: var(--spacing-xl);
  }

  .workers-title {
    font-size: var(--font-size-3xl);
    font-weight: var(--font-weight-bold);
    color: var(--text-primary);
  }

  .workers-loading {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 200px;
  }

  .workers-spinner {
    width: 2rem;
    height: 2rem;
    border: 2px solid var(--border-color);
    border-top-color: var(--color-primary);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  .workers-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: var(--spacing-lg);
  }

  .worker-card {
    background-color: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-lg);
    overflow: hidden;
    box-shadow: var(--shadow-sm);
  }

  .worker-card-header {
    display: flex;
    align-items: center;
    gap: var(--spacing-md);
    padding: var(--spacing-lg);
    border-bottom: 1px solid var(--border-color);
  }

  .worker-avatar {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 3rem;
    height: 3rem;
    background-color: var(--color-primary);
    color: white;
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-semibold);
    border-radius: var(--border-radius-full);
  }

  .worker-info {
    flex: 1;
    min-width: 0;
  }

  .worker-name {
    font-size: var(--font-size-base);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .worker-id {
    font-size: var(--font-size-xs);
    color: var(--text-muted);
    font-family: var(--font-mono);
  }

  .worker-status {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    font-size: var(--font-size-xs);
    color: var(--color-success);
  }

  .worker-status-dot {
    width: 0.5rem;
    height: 0.5rem;
    background-color: var(--color-success);
    border-radius: 50%;
  }

  .worker-card-body {
    padding: var(--spacing-md) var(--spacing-lg);
  }

  .worker-detail {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--spacing-sm) 0;
  }

  .worker-detail-label {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: var(--font-size-sm);
    color: var(--text-secondary);
  }

  .worker-detail-value {
    font-size: var(--font-size-sm);
    color: var(--text-primary);
  }

  .workers-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: var(--spacing-2xl);
    color: var(--text-muted);
  }

  .workers-empty-icon {
    width: 3rem;
    height: 3rem;
    margin-bottom: var(--spacing-md);
  }

  .workers-empty-text {
    font-size: var(--font-size-sm);
  }
</style>