<script>
  import { app, link, look, navigate } from '../../lib/state.svelte.js';
  import Logo from '../../components/Logo.svelte';
  import Appearance from './Appearance.svelte';
  import Groups from './Groups.svelte';
  import { visibleTabs } from './rights.js';
  import { api } from '../../lib/api.js';
  import Requests from './Requests.svelte';
  import Users from './Users.svelte';
  import Sites from './Sites.svelte';
  import Settings from './Settings.svelte';
  import Audit from './Audit.svelte';
  import { t } from '../../lib/i18n.svelte.js';

  // label is a translation key, translated in the template so it follows language changes.
  const tabs = [
    { id: 'requests', label: 'admin.tab_requests', path: '/admin' },
    { id: 'users', label: 'admin.tab_users', path: '/admin/users' },
    { id: 'groups', label: 'admin.tab_groups', path: '/admin/groups' },
    { id: 'sites', label: 'admin.tab_sites', path: '/admin/sites' },
    { id: 'appearance', label: 'admin.tab_appearance', path: '/admin/appearance' },
    { id: 'settings', label: 'admin.tab_settings', path: '/admin/settings' },
    { id: 'audit', label: 'admin.tab_audit', path: '/admin/audit' },
  ];
  // Only the tabs this person has rights for; the first one is the start page.
  const myTabs = $derived(data ? tabs.filter((tb) => visibleTabs(data).includes(tb.id)) : []);
  const tab = $derived.by(() => {
    const wanted = tabs.find((tb) => tb.path === app.path.replace(/\/$/, ''))?.id;
    if (myTabs.some((tb) => tb.id === wanted)) return wanted;
    return myTabs[0]?.id || '';
  });

  let data = $state(null);
  let error = $state('');

  async function load() {
    const r = await api('/admin/data');
    if (r.ok) {
      data = r.data;
      error = '';
    } else if (r.status === 401) {
      navigate('/login');
    } else {
      error = r.data.error;
    }
  }
  load();

  const counts = $derived({
    requests: data?.access.filter((a) => a.status === 'pending').length || 0,
    users: data?.users.length || 0,
    groups: (data?.groups.length || 0) + 1,
    sites: data?.sites.length || 0,
  });
</script>

<!-- A custom background stays visible behind the panel, dimmed so tables remain readable. -->
<div class="min-h-screen text-zinc-100 {look().bg_type && look().bg_type !== 'default' ? 'bg-zinc-950/75' : ''}">
  <header class="sticky top-0 z-40 border-b border-zinc-800/80 bg-zinc-950/80 backdrop-blur">
    <div class="mx-auto flex h-14 max-w-5xl items-center justify-between gap-4 px-4">
      <a href="/" onclick={link} class="flex min-w-0 items-center gap-2.5">
        <Logo appearance={look()} size="sm" />
        <span class="truncate text-sm font-semibold">{look().site_name || app.session?.brand}</span>
        <span class="rounded-full bg-zinc-800 px-2 py-0.5 text-xs font-medium text-zinc-400">{t('admin.badge')}</span>
      </a>
      <div class="flex items-center gap-3 text-sm">
        <span class="hidden text-zinc-400 sm:inline">{app.session?.user?.username}</span>
        <a href="/logout" onclick={link} class="rounded-lg px-2.5 py-1.5 text-zinc-400 transition hover:bg-zinc-800 hover:text-zinc-100">{t('admin.sign_out')}</a>
      </div>
    </div>
    <nav class="mx-auto flex max-w-5xl gap-1 overflow-x-auto px-4" aria-label={t('admin.sections')}>
      {#each myTabs as tb}
        <a
          href={tb.id === myTabs[0]?.id ? '/admin' : tb.path}
          onclick={link}
          aria-current={tab === tb.id ? 'page' : undefined}
          class="relative flex shrink-0 items-center gap-2 px-3 py-3 text-sm font-medium transition {tab === tb.id ? 'text-zinc-100' : 'text-zinc-500 hover:text-zinc-300'}"
        >
          {t(tb.label)}
          {#if tb.id === 'requests' && counts.requests > 0}
            <span class="rounded-full bg-amber-500/20 px-1.5 text-xs font-semibold text-amber-300">{counts.requests}</span>
          {:else if (tb.id === 'users' || tb.id === 'sites' || tb.id === 'groups') && data}
            <span class="text-xs text-zinc-600">{counts[tb.id]}</span>
          {/if}
          {#if tab === tb.id}
            <span class="absolute inset-x-2 -bottom-px h-0.5 rounded-full bg-indigo-500"></span>
          {/if}
        </a>
      {/each}
    </nav>
  </header>

  <main class="mx-auto max-w-5xl px-4 py-8">
    {#if error}
      <div class="rounded-xl border border-rose-500/30 bg-rose-500/10 p-6 text-sm text-rose-200">
        {error}
        <a href="/" onclick={link} class="ms-2 underline underline-offset-2">{t('admin.back')}</a>
      </div>
    {:else if !data}
      <div class="grid place-items-center py-24">
        <svg class="size-6 animate-spin text-zinc-600" viewBox="0 0 24 24" fill="none" aria-label={t('admin.loading')}>
          <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" class="opacity-25" />
          <path d="M22 12a10 10 0 0 0-10-10" stroke="currentColor" stroke-width="3" stroke-linecap="round" />
        </svg>
      </div>
    {:else if tab === 'requests'}
      <Requests {data} reload={load} />
    {:else if tab === 'users'}
      <Users {data} reload={load} />
    {:else if tab === 'groups'}
      <Groups {data} reload={load} />
    {:else if tab === 'sites'}
      <Sites {data} reload={load} />
    {:else if tab === 'appearance'}
      <Appearance />
    {:else if tab === 'settings'}
      <Settings {data} reload={load} />
    {:else if tab === 'audit'}
      <Audit />
    {/if}
  </main>
  {#if data?.version}
    <footer class="mx-auto max-w-5xl px-4 pb-8 text-xs text-zinc-500">
      turkushan-auth {data.version === 'dev' ? t('admin.dev_build') : t('admin.version', { version: data.version })}
    </footer>
  {/if}
</div>
