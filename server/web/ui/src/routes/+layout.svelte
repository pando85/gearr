<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { themeStore, applyTheme, authStore } from '$lib/stores';
  import Navbar from '$lib/components/Navbar.svelte';
  import Toast from '$lib/components/Toast.svelte';

  let { children } = $props();

  onMount(() => {
    const updateTheme = () => {
      const setting = $themeStore;
      const resolved = themeStore.resolve(setting);
      applyTheme(resolved);
    };

    updateTheme();

    const unsubscribe = themeStore.subscribe(updateTheme);
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    mediaQuery.addEventListener('change', updateTheme);

    return () => {
      unsubscribe();
      mediaQuery.removeEventListener('change', updateTheme);
    };
  });
</script>

<div class="page">
  {#if $authStore.isAuthenticated}
    <Navbar />
  {/if}
  {@render children()}
  <Toast />
</div>