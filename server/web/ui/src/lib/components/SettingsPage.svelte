<script lang="ts">
  import { authStore, toastStore } from '$lib/stores';
  import { testWebhook } from '$lib/api';
  import WebhookCard from '$lib/components/WebhookCard.svelte';

  type WebhookSource = 'radarr' | 'sonarr';

  interface WebhookConfig {
    title: string;
    badge: string;
    getPath: () => string;
    instructions: string[];
  }

  const webhookConfigs: Record<WebhookSource, WebhookConfig> = {
    radarr: {
      title: 'Radarr',
      badge: 'Movies',
      getPath: () => getBaseUrl() + '/api/v1/webhook/radarr?source=radarr&event=download',
      instructions: [
        'Go to Settings → Connect in Radarr',
        'Add a new Webhook connection',
        'Paste the URL above as the webhook URL',
        'Set the notification triggers (On Download, On Upgrade)',
        'Save the connection',
      ],
    },
    sonarr: {
      title: 'Sonarr',
      badge: 'TV Shows',
      getPath: () => getBaseUrl() + '/api/v1/webhook/sonarr?source=sonarr&event=download',
      instructions: [
        'Go to Settings → Connect in Sonarr',
        'Add a new Webhook connection',
        'Paste the URL above as the webhook URL',
        'Set the notification triggers (On Download, On Upgrade)',
        'Save the connection',
      ],
    },
  };

  let testingState = $state<Record<WebhookSource, boolean>>({ radarr: false, sonarr: false });
  let testResults = $state<Record<WebhookSource, { success: boolean; message: string } | null>>({
    radarr: null,
    sonarr: null,
  });

  function getBaseUrl(): string {
    if (typeof window !== 'undefined') {
      return `${window.location.protocol}//${window.location.host}`;
    }
    return '';
  }

  async function copyToClipboard(text: string) {
    try {
      await navigator.clipboard.writeText(text);
      toastStore.success('Copied to clipboard');
    } catch {
      toastStore.error('Failed to copy to clipboard');
    }
  }

  async function handleTestWebhook(source: WebhookSource) {
    const token = authStore.getToken();
    if (!token) return;

    testingState[source] = true;
    testResults[source] = null;

    try {
      const result = await testWebhook(token, source);
      testResults[source] = result;
      if (result.success) {
        toastStore.success(result.message);
      } else {
        toastStore.error(result.message);
      }
    } finally {
      testingState[source] = false;
    }
  }
</script>

<div class="settings-page">
  <div class="settings-header">
    <h1 class="settings-title">Settings</h1>
    <p class="settings-subtitle">Configure webhook integration with Radarr and Sonarr</p>
  </div>

  <div class="settings-content">
    <section class="settings-section">
      <h2 class="section-title">Webhook Configuration</h2>
      <p class="section-description">
        Configure your Radarr and Sonarr instances to send webhooks to Gearr when new downloads are completed.
        Copy the webhook URL below and paste it into your Radarr or Sonarr webhook settings.
      </p>

      <div class="webhook-cards">
        {#each Object.entries(webhookConfigs) as [source, config]}
          <WebhookCard
            title={config.title}
            badge={config.badge}
            webhookUrl={config.getPath()}
            instructions={config.instructions}
            testing={testingState[source as WebhookSource]}
            testResult={testResults[source as WebhookSource]}
            onTest={() => handleTestWebhook(source as WebhookSource)}
            onCopy={copyToClipboard}
          />
        {/each}
      </div>
    </section>

    <section class="settings-section">
      <h2 class="section-title">API Key Authentication</h2>
      <div class="info-card">
        <p class="info-text">
          Webhook endpoints can be secured with API key authentication. Configure your API keys
          in the Gearr configuration file or using command-line flags:
        </p>
        <ul class="info-list">
          <li><code>--webhook.enabled=true</code> - Enable webhook authentication</li>
          <li><code>--webhook.radarr.apiKey=YOUR_KEY</code> - Radarr API key</li>
          <li><code>--webhook.sonarr.apiKey=YOUR_KEY</code> - Sonarr API key</li>
        </ul>
        <p class="info-note">
          When API key authentication is enabled, include the key as a query parameter:
          <code>?apiKey=YOUR_KEY</code>
        </p>
      </div>
    </section>
  </div>
</div>

<style>
  .settings-page {
    padding: var(--spacing-lg);
    max-width: 1200px;
    margin: 0 auto;
  }

  .settings-header {
    margin-bottom: var(--spacing-xl);
  }

  .settings-title {
    font-size: var(--font-size-3xl);
    font-weight: var(--font-weight-bold);
    color: var(--text-primary);
    margin-bottom: var(--spacing-xs);
  }

  .settings-subtitle {
    font-size: var(--font-size-base);
    color: var(--text-secondary);
  }

  .settings-section {
    margin-bottom: var(--spacing-2xl);
  }

  .section-title {
    font-size: var(--font-size-xl);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    margin-bottom: var(--spacing-sm);
  }

  .section-description {
    font-size: var(--font-size-sm);
    color: var(--text-secondary);
    margin-bottom: var(--spacing-lg);
    line-height: var(--line-height-relaxed);
  }

  .webhook-cards {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
    gap: var(--spacing-lg);
  }

  @media (max-width: 768px) {
    .webhook-cards {
      grid-template-columns: 1fr;
    }
  }

  .info-card {
    background-color: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-lg);
    padding: var(--spacing-lg);
  }

  .info-text {
    font-size: var(--font-size-sm);
    color: var(--text-secondary);
    margin-bottom: var(--spacing-md);
    line-height: var(--line-height-relaxed);
  }

  .info-list {
    font-size: var(--font-size-sm);
    color: var(--text-primary);
    margin: 0 0 var(--spacing-md) 0;
    padding: 0;
    list-style: none;
  }

  .info-list li {
    padding: var(--spacing-xs) 0;
    font-family: var(--font-mono);
    font-size: var(--font-size-xs);
  }

  .info-list code {
    background-color: var(--bg-tertiary);
    padding: 0.125rem 0.25rem;
    border-radius: var(--border-radius-sm);
  }

  .info-note {
    font-size: var(--font-size-xs);
    color: var(--text-muted);
    font-style: italic;
  }

  .info-note code {
    background-color: var(--bg-tertiary);
    padding: 0.125rem 0.25rem;
    border-radius: var(--border-radius-sm);
    font-style: normal;
  }
</style>