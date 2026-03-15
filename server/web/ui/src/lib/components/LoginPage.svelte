<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores';
  import IconVisibility from '$lib/components/icons/IconVisibility.svelte';
  import IconVisibilityOff from '$lib/components/icons/IconVisibilityOff.svelte';
  import IconErrorOutline from '$lib/components/icons/IconErrorOutline.svelte';

  let { errorText = '' }: { errorText?: string } = $props();

  let token = $state('');
  let showToken = $state(false);

  function handleSubmit(event: Event) {
    event.preventDefault();
    if (token) {
      authStore.login(token);
      goto('/dashboard');
    }
  }
</script>

<div class="login-page">
  <div class="login-card">
    <div class="login-header">
      <div class="login-logo">
        <img src="/logo.svg" alt="Gearr" />
        <span>Gearr</span>
      </div>
      <div class="login-subtitle">Video Encoding Management</div>
    </div>

    <form class="login-body" onsubmit={handleSubmit}>
      {#if errorText}
        <div class="login-error animate-slide-down">
          <IconErrorOutline class="login-error-icon" />
          <div class="login-error-content">
            <div class="login-error-title">Login Failed</div>
            <div class="login-error-message">{errorText}</div>
          </div>
        </div>
      {/if}

      <div class="login-field">
        <label class="login-label" for="token">Authentication Token</label>
        <div class="login-input-wrapper">
          <input
            id="token"
            class="login-input"
            type={showToken ? 'text' : 'password'}
            bind:value={token}
            placeholder="Enter your token"
            autocomplete="off"
          />
          <button
            type="button"
            class="login-toggle-visibility"
            onclick={() => showToken = !showToken}
            tabindex={-1}
          >
            {#if showToken}
              <IconVisibilityOff class="w-5 h-5" />
            {:else}
              <IconVisibility class="w-5 h-5" />
            {/if}
          </button>
        </div>
      </div>

      <button class="login-submit" type="submit">Sign In</button>
    </form>

    <div class="login-footer">
      <p class="login-footer-text">
        Need help?
        <a
          href="https://github.com/pando85/gearr"
          target="_blank"
          rel="noopener noreferrer"
          class="login-footer-link"
        >
          Documentation
        </a>
      </p>
    </div>
  </div>
</div>

<style>
  .login-page {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: calc(100vh - var(--navbar-height));
    padding: var(--spacing-lg);
  }

  .login-card {
    width: 100%;
    max-width: 28rem;
    background-color: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-xl);
    box-shadow: var(--shadow-lg);
    overflow: hidden;
  }

  .login-header {
    padding: var(--spacing-xl);
    text-align: center;
    background: linear-gradient(135deg, var(--color-primary), var(--color-primary-hover));
  }

  .login-logo {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
    color: white;
    font-size: var(--font-size-2xl);
    font-weight: var(--font-weight-bold);
  }

  .login-logo img {
    width: 2.5rem;
    height: 2.5rem;
  }

  .login-subtitle {
    margin-top: 0.5rem;
    color: rgba(255, 255, 255, 0.8);
    font-size: var(--font-size-sm);
  }

  .login-body {
    padding: var(--spacing-xl);
  }

  .login-error {
    display: flex;
    align-items: flex-start;
    gap: 0.75rem;
    padding: 0.75rem;
    margin-bottom: var(--spacing-lg);
    background-color: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.2);
    border-radius: var(--border-radius-md);
  }

  .login-error-icon {
    flex-shrink: 0;
    width: 1.25rem;
    height: 1.25rem;
    color: var(--color-error);
  }

  .login-error-title {
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-medium);
    color: var(--color-error);
  }

  .login-error-message {
    font-size: var(--font-size-xs);
    color: var(--text-secondary);
    margin-top: 0.25rem;
  }

  .login-field {
    margin-bottom: var(--spacing-lg);
  }

  .login-label {
    display: block;
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-medium);
    color: var(--text-secondary);
    margin-bottom: var(--spacing-xs);
  }

  .login-input-wrapper {
    position: relative;
  }

  .login-input {
    width: 100%;
    padding: 0.75rem 3rem 0.75rem 1rem;
    font-size: var(--font-size-base);
    color: var(--text-primary);
    background-color: var(--bg-input);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-md);
    transition: all var(--transition-fast);
  }

  .login-input:focus {
    outline: none;
    border-color: var(--color-primary);
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
  }

  .login-input::placeholder {
    color: var(--text-muted);
  }

  .login-toggle-visibility {
    position: absolute;
    right: 0.5rem;
    top: 50%;
    transform: translateY(-50%);
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2rem;
    height: 2rem;
    border: none;
    background: none;
    color: var(--text-muted);
    cursor: pointer;
    border-radius: var(--border-radius-md);
  }

  .login-toggle-visibility:hover {
    color: var(--text-primary);
  }

  .login-submit {
    width: 100%;
    padding: 0.75rem 1.5rem;
    font-size: var(--font-size-base);
    font-weight: var(--font-weight-medium);
    color: white;
    background-color: var(--color-primary);
    border: none;
    border-radius: var(--border-radius-md);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .login-submit:hover {
    background-color: var(--color-primary-hover);
  }

  .login-footer {
    padding: var(--spacing-md) var(--spacing-xl);
    text-align: center;
    background-color: var(--bg-secondary);
    border-top: 1px solid var(--border-color);
  }

  .login-footer-text {
    font-size: var(--font-size-sm);
    color: var(--text-secondary);
  }

  .login-footer-link {
    color: var(--color-primary);
  }

  .login-footer-link:hover {
    color: var(--color-primary-hover);
  }
</style>