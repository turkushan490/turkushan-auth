<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Alert from '../components/Alert.svelte';
  import Field from '../components/Field.svelte';
  import SubmitButton from '../components/SubmitButton.svelte';
  import { api } from '../lib/api.js';
  import { link, navigate, withRD, toast } from '../lib/state.svelte.js';
  import { t } from '../lib/i18n.svelte.js';

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
  // Only while the tab is visible; coming back to the tab checks right away.
  $effect(() => {
    if (!waiting) return;
    waited = true;
    const timer = setInterval(() => {
      if (document.visibilityState === 'visible') check(true);
    }, 5000);
    const onVisible = () => {
      if (document.visibilityState === 'visible') check(true);
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      clearInterval(timer);
      document.removeEventListener('visibilitychange', onVisible);
    };
  });

  const siteName = $derived(result?.site?.name || host || t('pending.this_site'));

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
    toast(r.ok ? t('pending.resent') : r.data.error, r.ok ? 'success' : 'error');
  }

  function changeAddress() {
    email = result.email || '';
    emailError = '';
    emailStep = 'form';
  }

  const views = {
    ok: { tone: 'emerald', title: 'pending.title_ok', icon: 'check' },
    pending: { tone: 'amber', title: 'pending.title_pending', icon: 'clock' },
    denied: { tone: 'rose', title: 'pending.title_denied', icon: 'x' },
    unknown_site: { tone: 'zinc', title: 'pending.title_unknown', icon: 'question' },
    email_required: { tone: 'sky', title: 'pending.title_email', icon: 'mail' },
    error: { tone: 'rose', title: 'pending.title_error', icon: 'x' },
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
    if (!view) return t('pending.checking_access');
    if (result.status !== 'email_required') return t(view.title);
    return t({ intro: 'pending.title_email', form: 'pending.title_form', sent: 'pending.title_sent' }[emailStep]);
  });

  const primaryBtn = 'flex w-full items-center justify-center rounded-lg bg-indigo-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-400 disabled:opacity-60';
  const secondaryBtn = 'flex w-full items-center justify-center rounded-lg bg-zinc-800 px-4 py-2.5 text-sm font-medium text-zinc-100 transition hover:bg-zinc-700 disabled:opacity-60';
</script>

<AuthLayout {title}>
  {#if !result}
    <p class="text-sm text-zinc-400">{t('common.one_moment')}</p>
  {:else if result.status === 'email_required'}
    <!-- Two clear steps: 1. add your email, 2. click the link in the mail. -->
    <div class="space-y-5">
      <ol class="flex items-center gap-2 text-xs font-medium">
        <li class="flex items-center gap-1.5 {emailStep === 'sent' ? 'text-emerald-400' : 'text-indigo-300'}">
          <span class="grid size-5 place-items-center rounded-full {emailStep === 'sent' ? 'bg-emerald-500/20' : 'bg-indigo-500/25'}">{emailStep === 'sent' ? '✓' : '1'}</span>
          {t('pending.step1')}
        </li>
        <li class="h-px flex-1 bg-zinc-700" aria-hidden="true"></li>
        <li class="flex items-center gap-1.5 {emailStep === 'sent' ? 'text-indigo-300' : 'text-zinc-500'}">
          <span class="grid size-5 place-items-center rounded-full {emailStep === 'sent' ? 'bg-indigo-500/25' : 'bg-zinc-800'}">2</span>
          {t('pending.step2')}
        </li>
      </ol>

      {#if !result.mail_ready}
        <Alert>{t('pending.mail_not_ready')}</Alert>
      {:else if emailStep === 'intro'}
        <p class="text-sm leading-relaxed text-zinc-300">
          {t('pending.intro_a')} <span class="font-semibold text-zinc-100">{siteName}</span> {t('pending.intro_b')}
        </p>
        <button type="button" class={primaryBtn} onclick={() => (emailStep = 'form')}>{t('pending.add_email')}</button>
      {:else if emailStep === 'form'}
        <form class="space-y-4" onsubmit={saveEmail}>
          {#if emailError}<Alert>{emailError}</Alert>{/if}
          <Field label={t('pending.your_email')} type="email" name="email" autocomplete="email" required bind:value={email}
            hint={t('pending.email_hint')} />
          <SubmitButton {busy}>{t('common.send_verification')}</SubmitButton>
        </form>
      {:else}
        <div class="rounded-xl border border-sky-500/25 bg-sky-500/10 p-4 text-sm leading-relaxed text-sky-100">
          <p class="font-semibold">{t('pending.sent_to', { email: result.email })}</p>
          <p class="mt-1 text-sky-200/90">
            {t('pending.open_click_a')} <span class="font-semibold">{t('pending.verify_button')}</span>{t('pending.open_click_b', { name: siteName })}
          </p>
        </div>
        <p class="flex items-center gap-2 text-xs text-zinc-500">
          <svg class="size-3.5 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" class="opacity-25" />
            <path d="M22 12a10 10 0 0 0-10-10" stroke="currentColor" stroke-width="3" stroke-linecap="round" />
          </svg>
          {t('pending.waiting_click')}
        </p>
        <div class="space-y-2">
          <button type="button" class={secondaryBtn} onclick={resend} disabled={busy}>{t('pending.send_again')}</button>
          <button type="button" class="w-full text-center text-sm font-medium text-zinc-400 hover:text-zinc-200" onclick={changeAddress}>
            {t('pending.other_address')}
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
            {t('pending.ok_a')} <span class="font-semibold text-zinc-100">{siteName}</span>{t('pending.ok_b')}
          {:else if result.status === 'pending'}
            {t('pending.pending_a')} <span class="font-semibold text-zinc-100">{siteName}</span> {t('pending.pending_b')}
          {:else if result.status === 'denied'}
            {t('pending.denied_a')} <span class="font-semibold text-zinc-100">{siteName}</span>{t('pending.denied_b')}
          {:else if result.status === 'unknown_site'}
            <span class="font-mono text-zinc-100">{host || t('pending.this_site_start')}</span> {t('pending.unknown')}
            {#if result.is_admin}{t('pending.unknown_admin')}{:else}{t('pending.unknown_ask')}{/if}
          {:else}
            {result.error}
          {/if}
        </p>
      </div>

      <div class="space-y-2">
        {#if result.status === 'ok' && openURL}
          <a href={openURL} class={primaryBtn}>{t('pending.open', { name: siteName })}</a>
        {:else if result.status === 'pending'}
          <button type="button" onclick={() => check()} disabled={checking} class={secondaryBtn}>
            {checking ? t('pending.checking') : t('pending.check_again')}
          </button>
        {:else if result.status === 'unknown_site' && result.is_admin}
          <a href="/admin/sites" onclick={link} class={primaryBtn}>{t('pending.go_sites')}</a>
        {/if}
      </div>
    </div>
  {/if}

  {#snippet footer()}
    <a href="/" onclick={link} class="font-medium text-zinc-400 hover:text-zinc-200">{t('pending.your_account')}</a>
  {/snippet}
</AuthLayout>
