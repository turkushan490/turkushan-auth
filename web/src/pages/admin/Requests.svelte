<script>
  import { api } from '../../lib/api.js';
  import { toast } from '../../lib/state.svelte.js';
  import { timeAgo, fullDate } from '../../lib/format.js';
  import { card, btnApprove, btnDanger, btnGhost, badge, statusBadge } from './ui.js';

  let { data, reload } = $props();

  const pending = $derived(data.access.filter((a) => a.status === 'pending'));
  const decided = $derived(data.access.filter((a) => a.status !== 'pending'));
  let busy = $state('');

  async function decide(entry, status) {
    busy = `${entry.user_id}:${entry.site_id}`;
    const r = await api('/admin/access', { user_id: entry.user_id, site_id: entry.site_id, status }, 'PUT');
    busy = '';
    if (!r.ok) return toast(r.data.error, 'error');
    toast(status === 'approved' ? `${entry.username} can now open ${entry.site_name}.` : `${entry.username} was denied ${entry.site_name}.`);
    reload();
  }
</script>

<section>
  <div class="mb-4 flex items-baseline justify-between">
    <h2 class="text-lg font-semibold tracking-tight">Waiting for approval</h2>
    <span class="text-sm text-zinc-500">{pending.length} open</span>
  </div>

  {#if pending.length === 0}
    <div class="{card} flex flex-col items-center px-6 py-12 text-center">
      <div class="grid size-11 place-items-center rounded-full bg-emerald-500/10 text-emerald-400">
        <svg viewBox="0 0 24 24" class="size-5" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12l5 5L20 7" /></svg>
      </div>
      <p class="mt-3 font-medium">No open requests</p>
      <p class="mt-1 text-sm text-zinc-500">When someone opens a site that needs approval, the request shows up here.</p>
    </div>
  {:else}
    <ul class="space-y-2">
      {#each pending as entry (entry.user_id + ':' + entry.site_id)}
        <li class="{card} flex flex-wrap items-center gap-x-4 gap-y-3 p-4">
          <div class="grid size-10 shrink-0 place-items-center rounded-full bg-amber-500/15 text-sm font-semibold uppercase text-amber-300">
            {entry.username.slice(0, 1)}
          </div>
          <div class="min-w-0 flex-1">
            <p class="text-sm">
              <span class="font-semibold">{entry.username}</span>
              <span class="text-zinc-400">wants to open</span>
              <span class="font-semibold">{entry.site_name}</span>
            </p>
            <p class="mt-0.5 text-xs text-zinc-500" title={fullDate(entry.requested_at)}>
              {entry.site_host} · {timeAgo(entry.requested_at)}
            </p>
          </div>
          <div class="flex w-full gap-2 sm:w-auto">
            <button class="{btnDanger} flex-1 sm:flex-none" disabled={busy !== ''} onclick={() => decide(entry, 'denied')}>Deny</button>
            <button class="{btnApprove} flex-1 sm:flex-none" disabled={busy !== ''} onclick={() => decide(entry, 'approved')}>
              <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12l5 5L20 7" /></svg>
              Approve
            </button>
          </div>
        </li>
      {/each}
    </ul>
  {/if}
</section>

{#if decided.length > 0}
  <section class="mt-12">
    <h2 class="mb-4 text-lg font-semibold tracking-tight">Decided</h2>
    <div class="{card} divide-y divide-zinc-800/80">
      {#each decided as entry (entry.user_id + ':' + entry.site_id)}
        <div class="flex flex-wrap items-center gap-x-4 gap-y-2 px-4 py-3">
          <div class="min-w-0 flex-1 text-sm">
            <span class="font-medium">{entry.username}</span>
            <span class="text-zinc-500">→</span>
            <span class="font-medium">{entry.site_name}</span>
            <span class="text-zinc-500">({entry.site_host})</span>
          </div>
          <span class="{badge} {statusBadge[entry.status]}">{entry.status}</span>
          <span class="w-full text-xs text-zinc-500 sm:w-auto" title={fullDate(entry.decided_at)}>
            by {entry.decided_by || '—'} · {timeAgo(entry.decided_at)}
          </span>
          {#if entry.status === 'approved'}
            <button class={btnGhost} disabled={busy !== ''} onclick={() => decide(entry, 'denied')}>Revoke</button>
          {:else}
            <button class={btnGhost} disabled={busy !== ''} onclick={() => decide(entry, 'approved')}>Approve</button>
          {/if}
        </div>
      {/each}
    </div>
  </section>
{/if}
