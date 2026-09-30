<script>
  import { app, link, navigate } from '../../lib/state.svelte.js';
  import { api } from '../../lib/api.js';
  import Requests from './Requests.svelte';
  import Users from './Users.svelte';
  import Sites from './Sites.svelte';
  import Settings from './Settings.svelte';
  import Audit from './Audit.svelte';

  const tabs = [
    { id: 'requests', label: 'Requests', path: '/admin' },
    { id: 'users', label: 'Users', path: '/admin/users' },
    { id: 'sites', label: 'Sites', path: '/admin/sites' },
    { id: 'settings', label: 'Settings', path: '/admin/settings' },
    { id: 'audit', label: 'Audit log', path: '/admin/audit' },
  ];
  const tab = $derived(tabs.find((t) => t.path === app.path.replace(/\/$/, ''))?.id || 'requests');

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
    sites: data?.sites.length || 0,
  });
</script>

<div class="min-h-screen bg-zinc-950 text-zinc-100">
  <header class="sticky top-0 z-40 border-b border-zinc-800/80 bg-zinc-950/80 backdrop-blur">
    <div class="mx-auto flex h-14 max-w-5xl items-center justify-between gap-4 px-4">
      <a href="/" onclick={link} class="flex min-w-0 items-center gap-2.5">
        <span class="grid size-8 shrink-0 place-items-center rounded-lg bg-indigo-500/15 text-indigo-400 ring-1 ring-indigo-400/20">
          <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <rect x="4" y="11" width="16" height="10" rx="2" />
            <path d="M8 11V7a4 4 0 0 1 8 0v4" />
          </svg>
        </span>
        <span class="truncate text-sm font-semibold">{app.session?.brand}</span>
        <span class="rounded-full bg-zinc-800 px-2 py-0.5 text-xs font-medium text-zinc-400">Admin</span>
      </a>
      <div class="flex items-center gap-3 text-sm">
        <span class="hidden text-zinc-400 sm:inline">{app.session?.user?.username}</span>
        <a href="/logout" onclick={link} class="rounded-lg px-2.5 py-1.5 text-zinc-400 transition hover:bg-zinc-800 hover:text-zinc-100">Sign out</a>
      </div>
    </div>
    <nav class="mx-auto flex max-w-5xl gap-1 overflow-x-auto px-4" aria-label="Admin sections">
      {#each tabs as t}
        <a
          href={t.path}
          onclick={link}
          aria-current={tab === t.id ? 'page' : undefined}
          class="relative flex shrink-0 items-center gap-2 px-3 py-3 text-sm font-medium transition {tab === t.id ? 'text-zinc-100' : 'text-zinc-500 hover:text-zinc-300'}"
        >
          {t.label}
          {#if t.id === 'requests' && counts.requests > 0}
            <span class="rounded-full bg-amber-500/20 px-1.5 text-xs font-semibold text-amber-300">{counts.requests}</span>
          {:else if (t.id === 'users' || t.id === 'sites') && data}
            <span class="text-xs text-zinc-600">{counts[t.id]}</span>
          {/if}
          {#if tab === t.id}
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
        <a href="/" onclick={link} class="ml-2 underline underline-offset-2">Back</a>
      </div>
    {:else if !data}
      <div class="grid place-items-center py-24">
        <svg class="size-6 animate-spin text-zinc-600" viewBox="0 0 24 24" fill="none" aria-label="Loading">
          <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" class="opacity-25" />
          <path d="M22 12a10 10 0 0 0-10-10" stroke="currentColor" stroke-width="3" stroke-linecap="round" />
        </svg>
      </div>
    {:else if tab === 'requests'}
      <Requests {data} reload={load} />
    {:else if tab === 'users'}
      <Users {data} reload={load} />
    {:else if tab === 'sites'}
      <Sites {data} reload={load} />
    {:else if tab === 'settings'}
      <Settings {data} reload={load} />
    {:else}
      <Audit />
    {/if}
  </main>
</div>
