<script lang="ts">
  let { class: className = '' }: { class?: string } = $props();
</script>

<svg
  class={className}
  xmlns="http://www.w3.org/2000/svg"
  viewBox="0 0 24 24"
  fill="currentColor"
>
  <path d="M20 6h-8l-2-2H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V6h5.17l2 2H20v10zm-3.5-5c0 1.66-1.34 3-3 3s-3-1.34-3-3 1.34-3 3-3 3 1.34 3 3zm1.5 4.5l-2.09-2.09c-.63.37-1.37.59-2.16.59-2.21 0-4-1.79-4-4s1.79-4 4-4 4 1.79 4 4c0 .79-.22 1.53-.59 2.16l2.09 2.09-.75.75-.5-.5z"/>
</svg>