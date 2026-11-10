<script>
  import { api } from '../../lib/api.js';
  import { timeAgo, fullDate } from '../../lib/format.js';
  import { card } from './ui.js';
  import { t } from '../../lib/i18n.svelte.js';

  // Each known action has an "audit.action.<action>" text; unknown ones show as is.
  const actions = new Set([
    'admin.bootstrap', 'admin.promote', 'admin.demote', 'user.create',
    'group.create', 'group.update', 'group.delete', 'group.rights', 'group.site',
    'group.member_add', 'group.member_remove', 'group.rule_add', 'group.rule_remove',
    'site.create', 'site.update', 'site.delete', 'access.approved', 'access.denied',
    'user.block', 'user.unblock', 'user.password_reset', 'user.delete',
    'settings.portal_url', 'settings.discord_webhook', 'settings.mail', 'settings.login', 'settings.test_mail',
    'appearance.update', 'appearance.logo', 'appearance.background', 'backup.download',
  ]);
  const actionLabel = (action) => (actions.has(action) ? t(`audit.action.${action}`) : action);
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
  <h2 class="text-lg font-semibold tracking-tight">{t('audit.title')}</h2>
  <div class="w-full sm:w-72">
    <label class="sr-only" for="audit-search">{t('audit.search_label')}</label>
    <input id="audit-search" type="search" bind:value={search} oninput={onsearch} placeholder={t('audit.search_placeholder')}
      class="w-full rounded-lg border border-zinc-800 bg-zinc-950/60 px-3 py-1.5 text-sm outline-none placeholder:text-zinc-500 focus:border-indigo-500" />
  </div>
</div>

{#if error}
  <p class="text-sm text-rose-400">{error}</p>
{:else if !entries}
  <p class="text-sm text-zinc-500">{t('audit.loading')}</p>
{:else if entries.length === 0}
  <div class="{card} p-8 text-center text-sm text-zinc-500">{search.trim() ? t('audit.no_match') : t('audit.empty')}</div>
{:else}
  <ul class="{card} divide-y divide-zinc-800/80">
    {#each entries as e (e.id)}
      <li class="flex flex-wrap items-center gap-x-3 gap-y-1 px-4 py-3 text-sm">
        <span class="size-2 shrink-0 rounded-full {tone(e.action)}" aria-hidden="true"></span>
        <span class="font-medium">{actionLabel(e.action)}</span>
        {#if e.target}<span class="min-w-0 truncate text-zinc-300">{e.target}</span>{/if}
        {#if e.detail}<span class="min-w-0 truncate text-xs text-zinc-500">{e.detail}</span>{/if}
        <span class="ms-auto shrink-0 text-xs text-zinc-500" title={fullDate(e.created_at)}>
          {e.actor}{#if e.ip} · {e.ip}{/if} · {timeAgo(e.created_at)}
        </span>
      </li>
    {/each}
  </ul>
{/if}
{#if more}
  <div class="mt-4 flex justify-center">
    <button class="rounded-lg border border-zinc-700 px-3.5 py-2 text-sm font-medium text-zinc-200 transition hover:bg-zinc-800 disabled:opacity-50" onclick={() => load(true)} disabled={loading}>
      {loading ? t('audit.loading') : t('audit.load_older')}
    </button>
  </div>
{/if}
