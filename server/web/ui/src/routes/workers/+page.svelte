<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores';
  import WorkersPage from '$lib/components/WorkersPage.svelte';

  if (!$authStore.isAuthenticated) {
    goto('/');
  }
</script>

<WorkersPage />