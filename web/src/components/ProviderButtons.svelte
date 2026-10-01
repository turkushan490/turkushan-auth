<script>
  import { app } from '../lib/state.svelte.js';
  import ProviderIcon from './ProviderIcon.svelte';

  // "Continue with Discord / Google" for the logins the admin has set up.
  // rd: where to go after signing in. link: connect to the signed-in account. only: limit to these providers.
  let { rd = '', link = false, only = null, verb = 'Continue with' } = $props();

  const labels = { discord: 'Discord', google: 'Google' };
  const providers = $derived((app.session?.providers || []).filter((p) => !only || only.includes(p)));

  function href(p) {
    const q = new URLSearchParams();
    if (link) q.set('link', '1');
    if (rd) q.set('rd', rd);
    const qs = q.toString();
    return `/api/oauth/${p}/start${qs ? '?' + qs : ''}`;
  }
</script>

{#if providers.length}
  <div class="space-y-2">
    {#each providers as p}
      <a
        href={href(p)}
        class="flex w-full items-center justify-center gap-2.5 rounded-lg px-4 py-2.5 text-sm font-semibold transition {p === 'discord'
          ? 'bg-[#5865F2] text-white hover:bg-[#4752c4]'
          : 'bg-white text-zinc-800 hover:bg-zinc-200'}"
      >
        <ProviderIcon provider={p} />
        {verb} {labels[p]}
      </a>
    {/each}
  </div>
{/if}
