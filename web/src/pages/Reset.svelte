<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Field from '../components/Field.svelte';
  import SubmitButton from '../components/SubmitButton.svelte';
  import Alert from '../components/Alert.svelte';
  import PasswordRules from '../components/PasswordRules.svelte';
  import { api } from '../lib/api.js';
  import { passwordRules } from '../lib/password.js';
  import { app, link, navigate, refreshSession } from '../lib/state.svelte.js';
  import { t } from '../lib/i18n.svelte.js';

  const params = new URLSearchParams(location.search);
  const token = params.get('token') || '';
  const invite = params.get('invite') === '1'; // link from an invite mail: first password

  let password = $state('');
  let confirmPw = $state('');
  const mismatch = $derived(confirmPw.length > 0 && confirmPw !== password);
  let error = $state('');
  let errorCode = $state(''); // the server's code, to offer a new link when this one is used up
  let busy = $state(false);

  async function submit(e) {
    e.preventDefault();
    error = '';
    errorCode = '';
    if (!passwordRules(password).every((r) => r.ok)) {
      error = t('common.pw_rules');
      return;
    }
    if (confirmPw !== password) {
      error = t('common.pw_mismatch');
      return;
    }
    busy = true;
    const r = await api('/password/reset', { token, password });
    busy = false;
    if (!r.ok) {
      error = r.data.error;
      errorCode = r.data.code || '';
      return;
    }
    await refreshSession();
    app.flash = invite ? () => t('reset.flash_invite') : () => t('reset.flash_changed');
    navigate('/login');
  }
</script>

<AuthLayout
  title={invite ? t('reset.title_invite') : t('reset.title')}
  subtitle={invite ? t('reset.subtitle_invite') : t('reset.subtitle')}
>
  {#if !token}
    <div class="space-y-4">
      <Alert>{t('reset.incomplete')}</Alert>
      <a href="/forgot" onclick={link} class="block text-center text-sm font-medium text-indigo-400 hover:text-indigo-300">{t('reset.request_link')}</a>
    </div>
  {:else}
    <form class="space-y-4" onsubmit={submit}>
      {#if error}
        <Alert>
          {error}
          {#if errorCode === 'link_invalid' || errorCode === 'link_invalid_account'}<a href="/forgot" onclick={link} class="ms-1 underline underline-offset-2">{t('reset.request_link')}</a>{/if}
        </Alert>
      {/if}
      <Field label={invite ? t('common.password') : t('common.new_password')} type="password" autocomplete="new-password" required bind:value={password}>
        <PasswordRules {password} />
      </Field>
      <Field label={invite ? t('common.confirm_password') : t('common.confirm_new_password')} type="password" autocomplete="new-password" required bind:value={confirmPw}>
        {#if mismatch}<p class="mt-1.5 text-xs text-rose-400">{t('reset.mismatch_yet')}</p>{/if}
      </Field>
      <div class="pt-2">
        <SubmitButton {busy}>{t('reset.submit')}</SubmitButton>
      </div>
    </form>
  {/if}
</AuthLayout>
