<script lang="ts">
  import IconContentCopy from '$lib/components/icons/IconContentCopy.svelte';
  import IconCheckCircle from '$lib/components/icons/IconCheckCircle.svelte';
  import IconError from '$lib/components/icons/IconError.svelte';
  import Spinner from '$lib/components/Spinner.svelte';

  interface Props {
    title: string;
    badge: string;
    webhookUrl: string;
    instructions: string[];
    testing: boolean;
    testResult: { success: boolean; message: string } | null;
    onTest: () => void;
    onCopy: (text: string) => void;
  }

  let { title, badge, webhookUrl, instructions, testing, testResult, onTest, onCopy }: Props = $props();
</script>

<div class="webhook-card">
  <div class="webhook-card-header">
    <h3 class="webhook-card-title">{title}</h3>
    <span class="badge badge-info">{badge}</span>
  </div>

  <div class="webhook-url-section">
    <div class="webhook-label">Webhook URL</div>
    <div class="webhook-url-container">
      <code class="webhook-url">{webhookUrl}</code>
      <button
        class="copy-btn"
        onclick={() => onCopy(webhookUrl)}
        title="Copy to clipboard"
      >
        <IconContentCopy class="w-4 h-4" />
      </button>
    </div>
  </div>

  <div class="webhook-instructions">
    <h4 class="instructions-title">Setup Instructions</h4>
    <ol class="instructions-list">
      {#each instructions as instruction}
        <li>{instruction}</li>
      {/each}
    </ol>
  </div>

  <div class="webhook-test-section">
    <button
      class="btn btn-secondary test-btn"
      onclick={onTest}
      disabled={testing}
    >
      {#if testing}
        <Spinner size="14px" />
        Testing...
      {:else}
        Test Webhook
      {/if}
    </button>

    {#if testResult}
      <div class="test-result {testResult.success ? 'success' : 'error'}">
        {#if testResult.success}
          <IconCheckCircle class="w-4 h-4" />
        {:else}
          <IconError class="w-4 h-4" />
        {/if}
        <span>{testResult.message}</span>
      </div>
    {/if}
  </div>
</div>

<style>
  .webhook-card {
    background-color: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-lg);
    padding: var(--spacing-lg);
    box-shadow: var(--shadow-sm);
  }

  .webhook-card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-lg);
  }

  .webhook-card-title {
    font-size: var(--font-size-lg);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
  }

  .webhook-url-section {
    margin-bottom: var(--spacing-lg);
  }

  .webhook-label {
    display: block;
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-medium);
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: var(--spacing-xs);
  }

  .webhook-url-container {
    display: flex;
    align-items: stretch;
    gap: var(--spacing-xs);
  }

  .webhook-url {
    flex: 1;
    padding: var(--spacing-sm) var(--spacing-md);
    font-family: var(--font-mono);
    font-size: var(--font-size-xs);
    color: var(--text-primary);
    background-color: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-md);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .copy-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--spacing-sm);
    border: 1px solid var(--border-color);
    background-color: var(--bg-secondary);
    color: var(--text-secondary);
    border-radius: var(--border-radius-md);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .copy-btn:hover {
    background-color: var(--bg-hover);
    color: var(--text-primary);
  }

  .webhook-instructions {
    background-color: var(--bg-secondary);
    border: 1px solid var(--border-color-light);
    border-radius: var(--border-radius-md);
    padding: var(--spacing-md);
    margin-bottom: var(--spacing-lg);
  }

  .instructions-title {
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    margin-bottom: var(--spacing-sm);
  }

  .instructions-list {
    font-size: var(--font-size-xs);
    color: var(--text-secondary);
    padding-left: var(--spacing-lg);
    margin: 0;
  }

  .instructions-list li {
    margin-bottom: var(--spacing-xs);
  }

  .webhook-test-section {
    display: flex;
    align-items: center;
    gap: var(--spacing-md);
    flex-wrap: wrap;
  }

  .test-btn {
    min-width: 140px;
    display: flex;
    align-items: center;
    gap: var(--spacing-xs);
  }

  .test-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .test-result {
    display: flex;
    align-items: center;
    gap: var(--spacing-xs);
    font-size: var(--font-size-sm);
    padding: var(--spacing-xs) var(--spacing-sm);
    border-radius: var(--border-radius-md);
  }

  .test-result.success {
    background-color: rgba(34, 197, 94, 0.1);
    color: var(--color-success);
  }

  .test-result.error {
    background-color: rgba(239, 68, 68, 0.1);
    color: var(--color-error);
  }
</style>