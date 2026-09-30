<script>
  import { app } from '../lib/state.svelte.js';

  let { title, subtitle = '', children, footer } = $props();

  const brand = $derived(app.session?.brand || 'Login');

  // Cookies only work on the real portal domain, not on the LAN IP.
  const wrongHost = $derived.by(() => {
    const url = app.session?.app_url;
    if (!url) return '';
    try {
      return new URL(url).origin !== location.origin ? url : '';
    } catch {
      return '';
    }
  });
</script>

<main class="relative grid min-h-screen place-items-center overflow-hidden bg-zinc-950 px-4 py-10 text-zinc-100">
  <div class="pointer-events-none absolute inset-x-0 top-0 h-96 bg-gradient-to-b from-indigo-500/10 via-indigo-500/5 to-transparent"></div>

  <div class="relative w-full max-w-sm">
    <div class="mb-8 flex flex-col items-center text-center">
      <div class="grid size-12 place-items-center rounded-2xl bg-indigo-500/15 text-indigo-400 ring-1 ring-indigo-400/20">
        <svg viewBox="0 0 24 24" class="size-6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <rect x="4" y="11" width="16" height="10" rx="2" />
          <path d="M8 11V7a4 4 0 0 1 8 0v4" />
        </svg>
      </div>
      <p class="mt-3 text-sm font-medium tracking-wide text-zinc-400">{brand}</p>
    </div>

    {#if wrongHost}
      <div class="mb-4 rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-200">
        You opened the portal via <span class="font-mono">{location.host}</span>. Logging in only works on
        <a class="font-medium underline underline-offset-2 hover:text-amber-100" href={wrongHost + location.pathname + location.search}>{new URL(wrongHost).host}</a>.
      </div>
    {/if}

    <div class="rounded-2xl border border-zinc-800 bg-zinc-900/70 p-6 shadow-2xl shadow-black/40 backdrop-blur sm:p-8">
      <h1 class="text-xl font-semibold tracking-tight">{title}</h1>
      {#if subtitle}
        <p class="mt-1 text-sm text-zinc-400">{subtitle}</p>
      {/if}
      <div class="mt-6">
        {@render children()}
      </div>
    </div>

    {#if footer}
      <div class="mt-6 text-center text-sm text-zinc-400">
        {@render footer()}
      </div>
    {/if}
  </div>
</main>
