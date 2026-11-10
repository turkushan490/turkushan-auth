<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Field from '../components/Field.svelte';
  import SubmitButton from '../components/SubmitButton.svelte';
  import Alert from '../components/Alert.svelte';
  import PasswordRules from '../components/PasswordRules.svelte';
  import { api } from '../lib/api.js';
  import { passwordRules } from '../lib/password.js';
  import { app, link, currentRD, withRD, siteInfo } from '../lib/state.svelte.js';
  import { i18n, t } from '../lib/i18n.svelte.js';
  import ProviderButtons from '../components/ProviderButtons.svelte';

  const rd = currentRD();
  let site = $state({ known: false });
  siteInfo(rd).then((s) => (site = s));

  let username = $state('');
  let email = $state('');
  let password = $state('');
  let confirm = $state('');
  let error = $state('');
  let busy = $state(false);

  const passwordOk = $derived(passwordRules(password).every((r) => r.ok));
  const mismatch = $derived(confirm.length > 0 && confirm !== password);

  async function submit(e) {
    e.preventDefault();
    error = '';
    if (!passwordOk) {
      error = t('common.pw_rules');
      return;
    }
    if (confirm !== password) {
      error = t('common.pw_mismatch');
      return;
    }
    busy = true;
    const r = await api('/register', { username, email, password, rd, lang: i18n.lang });
    if (r.ok) {
      window.location.href = r.data.redirect;
      return;
    }
    busy = false;
    error = r.data.error;
  }
</script>

<AuthLayout
  title={t('register.title')}
  subtitle={site.known ? t('register.subtitle_site', { name: site.name }) : t('register.subtitle')}
>
  <form class="space-y-4" onsubmit={submit}>
    {#if error}
      <Alert>{error}</Alert>
    {/if}
    {#if site.require_email}
      <Alert kind="info">
        <span class="font-semibold">{t('register.needs_email', { name: site.name })}</span>
        {t('register.needs_email_more')}
      </Alert>
    {/if}

    <Field
      label={t('common.username')}
      name="username"
      autocomplete="username"
      required
      bind:value={username}
      hint={t('common.username_hint')}
    />
    <Field
      label={t('register.email')}
      name="email"
      type="email"
      autocomplete="email"
      optional={!site.require_email}
      bind:value={email}
      hint={site.require_email
        ? t('register.email_hint_site', { name: site.name })
        : t('register.email_hint')}
    />
    <Field label={t('common.password')} name="password" type="password" autocomplete="new-password" required bind:value={password}>
      <PasswordRules {password} />
    </Field>
    <Field label={t('common.confirm_password')} name="confirm" type="password" autocomplete="new-password" required bind:value={confirm}>
      {#if mismatch}
        <p class="mt-1.5 text-xs text-rose-400">{t('common.pw_mismatch')}</p>
      {/if}
    </Field>

    <div class="pt-2">
      <SubmitButton {busy}>{t('register.title')}</SubmitButton>
    </div>
  </form>

  {#if app.session?.providers?.length}
    <div class="my-5 flex items-center gap-3 text-xs text-zinc-500">
      <span class="h-px flex-1 bg-zinc-800"></span>{t('common.or')}<span class="h-px flex-1 bg-zinc-800"></span>
    </div>
    <ProviderButtons {rd} />
  {/if}

  {#snippet footer()}
    {t('register.have_account')}
    <a href={withRD('/login', rd)} onclick={link} class="font-medium text-indigo-400 hover:text-indigo-300">{t('common.sign_in')}</a>
  {/snippet}
</AuthLayout>
