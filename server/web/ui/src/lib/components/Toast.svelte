<script lang="ts">
  import { toastStore } from '$lib/stores';
  import IconCheckCircle from '$lib/components/icons/IconCheckCircle.svelte';
  import IconError from '$lib/components/icons/IconError.svelte';
  import IconWarning from '$lib/components/icons/IconWarning.svelte';
  import IconInfo from '$lib/components/icons/IconInfo.svelte';
  import IconClose from '$lib/components/icons/IconClose.svelte';
</script>

<div class="toast-container">
  {#each $toastStore as toast (toast.id)}
    {@const icons = {
      success: IconCheckCircle,
      error: IconError,
      warning: IconWarning,
      info: IconInfo,
    }}
    {@const Icon = icons[toast.type]}
    <div class="toast toast-{toast.type}" role="alert">
      <Icon class="toast-icon" />
      <div class="toast-content">
        {#if toast.title}
          <div class="toast-title">{toast.title}</div>
        {/if}
        <div class="toast-message">{toast.message}</div>
      </div>
      <button class="toast-close" onclick={() => toastStore.remove(toast.id)} aria-label="Close">
        <IconClose class="w-4 h-4" />
      </button>
    </div>
  {/each}
</div>

<style>
  .toast-container {
    position: fixed;
    bottom: 1rem;
    right: 1rem;
    z-index: 1080;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    max-width: 24rem;
  }

  .toast {
    display: flex;
    align-items: flex-start;
    gap: 0.75rem;
    padding: 0.875rem 1rem;
    background-color: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-lg);
    box-shadow: var(--shadow-lg);
    animation: slideUp var(--transition-normal);
  }

  .toast-success {
    border-left: 4px solid var(--color-success);
  }

  .toast-error {
    border-left: 4px solid var(--color-error);
  }

  .toast-warning {
    border-left: 4px solid var(--color-warning);
  }

  .toast-info {
    border-left: 4px solid var(--color-info);
  }

  .toast-icon {
    flex-shrink: 0;
    width: 1.25rem;
    height: 1.25rem;
  }

  .toast-success .toast-icon {
    color: var(--color-success);
  }

  .toast-error .toast-icon {
    color: var(--color-error);
  }

  .toast-warning .toast-icon {
    color: var(--color-warning);
  }

  .toast-info .toast-icon {
    color: var(--color-info);
  }

  .toast-content {
    flex: 1;
    min-width: 0;
  }

  .toast-title {
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    margin-bottom: 0.25rem;
  }

  .toast-message {
    font-size: var(--font-size-sm);
    color: var(--text-secondary);
    word-wrap: break-word;
  }

  .toast-close {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.5rem;
    height: 1.5rem;
    border: none;
    background: none;
    color: var(--text-muted);
    cursor: pointer;
    border-radius: var(--border-radius-md);
    transition: all var(--transition-fast);
  }

  .toast-close:hover {
    background-color: var(--bg-hover);
    color: var(--text-primary);
  }
</style>