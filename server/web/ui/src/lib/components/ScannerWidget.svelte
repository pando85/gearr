<script lang="ts">
  import { scannerStore, scannerEnabled, scannerIsScanning, lastScan } from '$lib/stores';
  import { triggerScan } from '$lib/api';
  import { authStore } from '$lib/stores';
  import IconFolderSearch from '$lib/components/icons/IconFolderSearch.svelte';
  import IconRefresh from '$lib/components/icons/IconRefresh.svelte';
  import IconCheckCircleOutline from '$lib/components/icons/IconCheckCircleOutline.svelte';
  import IconHourglass from '$lib/components/icons/IconHourglass.svelte';
  import Spinner from '$lib/components/Spinner.svelte';

  let isTriggeringScan = false;

  async function handleScanNow() {
    const token = $authStore.token;
    if (!token || isTriggeringScan) return;
    
    isTriggeringScan = true;
    try {
      await triggerScan(token);
    } catch (e) {
      console.error('Failed to trigger scan:', e);
    } finally {
      setTimeout(() => {
        isTriggeringScan = false;
      }, 1000);
    }
  }

  function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function formatDuration(ms: number): string {
    const hours = Math.floor(ms / 3600000);
    const minutes = Math.floor((ms % 3600000) / 60000);
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }
</script>

{#if $scannerEnabled}
  <div class="scanner-widget">
    <div class="scanner-header">
      <div class="scanner-title-row">
        <IconFolderSearch class="scanner-icon" />
        <h3 class="scanner-title">Library Scanner</h3>
      </div>
      <button 
        class="scan-button" 
        on:click={handleScanNow}
        disabled={$scannerIsScanning || isTriggeringScan}
      >
        {#if $scannerIsScanning || isTriggeringScan}
          <IconHourglass class="w-4 h-4 animate-spin" />
        {:else}
          <IconRefresh class="w-4 h-4" />
        {/if}
        <span>{($scannerIsScanning || isTriggeringScan) ? 'Scanning...' : 'Scan Now'}</span>
      </button>
    </div>

    <div class="scanner-content">
      {#if $scannerIsScanning}
        <div class="scanning-status">
          <Spinner size="1rem" color="info" />
          <span>Scanning library...</span>
        </div>
      {:else if $lastScan}
        <div class="scan-results">
          <div class="scan-stat">
            <span class="scan-stat-value">{$lastScan.files_found}</span>
            <span class="scan-stat-label">Found</span>
          </div>
          <div class="scan-stat highlight">
            <span class="scan-stat-value">{$lastScan.files_queued}</span>
            <span class="scan-stat-label">Queued</span>
          </div>
          <div class="scan-stat">
            <span class="scan-stat-value">{$lastScan.files_skipped_size + $lastScan.files_skipped_codec + $lastScan.files_skipped_exists}</span>
            <span class="scan-stat-label">Skipped</span>
          </div>
        </div>
        <div class="scan-details">
          {#if $lastScan.files_skipped_size > 0}
            <span class="scan-detail">{$lastScan.files_skipped_size} below size threshold</span>
          {/if}
          {#if $lastScan.files_skipped_codec > 0}
            <span class="scan-detail">{$lastScan.files_skipped_codec} already x265</span>
          {/if}
          {#if $lastScan.files_skipped_exists > 0}
            <span class="scan-detail">{$lastScan.files_skipped_exists} already queued</span>
          {/if}
        </div>
        {#if $lastScan.completed_at}
          <div class="scan-time">
            <IconCheckCircleOutline class="w-4 h-4" />
            <span>Completed {new Date($lastScan.completed_at).toLocaleString()}</span>
          </div>
        {/if}
      {:else}
        <div class="no-scan">
          <p>No scans yet. Click "Scan Now" to start scanning your library.</p>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .scanner-widget {
    background-color: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-lg);
    box-shadow: var(--shadow-sm);
    overflow: hidden;
  }

  .scanner-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--spacing-md) var(--spacing-lg);
    border-bottom: 1px solid var(--border-color);
    background-color: var(--bg-secondary);
  }

  .scanner-title-row {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
  }

  .scanner-icon {
    width: 1.25rem;
    height: 1.25rem;
    color: var(--color-primary);
  }

  .scanner-title {
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    margin: 0;
  }

  .scan-button {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    padding: 0.375rem 0.75rem;
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-medium);
    color: var(--color-primary);
    background-color: transparent;
    border: 1px solid var(--color-primary);
    border-radius: var(--border-radius-md);
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .scan-button:hover:not(:disabled) {
    background-color: var(--color-primary);
    color: white;
  }

  .scan-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .scanner-content {
    padding: var(--spacing-md) var(--spacing-lg);
  }

  .scanning-status {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
    color: var(--color-info);
    font-size: var(--font-size-sm);
  }

  .scan-results {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--spacing-md);
    margin-bottom: var(--spacing-sm);
  }

  .scan-stat {
    text-align: center;
  }

  .scan-stat-value {
    display: block;
    font-size: var(--font-size-xl);
    font-weight: var(--font-weight-bold);
    color: var(--text-primary);
    line-height: 1.2;
  }

  .scan-stat.highlight .scan-stat-value {
    color: var(--color-success);
  }

  .scan-stat-label {
    font-size: var(--font-size-xs);
    color: var(--text-muted);
  }

  .scan-details {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-sm);
    margin-bottom: var(--spacing-sm);
  }

  .scan-detail {
    font-size: var(--font-size-xs);
    color: var(--text-muted);
    padding: 0.125rem 0.375rem;
    background-color: var(--bg-secondary);
    border-radius: var(--border-radius-sm);
  }

  .scan-time {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    font-size: var(--font-size-xs);
    color: var(--text-muted);
  }

  .no-scan {
    text-align: center;
    padding: var(--spacing-md) 0;
  }

  .no-scan p {
    font-size: var(--font-size-sm);
    color: var(--text-muted);
    margin: 0;
  }

  .animate-spin {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>