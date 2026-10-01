<script>
  import { app, look } from '../lib/state.svelte.js';
  import Logo from './Logo.svelte';

  let { title, subtitle = '', children, footer } = $props();

  const brand = $derived(look().site_name || app.session?.brand || 'Login');

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

<main class="relative grid min-h-screen place-items-center px-4 py-10 text-zinc-100">

  <div class="relative w-full max-w-sm">
    <div class="mb-8 flex flex-col items-center text-center">
      <Logo appearance={look()} />
      <p class="mt-3 text-sm font-medium tracking-wide text-zinc-300 [text-shadow:0_1px_6px_rgb(0_0_0/0.7)]">{brand}</p>
    </div>

    {#if wrongHost}
      <div class="mb-4 rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-200">
        You opened the portal via <span class="font-mono">{location.host}</span>. Logging in only works on
        <a class="font-medium underline underline-offset-2 hover:text-amber-100" href={wrongHost + location.pathname + location.search}>{new URL(wrongHost).host}</a>.
      </div>
    {/if}

    <div class="rounded-2xl border border-zinc-800 bg-zinc-900/80 p-6 shadow-2xl shadow-black/40 backdrop-blur-md sm:p-8">
      <h1 class="text-xl font-semibold tracking-tight">{title}</h1>
      {#if subtitle}
        <p class="mt-1 text-sm text-zinc-400">{subtitle}</p>
      {/if}
      <div class="mt-6">
        {@render children()}
      </div>
    </div>

    {#if footer}
      <div class="mt-6 text-center text-sm text-zinc-300 [text-shadow:0_1px_6px_rgb(0_0_0/0.7)]">
        {@render footer()}
      </div>
    {/if}
  </div>
</main>
