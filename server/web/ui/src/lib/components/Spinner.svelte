<script lang="ts">
  interface Props {
    size?: string;
    color?: 'primary' | 'info';
  }

  let { size = '1rem', color = 'primary' }: Props = $props();
</script>

<div
  class="spinner"
  class:spinner-primary={color === 'primary'}
  class:spinner-info={color === 'info'}
  style="width: {size}; height: {size};"
></div>

<style>
  .spinner {
    border: 2px solid var(--border-color);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  .spinner-primary {
    border-top-color: var(--color-primary);
  }

  .spinner-info {
    border-top-color: var(--color-info);
    border-color: var(--color-info);
    border-top-color: transparent;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>