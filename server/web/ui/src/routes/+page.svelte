<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores';
  import LoginPage from '$lib/components/LoginPage.svelte';

  if ($authStore.isAuthenticated) {
    goto('/dashboard');
  }
</script>

<LoginPage />