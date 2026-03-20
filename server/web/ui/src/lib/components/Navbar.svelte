<script lang="ts">
  import { page } from '$app/stores';
  import { themeStore, type ThemeSetting } from '$lib/stores';
  import IconSun from '$lib/components/icons/IconSun.svelte';
  import IconMoon from '$lib/components/icons/IconMoon.svelte';
  import IconBrightness from '$lib/components/icons/IconBrightness.svelte';
  import IconMenu from '$lib/components/icons/IconMenu.svelte';
  import IconClose from '$lib/components/icons/IconClose.svelte';
  import IconWork from '$lib/components/icons/IconWork.svelte';
  import IconPeople from '$lib/components/icons/IconPeople.svelte';
  import IconGitHub from '$lib/components/icons/IconGitHub.svelte';
  import IconDns from '$lib/components/icons/IconDns.svelte';

  let mobileMenuOpen = $state(false);

  function toggleMobileMenu() {
    mobileMenuOpen = !mobileMenuOpen;
  }

  function closeMobileMenu() {
    mobileMenuOpen = false;
  }

  function isActive(path: string) {
    return $page.url.pathname === path;
  }

  const themeButtons: { setting: ThemeSetting; icon: typeof IconSun; label: string }[] = [
    { setting: 'light', icon: IconSun, label: 'Light theme' },
    { setting: 'dark', icon: IconMoon, label: 'Dark theme' },
    { setting: 'auto', icon: IconBrightness, label: 'System theme' },
  ];
</script>

<nav class="navbar">
  <a href="/jobs" class="navbar-brand" onclick={closeMobileMenu}>
    <img src="/logo.svg" alt="Gearr" />
    <span class="navbar-brand-text">Gearr</span>
  </a>

  <ul class="navbar-nav">
    <li>
      <a href="/jobs" class="navbar-link" class:active={isActive('/jobs')}>
        <IconWork class="navbar-link-icon" />
        <span>Jobs</span>
      </a>
    </li>
    <li>
      <a href="/workers" class="navbar-link" class:active={isActive('/workers')}>
        <IconPeople class="navbar-link-icon" />
        <span>Workers</span>
      </a>
    </li>
    <li>
      <a href="/webhooks" class="navbar-link" class:active={isActive('/webhooks')}>
        <IconDns class="navbar-link-icon" />
        <span>Webhooks</span>
      </a>
    </li>
    <li>
      <a
        href="https://github.com/pando85/gearr"
        target="_blank"
        rel="noopener noreferrer"
        class="navbar-link"
      >
        <IconGitHub class="navbar-link-icon" />
        <span class="hide-mobile">GitHub</span>
      </a>
    </li>
  </ul>

  <div class="navbar-actions">
    <div class="navbar-theme-toggle">
      {#each themeButtons as { setting, icon: Icon, label }}
        <button
          class="navbar-theme-btn"
          class:active={$themeStore === setting}
          onclick={() => themeStore.setTheme(setting)}
          title={label}
          aria-label={label}
        >
          <Icon class="w-5 h-5" />
        </button>
      {/each}
    </div>

    <button
      class="navbar-mobile-toggle"
      onclick={toggleMobileMenu}
      aria-label="Toggle menu"
    >
      {#if mobileMenuOpen}
        <IconClose class="w-6 h-6" />
      {:else}
        <IconMenu class="w-6 h-6" />
      {/if}
    </button>
  </div>
</nav>

<div class="navbar-mobile-menu" class:open={mobileMenuOpen}>
  <ul class="navbar-mobile-nav">
    <li>
      <a
        href="/jobs"
        class="navbar-mobile-link"
        class:active={isActive('/jobs')}
        onclick={closeMobileMenu}
      >
        <IconWork class="w-5 h-5" />
        <span>Jobs</span>
      </a>
    </li>
    <li>
      <a
        href="/workers"
        class="navbar-mobile-link"
        class:active={isActive('/workers')}
        onclick={closeMobileMenu}
      >
        <IconPeople class="w-5 h-5" />
        <span>Workers</span>
      </a>
    </li>
    <li>
      <a
        href="/webhooks"
        class="navbar-mobile-link"
        class:active={isActive('/webhooks')}
        onclick={closeMobileMenu}
      >
        <IconDns class="w-5 h-5" />
        <span>Webhooks</span>
      </a>
    </li>
    <li>
      <a
        href="https://github.com/pando85/gearr"
        target="_blank"
        rel="noopener noreferrer"
        class="navbar-mobile-link"
        onclick={closeMobileMenu}
      >
        <IconGitHub class="w-5 h-5" />
        <span>GitHub</span>
      </a>
    </li>
  </ul>
</div>

<style>
  .navbar {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    height: var(--navbar-height);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 1.5rem;
    background-color: var(--bg-navbar);
    border-bottom: 1px solid var(--border-color);
    z-index: 1030;
  }

  .navbar-brand {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: var(--font-size-xl);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
  }

  .navbar-brand img {
    width: 2rem;
    height: 2rem;
  }

  .navbar-brand-text {
    @media (max-width: 768px) {
      display: none;
    }
  }

  .navbar-nav {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    list-style: none;
    margin: 0;
    padding: 0;

    @media (max-width: 768px) {
      display: none;
    }
  }

  .navbar-link {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0.75rem;
    color: var(--text-secondary);
    border-radius: var(--border-radius-md);
    transition: all var(--transition-fast);
  }

  .navbar-link:hover {
    background-color: var(--bg-hover);
    color: var(--text-primary);
  }

  .navbar-link.active {
    background-color: var(--bg-hover);
    color: var(--color-primary);
  }

  .navbar-link-icon {
    width: 1.25rem;
    height: 1.25rem;
  }

  .navbar-actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .navbar-theme-toggle {
    display: flex;
    align-items: center;
    background-color: var(--bg-tertiary);
    border-radius: var(--border-radius-md);
    padding: 0.25rem;
  }

  .navbar-theme-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2rem;
    height: 2rem;
    border: none;
    background: none;
    color: var(--text-muted);
    border-radius: var(--border-radius-sm);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .navbar-theme-btn:hover {
    color: var(--text-primary);
  }

  .navbar-theme-btn.active {
    background-color: var(--bg-card);
    color: var(--color-primary);
    box-shadow: var(--shadow-sm);
  }

  .navbar-mobile-toggle {
    display: none;
    align-items: center;
    justify-content: center;
    width: 2.5rem;
    height: 2.5rem;
    border: none;
    background: none;
    color: var(--text-primary);
    border-radius: var(--border-radius-md);
    cursor: pointer;
  }

  @media (max-width: 768px) {
    .navbar-mobile-toggle {
      display: flex;
    }
  }

  .navbar-mobile-menu {
    position: fixed;
    top: var(--navbar-height);
    left: 0;
    right: 0;
    background-color: var(--bg-navbar);
    border-bottom: 1px solid var(--border-color);
    padding: 0.5rem;
    transform: translateY(-100%);
    opacity: 0;
    visibility: hidden;
    transition: all var(--transition-normal);
    z-index: 1020;
  }

  .navbar-mobile-menu.open {
    transform: translateY(0);
    opacity: 1;
    visibility: visible;
  }

  .navbar-mobile-nav {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .navbar-mobile-link {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem 1rem;
    color: var(--text-secondary);
    border-radius: var(--border-radius-md);
    transition: all var(--transition-fast);
  }

  .navbar-mobile-link:hover {
    background-color: var(--bg-hover);
    color: var(--text-primary);
  }

  .navbar-mobile-link.active {
    background-color: var(--bg-hover);
    color: var(--color-primary);
  }
</style>