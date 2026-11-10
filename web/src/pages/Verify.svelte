<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Alert from '../components/Alert.svelte';
  import { api } from '../lib/api.js';
  import { app, link, refreshSession } from '../lib/state.svelte.js';
  import { t } from '../lib/i18n.svelte.js';

  const params = new URLSearchParams(location.search);
  const token = params.get('token') || '';

  // Only continue to an https address on the portal's own domain.
  function safeNext(rd) {
    const base = app.session?.brand;
    try {
      const u = new URL(rd);
      if (base && u.protocol === 'https:' && (u.hostname === base || u.hostname.endsWith('.' + base))) return u.href;
    } catch {}
    return '';
  }
  const next = safeNext(params.get('rd') || '');

  let result = $state(null); // { ok, email } or { ok: false, error }

  // The page POSTs the token itself, so mail scanners that only open the link don't use it up.
  (async () => {
    if (!token) {
      result = { ok: false, incomplete: true };
      return;
    }
    const r = await api('/verify-email', { token });
    result = r.ok ? { ok: true, email: r.data.email } : { ok: false, error: r.data.error };
    if (r.ok) {
      refreshSession();
      if (next) setTimeout(() => (window.location.href = next), 1500);
    }
  })();
</script>

<AuthLayout title={result ? (result.ok ? t('verify.title_ok') : t('verify.title_failed')) : t('verify.title')}>
  {#if !result}
    <p class="text-sm text-zinc-400">{t('common.one_moment')}</p>
  {:else}
    <div class="space-y-4">
      <Alert kind={result.ok ? 'success' : 'error'}>{result.ok ? t('verify.verified', { email: result.email }) : result.incomplete ? t('verify.incomplete') : result.error}</Alert>
      {#if result.ok && next}
        <p class="text-sm text-zinc-400">{t('verify.taking_you')}</p>
        <a href={next} class="flex w-full items-center justify-center rounded-lg bg-indigo-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-400">
          {t('common.continue')}
        </a>
      {:else}
        {#if result.ok}
          <p class="text-sm text-zinc-400">{t('verify.now_you_can')}</p>
        {/if}
        <a href="/" onclick={link} class="flex w-full items-center justify-center rounded-lg bg-indigo-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-400">
          {app.session?.authenticated ? t('verify.go_account') : t('common.sign_in')}
        </a>
      {/if}
    </div>
  {/if}
</AuthLayout>
