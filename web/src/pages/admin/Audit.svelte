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
    'settings.test_mail': 'Test mail sent',
    'appearance.update': 'Appearance changed',
    'appearance.logo': 'Logo changed',
    'appearance.background': 'Background image changed',
  };
  const tone = (action) =>
    action.endsWith('delete') || action.endsWith('block') || action.endsWith('denied') || action.endsWith('demote') || action.endsWith('remove')
      ? 'bg-rose-400'
      : action.startsWith('settings') || action.startsWith('appearance') || action.endsWith('update')
        ? 'bg-sky-400'
        : 'bg-emerald-400';

  let entries = $state(null);
  let error = $state('');

  api('/admin/audit').then((r) => {
    if (r.ok) entries = r.data.entries;
    else error = r.data.error;
  });
</script>

<h2 class="mb-4 text-lg font-semibold tracking-tight">Audit log</h2>

{#if error}
  <p class="text-sm text-rose-400">{error}</p>
{:else if !entries}
  <p class="text-sm text-zinc-500">Loading…</p>
{:else if entries.length === 0}
  <div class="{card} p-8 text-center text-sm text-zinc-500">Nothing logged yet.</div>
{:else}
  <ul class="{card} divide-y divide-zinc-800/80">
    {#each entries as e (e.id)}
      <li class="flex flex-wrap items-center gap-x-3 gap-y-1 px-4 py-3 text-sm">
        <span class="size-2 shrink-0 rounded-full {tone(e.action)}"></span>
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
