<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Field from '../components/Field.svelte';
  import SubmitButton from '../components/SubmitButton.svelte';
  import Alert from '../components/Alert.svelte';
  import { api } from '../lib/api.js';
  import { app, link, currentRD, withRD, siteInfo, oauthError } from '../lib/state.svelte.js';
  import { t } from '../lib/i18n.svelte.js';
  import ProviderButtons from '../components/ProviderButtons.svelte';

  const rd = currentRD();
  let site = $state({ known: false });
  siteInfo(rd).then((s) => (site = s));
  const flash = app.flash;
  app.flash = '';

  let username = $state('');
  let password = $state('');
  let error = $state('');
  // Shown (in the current language) until the next attempt.
  let oauthCode = $state(new URLSearchParams(location.search).get('oauth_error') || '');
  let busy = $state(false);

  async function submit(e) {
    e.preventDefault();
    error = '';
    oauthCode = '';
    busy = true;
    const r = await api('/login', { username, password, rd });
    if (r.ok) {
      window.location.href = r.data.redirect; // may be another subdomain
      return;
    }
    busy = false;
    error = r.data.error;
    password = '';
  }
</script>

<AuthLayout title={t('common.sign_in')} subtitle={site.known ? t('login.subtitle_site', { name: site.name }) : rd ? t('login.subtitle_rd') : t('login.subtitle')}>
  <form class="space-y-4" onsubmit={submit}>
    {#if flash}
      <Alert kind="success">{typeof flash === 'function' ? flash() : flash}</Alert>
    {/if}
    {#if error || oauthCode}
      <Alert>{error || oauthError(oauthCode)}</Alert>
    {/if}

    <Field label={t('common.username')} name="username" autocomplete="username" required bind:value={username} />
    <Field label={t('common.password')} name="password" type="password" autocomplete="current-password" required bind:value={password} />
    <div class="-mt-1 text-end">
      <a href="/forgot" onclick={link} class="text-xs font-medium text-zinc-400 hover:text-zinc-200">{t('login.forgot')}</a>
    </div>

    <div class="pt-2">
      <SubmitButton {busy}>{t('common.sign_in')}</SubmitButton>
    </div>
  </form>

  {#if app.session?.providers?.length}
    <div class="my-5 flex items-center gap-3 text-xs text-zinc-500">
      <span class="h-px flex-1 bg-zinc-800"></span>{t('common.or')}<span class="h-px flex-1 bg-zinc-800"></span>
    </div>
    <ProviderButtons {rd} />
  {/if}

  {#snippet footer()}
    {t('login.no_account')}
    <a href={withRD('/register', rd)} onclick={link} class="font-medium text-indigo-400 hover:text-indigo-300">{t('login.create_one')}</a>
  {/snippet}
</AuthLayout>
