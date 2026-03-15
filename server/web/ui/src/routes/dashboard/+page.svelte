<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores';
  import Dashboard from '$lib/components/Dashboard.svelte';

  if (!$authStore.isAuthenticated) {
    goto('/');
  }
</script>

<Dashboard />