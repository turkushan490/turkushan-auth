<script>
  import { api } from '../../lib/api.js';
  import { timeAgo, fullDate } from '../../lib/format.js';
  import { card } from './ui.js';

  const labels = {
    'admin.bootstrap': 'Admin account created',
    'admin.promote': 'Made admin',
    'admin.demote': 'Admin removed',
    'user.create': 'User created',
    'group.create': 'Group created',
    'group.update': 'Group renamed',
    'group.delete': 'Group deleted',
    'group.rights': 'Group rights changed',
    'group.site': 'Group site rights changed',
    'group.member_add': 'Added to group',
    'group.member_remove': 'Removed from group',
    'group.rule_add': 'Auto-add rule added',
    'group.rule_remove': 'Auto-add rule removed',
    'site.create': 'Site added',
    'site.update': 'Site changed',
    'site.delete': 'Site removed',
    'access.approved': 'Access given',
    'access.denied': 'Access denied',
    'user.block': 'User blocked',
    'user.unblock': 'User unblocked',
    'user.password_reset': 'Password reset',
    'user.delete': 'User deleted',
    'settings.portal_url': 'Portal address changed',
    'settings.discord_webhook': 'Discord webhook changed',
    'settings.mail': 'Email settings changed',
    'settings.login': 'Login method changed',
    'settings.test_mail': 'Test mail sent',
    'appearance.update': 'Appearance changed',
    'appearance.logo': 'Logo changed',
    'appearance.background': 'Background image changed',
    'backup.download': 'Backup downloaded',
  };
  const tone = (action) =>
    action.endsWith('delete') || action.endsWith('block') || action.endsWith('denied') || action.endsWith('demote') || action.endsWith('remove')
      ? 'bg-rose-400'
      : action.startsWith('settings') || action.startsWith('appearance') || action.endsWith('update')
        ? 'bg-sky-400'
        : 'bg-emerald-400';

  let entries = $state(null);
  let more = $state(false);
  let loading = $state(false);
  let error = $state('');
  let search = $state('');

  // older = true appends the next page; otherwise starts over (new search).
  async function load(older = false) {
    loading = true;
    const p = new URLSearchParams();
    if (search.trim()) p.set('q', search.trim());
    if (older && entries?.length) p.set('before', entries[entries.length - 1].id);
    const r = await api('/admin/audit?' + p);
    loading = false;
    if (!r.ok) {
      error = r.data.error;
      return;
    }
    error = '';
    entries = older ? [...entries, ...r.data.entries] : r.data.entries;
    more = r.data.more;
  }
  load();

  // Search after a short pause in typing.
  let timer;
  function onsearch() {
    clearTimeout(timer);
    timer = setTimeout(() => load(), 300);
  }
</script>

<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
  <h2 class="text-lg font-semibold tracking-tight">Audit log</h2>
  <div class="w-full sm:w-72">
    <label class="sr-only" for="audit-search">Search the audit log</label>
    <input id="audit-search" type="search" bind:value={search} oninput={onsearch} placeholder="Search who, what or which…"
      class="w-full rounded-lg border border-zinc-800 bg-zinc-950/60 px-3 py-1.5 text-sm outline-none placeholder:text-zinc-500 focus:border-indigo-500" />
  </div>
</div>

{#if error}
  <p class="text-sm text-rose-400">{error}</p>
{:else if !entries}
  <p class="text-sm text-zinc-500">Loading…</p>
{:else if entries.length === 0}
  <div class="{card} p-8 text-center text-sm text-zinc-500">{search.trim() ? 'Nothing matches that search.' : 'Nothing logged yet.'}</div>
{:else}
  <ul class="{card} divide-y divide-zinc-800/80">
    {#each entries as e (e.id)}
      <li class="flex flex-wrap items-center gap-x-3 gap-y-1 px-4 py-3 text-sm">
        <span class="size-2 shrink-0 rounded-full {tone(e.action)}" aria-hidden="true"></span>
        <span class="font-medium">{labels[e.action] || e.action}</span>
        {#if e.target}<span class="min-w-0 truncate text-zinc-300">{e.target}</span>{/if}
        {#if e.detail}<span class="min-w-0 truncate text-xs text-zinc-500">{e.detail}</span>{/if}
        <span class="ml-auto shrink-0 text-xs text-zinc-500" title={fullDate(e.created_at)}>
          {e.actor}{#if e.ip} · {e.ip}{/if} · {timeAgo(e.created_at)}
        </span>
      </li>
    {/each}
  </ul>
{/if}
{#if more}
  <div class="mt-4 flex justify-center">
    <button class="rounded-lg border border-zinc-700 px-3.5 py-2 text-sm font-medium text-zinc-200 transition hover:bg-zinc-800 disabled:opacity-50" onclick={() => load(true)} disabled={loading}>
      {loading ? 'Loading…' : 'Load older entries'}
    </button>
  </div>
{/if}
