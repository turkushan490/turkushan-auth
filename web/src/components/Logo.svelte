<script>
  // The uploaded logo, or the built-in lock when there is none. size: 'lg' (login pages) | 'sm' (admin header).
  let { appearance, size = 'lg' } = $props();
</script>

{#if appearance?.logo_url}
  <img
    src={appearance.logo_url}
    alt={appearance.site_name || 'Logo'}
    class="object-contain {size === 'lg' ? 'max-h-20 max-w-[14rem]' : 'size-8 shrink-0'}"
  />
{:else}
  <span
    class="grid shrink-0 place-items-center bg-indigo-500/15 text-indigo-400 ring-1 ring-indigo-400/20 {size === 'lg'
      ? 'size-12 rounded-2xl'
      : 'size-8 rounded-lg'}"
  >
    <svg viewBox="0 0 24 24" class={size === 'lg' ? 'size-6' : 'size-4'} fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <rect x="4" y="11" width="16" height="10" rx="2" />
      <path d="M8 11V7a4 4 0 0 1 8 0v4" />
    </svg>
  </span>
{/if}
