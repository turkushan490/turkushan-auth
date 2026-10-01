<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Alert from '../components/Alert.svelte';
  import Field from '../components/Field.svelte';
  import SubmitButton from '../components/SubmitButton.svelte';
  import { api } from '../lib/api.js';
  import { link, navigate, withRD, toast } from '../lib/state.svelte.js';

  const params = new URLSearchParams(location.search);
  const host = params.get('site') || '';
  const rd = params.get('rd') || '';

  let result = $state(null);
  let checking = $state(false);
  let waited = false; // true once we showed a waiting screen; then "ok" continues by itself

  // Email step: 'intro' → 'form' → 'sent'
  let emailStep = $state('intro');
  let email = $state('');
  let emailError = $state('');
  let busy = $state(false);

  async function check(silent = false) {
    if (!silent) checking = true;
    const r = await api('/access?site=' + encodeURIComponent(host));
    checking = false;
    if (r.status === 401) {
      navigate(withRD('/login', rd));
      return;
    }
    if (!r.ok) {
      if (!silent) result = { status: 'error', error: r.data.error };
      return;
    }
    const first = result === null;
    result = r.data;
    if (first && result.status === 'email_required' && result.email) emailStep = 'sent';
    if (result.status === 'ok' && waited && openURL) window.location.href = openURL;
  }
  check();

  // While someone waits (for the mail link or for approval), look again every few seconds.
  const waiting = $derived(
    result?.status === 'pending' || (result?.status === 'email_required' && emailStep === 'sent'),
  );
  $effect(() => {
    if (!waiting) return;
    waited = true;
    const timer = setInterval(() => check(true), 5000);
    return () => clearInterval(timer);
  });

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

  async function saveEmail(e) {
    e.preventDefault();
    emailError = '';
    busy = true;
    const r = await api('/account/email', { email, rd: openURL });
    busy = false;
    if (!r.ok) {
      emailError = r.data.error;
      return;
    }
    result.email = email.trim().toLowerCase();
    emailStep = 'sent';
  }

  async function resend() {
    busy = true;
    const r = await api('/account/email/resend', { rd: openURL });
    busy = false;
    toast(r.ok ? 'Mail sent again. Check your inbox.' : r.data.error, r.ok ? 'success' : 'error');
  }

  function changeAddress() {
    email = result.email || '';
    emailError = '';
    emailStep = 'form';
  }

  const views = {
    ok: { tone: 'emerald', title: 'You have access', icon: 'check' },
    pending: { tone: 'amber', title: 'Waiting for approval', icon: 'clock' },
    denied: { tone: 'rose', title: 'No access', icon: 'x' },
    unknown_site: { tone: 'zinc', title: 'Site not set up', icon: 'question' },
    email_required: { tone: 'sky', title: 'Email address needed', icon: 'mail' },
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
  const title = $derived.by(() => {
    if (!view) return 'Checking access…';
    if (result.status !== 'email_required') return view.title;
    return { intro: 'Email address needed', form: 'Add your email address', sent: 'Check your inbox' }[emailStep];
  });

  const primaryBtn = 'flex w-full items-center justify-center rounded-lg bg-indigo-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-400 disabled:opacity-60';
  const secondaryBtn = 'flex w-full items-center justify-center rounded-lg bg-zinc-800 px-4 py-2.5 text-sm font-medium text-zinc-100 transition hover:bg-zinc-700 disabled:opacity-60';
</script>

<AuthLayout {title}>
  {#if !result}
    <p class="text-sm text-zinc-400">One moment…</p>
  {:else if result.status === 'email_required'}
    <!-- Two clear steps: 1. add your email, 2. click the link in the mail. -->
    <div class="space-y-5">
      <ol class="flex items-center gap-2 text-xs font-medium">
        <li class="flex items-center gap-1.5 {emailStep === 'sent' ? 'text-emerald-400' : 'text-indigo-300'}">
          <span class="grid size-5 place-items-center rounded-full {emailStep === 'sent' ? 'bg-emerald-500/20' : 'bg-indigo-500/25'}">{emailStep === 'sent' ? '✓' : '1'}</span>
          Add your email
        </li>
        <li class="h-px flex-1 bg-zinc-700" aria-hidden="true"></li>
        <li class="flex items-center gap-1.5 {emailStep === 'sent' ? 'text-indigo-300' : 'text-zinc-500'}">
          <span class="grid size-5 place-items-center rounded-full {emailStep === 'sent' ? 'bg-indigo-500/25' : 'bg-zinc-800'}">2</span>
          Click the link in the mail
        </li>
      </ol>

      {#if !result.mail_ready}
        <Alert>Email isn't set up on this portal yet, so you can't verify an address. Ask the admin.</Alert>
      {:else if emailStep === 'intro'}
        <p class="text-sm leading-relaxed text-zinc-300">
          To open <span class="font-semibold text-zinc-100">{siteName}</span> you need an email address on your account.
          Add it here and we'll send you a link to confirm it.
        </p>
        <button type="button" class={primaryBtn} onclick={() => (emailStep = 'form')}>Add my email address</button>
      {:else if emailStep === 'form'}
        <form class="space-y-4" onsubmit={saveEmail}>
          {#if emailError}<Alert>{emailError}</Alert>{/if}
          <Field label="Your email address" type="email" name="email" autocomplete="email" required bind:value={email}
            hint="We'll send a mail with a link to this address." />
          <SubmitButton {busy}>Send verification link</SubmitButton>
        </form>
      {:else}
        <div class="rounded-xl border border-sky-500/25 bg-sky-500/10 p-4 text-sm leading-relaxed text-sky-100">
          <p class="font-semibold">We sent a mail to {result.email}</p>
          <p class="mt-1 text-sky-200/90">
            Open it and click <span class="font-semibold">Verify email</span>. You then go on to {siteName} automatically.
          </p>
        </div>
        <p class="flex items-center gap-2 text-xs text-zinc-500">
          <svg class="size-3.5 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" class="opacity-25" />
            <path d="M22 12a10 10 0 0 0-10-10" stroke="currentColor" stroke-width="3" stroke-linecap="round" />
          </svg>
          Waiting for you to click the link… No mail? Check your spam folder.
        </p>
        <div class="space-y-2">
          <button type="button" class={secondaryBtn} onclick={resend} disabled={busy}>Send the mail again</button>
          <button type="button" class="w-full text-center text-sm font-medium text-zinc-400 hover:text-zinc-200" onclick={changeAddress}>
            Use a different address
          </button>
        </div>
      {/if}
    </div>
  {:else}
    <div class="space-y-5">
      <div class="flex items-start gap-4">
        <div class="grid size-11 shrink-0 place-items-center rounded-full ring-1 {toneClass[view.tone]}">
          <svg viewBox="0 0 24 24" class="size-5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            {#if view.icon === 'check'}<path d="M5 12l5 5L20 7" />
            {:else if view.icon === 'clock'}<circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" />
            {:else if view.icon === 'question'}<circle cx="12" cy="12" r="9" /><path d="M9.5 9a2.5 2.5 0 1 1 3.5 2.3c-.6.3-1 .8-1 1.5v.7M12 17h.01" />
            {:else}<circle cx="12" cy="12" r="9" /><path d="M15 9l-6 6M9 9l6 6" />{/if}
          </svg>
        </div>
        <p class="text-sm leading-relaxed text-zinc-300">
          {#if result.status === 'ok'}
            You can open <span class="font-semibold text-zinc-100">{siteName}</span>.
          {:else if result.status === 'pending'}
            Your request to open <span class="font-semibold text-zinc-100">{siteName}</span> has been sent to the admin.
            This page continues automatically as soon as it's approved.
          {:else if result.status === 'denied'}
            The admin has not given you access to <span class="font-semibold text-zinc-100">{siteName}</span>.
          {:else if result.status === 'unknown_site'}
            <span class="font-mono text-zinc-100">{host || 'This site'}</span> is not set up in the login portal.
            {#if result.is_admin}Add it in the admin panel under Sites.{:else}Ask the admin to add it.{/if}
          {:else}
            {result.error}
          {/if}
        </p>
      </div>

      <div class="space-y-2">
        {#if result.status === 'ok' && openURL}
          <a href={openURL} class={primaryBtn}>Open {siteName}</a>
        {:else if result.status === 'pending'}
          <button type="button" onclick={() => check()} disabled={checking} class={secondaryBtn}>
            {checking ? 'Checking…' : 'Check again'}
          </button>
        {:else if result.status === 'unknown_site' && result.is_admin}
          <a href="/admin/sites" onclick={link} class={primaryBtn}>Go to Sites</a>
        {/if}
      </div>
    </div>
  {/if}

  {#snippet footer()}
    <a href="/" onclick={link} class="font-medium text-zinc-400 hover:text-zinc-200">Your account</a>
  {/snippet}
</AuthLayout>
