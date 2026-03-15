<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores';
  import JobsPage from '$lib/components/JobsPage.svelte';

  if (!$authStore.isAuthenticated) {
    goto('/');
  }
</script>

<JobsPage />