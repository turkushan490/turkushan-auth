<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import { api } from '../lib/api.js';
  import { link, navigate, withRD } from '../lib/state.svelte.js';

  const params = new URLSearchParams(location.search);
  const host = params.get('site') || '';
  const rd = params.get('rd') || '';

  let result = $state(null);
  let checking = $state(false);

  async function check() {
    checking = true;
    const r = await api('/access?site=' + encodeURIComponent(host));
    checking = false;
    if (r.status === 401) {
      navigate(withRD('/login', rd));
      return;
    }
    result = r.ok ? r.data : { status: 'error', error: r.data.error };
  }
  check();

  const siteName = $derived(result?.site?.name || host || 'this site');

  // Only link to a registered site's own host; rd is used if it points there.
  const openURL = $derived.by(() => {
    const siteHost = result?.site?.host;
    if (!siteHost) return '';
    try {
      if (rd && new URL(rd).hostname === siteHost) return rd;
    } catch {}
    return `https://${siteHost}/`;
  });

  const views = {
    ok: { tone: 'emerald', title: 'You have access', icon: 'check' },
    pending: { tone: 'amber', title: 'Waiting for approval', icon: 'clock' },
    denied: { tone: 'rose', title: 'No access', icon: 'x' },
    unknown_site: { tone: 'zinc', title: 'Site not set up', icon: 'question' },
    email_required: { tone: 'sky', title: 'Verified email needed', icon: 'mail' },
    error: { tone: 'rose', title: 'Something went wrong', icon: 'x' },
  };
  const toneClass = {
    emerald: 'bg-emerald-500/15 text-emerald-400 ring-emerald-400/20',
    amber: 'bg-amber-500/15 text-amber-400 ring-amber-400/20',
    rose: 'bg-rose-500/15 text-rose-400 ring-rose-400/20',
    zinc: 'bg-zinc-800 text-zinc-400 ring-zinc-700',
    sky: 'bg-sky-500/15 text-sky-400 ring-sky-400/20',
  };
  const view = $derived(result ? views[result.status] || views.error : null);
</script>

<AuthLayout title={view?.title || 'Checking access…'}>
  {#if !result}
    <p class="text-sm text-zinc-400">One moment…</p>
  {:else}
    <div class="space-y-5">
      <div class="flex items-start gap-4">
        <div class="grid size-11 shrink-0 place-items-center rounded-full ring-1 {toneClass[view.tone]}">
          <svg viewBox="0 0 24 24" class="size-5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            {#if view.icon === 'check'}<path d="M5 12l5 5L20 7" />
            {:else if view.icon === 'clock'}<circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" />
            {:else if view.icon === 'mail'}<rect x="3" y="5" width="18" height="14" rx="2" /><path d="M3 7l9 6 9-6" />
            {:else if view.icon === 'question'}<circle cx="12" cy="12" r="9" /><path d="M9.5 9a2.5 2.5 0 1 1 3.5 2.3c-.6.3-1 .8-1 1.5v.7M12 17h.01" />
            {:else}<circle cx="12" cy="12" r="9" /><path d="M15 9l-6 6M9 9l6 6" />{/if}
          </svg>
        </div>
        <p class="text-sm leading-relaxed text-zinc-300">
          {#if result.status === 'ok'}
            You can open <span class="font-semibold text-zinc-100">{siteName}</span>.
          {:else if result.status === 'pending'}
            Your request to open <span class="font-semibold text-zinc-100">{siteName}</span> has been sent to the admin.
            You can open it as soon as it's approved.
          {:else if result.status === 'denied'}
            The admin has not given you access to <span class="font-semibold text-zinc-100">{siteName}</span>.
          {:else if result.status === 'unknown_site'}
            <span class="font-mono text-zinc-100">{host || 'This site'}</span> is not set up in the login portal.
            {#if result.is_admin}Add it in the admin panel under Sites.{:else}Ask the admin to add it.{/if}
          {:else if result.status === 'email_required'}
            <span class="font-semibold text-zinc-100">{siteName}</span> is only available for accounts with a verified email address.
            Email verification is coming soon.
          {:else}
            {result.error}
          {/if}
        </p>
      </div>

      <div class="space-y-2">
        {#if result.status === 'ok' && openURL}
          <a href={openURL} class="flex w-full items-center justify-center rounded-lg bg-indigo-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-400">
            Open {siteName}
          </a>
        {:else if result.status === 'pending'}
          <button type="button" onclick={check} disabled={checking}
            class="flex w-full items-center justify-center rounded-lg bg-zinc-800 px-4 py-2.5 text-sm font-medium text-zinc-100 transition hover:bg-zinc-700 disabled:opacity-60">
            {checking ? 'Checking…' : 'Check again'}
          </button>
        {:else if result.status === 'unknown_site' && result.is_admin}
          <a href="/admin/sites" onclick={link} class="flex w-full items-center justify-center rounded-lg bg-indigo-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-400">
            Go to Sites
          </a>
        {/if}
        <a href="/" onclick={link} class="flex w-full items-center justify-center rounded-lg px-4 py-2.5 text-sm font-medium text-zinc-400 transition hover:text-zinc-200">
          Your account
        </a>
      </div>
    </div>
  {/if}
</AuthLayout>
