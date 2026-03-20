<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores';
  import SettingsPage from '$lib/components/SettingsPage.svelte';

  if (!$authStore.isAuthenticated) {
    goto('/');
  }
</script>

<SettingsPage />