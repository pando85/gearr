<script lang="ts">
  import { jobStore, jobStats } from '$lib/stores';
  import { formatDateShort } from '$lib/utils';
  import IconAssignment from '$lib/components/icons/IconAssignment.svelte';
  import IconHourglass from '$lib/components/icons/IconHourglass.svelte';
  import IconCheckCircleOutline from '$lib/components/icons/IconCheckCircleOutline.svelte';
  import IconErrorOutline from '$lib/components/icons/IconErrorOutline.svelte';
  import IconSchedule from '$lib/components/icons/IconSchedule.svelte';
  import IconArrowForward from '$lib/components/icons/IconArrowForward.svelte';
  import ScannerWidget from '$lib/components/ScannerWidget.svelte';

  const stats = [
    { key: 'total', label: 'Total Jobs', icon: IconAssignment },
    { key: 'progressing', label: 'In Progress', icon: IconHourglass },
    { key: 'completed', label: 'Completed', icon: IconCheckCircleOutline },
    { key: 'failed', label: 'Failed', icon: IconErrorOutline },
    { key: 'queued', label: 'Queued', icon: IconSchedule },
  ];

  function getStatusBadgeClass(status: string) {
    const classes: Record<string, string> = {
      completed: 'badge-success',
      failed: 'badge-error',
      progressing: 'badge-info',
      queued: 'badge-neutral',
    };
    return classes[status] || 'badge-neutral';
  }

  const recentJobs = $derived(
    [...$jobStore.jobs]
      .sort((a, b) => new Date(b.last_update).getTime() - new Date(a.last_update).getTime())
      .slice(0, 5)
  );
</script>

<div class="dashboard">
  <div class="dashboard-header">
    <h1 class="dashboard-title">Dashboard</h1>
    <p class="dashboard-subtitle">Overview of your video encoding jobs</p>
  </div>

  <div class="dashboard-grid">
    {#each stats as stat}
      {@const Icon = stat.icon}
      <div class="stat-card">
        <div class="stat-card-header">
          <div class="stat-card-icon {stat.key}">
            <Icon class="w-5 h-5" />
          </div>
        </div>
        <div class="stat-card-value">{$jobStats[stat.key as keyof typeof $jobStats]}</div>
        <div class="stat-card-label">{stat.label}</div>
      </div>
    {/each}
  </div>

  <div class="dashboard-widgets">
    <ScannerWidget />
  </div>

  <div class="recent-section">
    <div class="recent-header">
      <h2 class="recent-title">Recent Jobs</h2>
      <a href="/jobs" class="recent-link">
        View all
        <IconArrowForward class="w-4 h-4" />
      </a>
    </div>

    {#if recentJobs.length > 0}
      <ul class="recent-list">
        {#each recentJobs as job}
          {@const fileName = job.source_path.split('/').pop() || job.source_path}
          <li class="recent-item">
            <div class="recent-item-content">
              <div class="recent-item-name">{fileName}</div>
              <div class="recent-item-meta">
                {formatDateShort(job.last_update)}
              </div>
            </div>
            <div class="recent-item-status">
              <span class="badge {getStatusBadgeClass(job.status)}">
                {job.status}
              </span>
            </div>
          </li>
        {/each}
      </ul>
    {:else}
      <div class="dashboard-empty">
        <IconAssignment class="dashboard-empty-icon" />
        <p class="dashboard-empty-text">No jobs yet</p>
      </div>
    {/if}
  </div>
</div>

<style>
  .dashboard {
    padding: var(--spacing-lg);
    max-width: 1400px;
    margin: 0 auto;
  }

  .dashboard-header {
    margin-bottom: var(--spacing-xl);
  }

  .dashboard-title {
    font-size: var(--font-size-3xl);
    font-weight: var(--font-weight-bold);
    color: var(--text-primary);
    margin-bottom: var(--spacing-xs);
  }

  .dashboard-subtitle {
    font-size: var(--font-size-base);
    color: var(--text-secondary);
  }

  .dashboard-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: var(--spacing-lg);
    margin-bottom: var(--spacing-2xl);
  }

  .dashboard-widgets {
    margin-bottom: var(--spacing-2xl);
  }

  .stat-card {
    background-color: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-lg);
    padding: var(--spacing-lg);
    box-shadow: var(--shadow-sm);
  }

  .stat-card-header {
    margin-bottom: var(--spacing-md);
  }

  .stat-card-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2.5rem;
    height: 2.5rem;
    border-radius: var(--border-radius-md);
  }

  .stat-card-icon.total {
    background-color: rgba(99, 102, 241, 0.1);
    color: var(--color-primary);
  }

  .stat-card-icon.progressing {
    background-color: rgba(59, 130, 246, 0.1);
    color: var(--color-info);
  }

  .stat-card-icon.completed {
    background-color: rgba(34, 197, 94, 0.1);
    color: var(--color-success);
  }

  .stat-card-icon.failed {
    background-color: rgba(239, 68, 68, 0.1);
    color: var(--color-error);
  }

  .stat-card-icon.queued {
    background-color: rgba(100, 116, 139, 0.1);
    color: var(--color-secondary);
  }

  .stat-card-value {
    font-size: var(--font-size-3xl);
    font-weight: var(--font-weight-bold);
    color: var(--text-primary);
    line-height: 1;
    margin-bottom: var(--spacing-xs);
  }

  .stat-card-label {
    font-size: var(--font-size-sm);
    color: var(--text-secondary);
  }

  .recent-section {
    background-color: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-lg);
    box-shadow: var(--shadow-sm);
  }

  .recent-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--spacing-lg);
    border-bottom: 1px solid var(--border-color);
  }

  .recent-title {
    font-size: var(--font-size-lg);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
  }

  .recent-link {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    font-size: var(--font-size-sm);
    color: var(--color-primary);
  }

  .recent-link:hover {
    color: var(--color-primary-hover);
  }

  .recent-list {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .recent-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--spacing-md) var(--spacing-lg);
    border-bottom: 1px solid var(--border-color-light);
  }

  .recent-item:last-child {
    border-bottom: none;
  }

  .recent-item-content {
    flex: 1;
    min-width: 0;
  }

  .recent-item-name {
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-medium);
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .recent-item-meta {
    font-size: var(--font-size-xs);
    color: var(--text-muted);
    margin-top: 0.25rem;
  }

  .dashboard-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: var(--spacing-2xl);
    color: var(--text-muted);
  }

  .dashboard-empty-icon {
    width: 3rem;
    height: 3rem;
    margin-bottom: var(--spacing-md);
  }

  .dashboard-empty-text {
    font-size: var(--font-size-sm);
  }
</style>