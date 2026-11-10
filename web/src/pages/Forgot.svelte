<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Field from '../components/Field.svelte';
  import SubmitButton from '../components/SubmitButton.svelte';
  import Alert from '../components/Alert.svelte';
  import { api } from '../lib/api.js';
  import { link } from '../lib/state.svelte.js';
  import { t } from '../lib/i18n.svelte.js';

  let login = $state('');
  let error = $state('');
  let done = $state('');
  let busy = $state(false);

  async function submit(e) {
    e.preventDefault();
    error = '';
    busy = true;
    const r = await api('/password/forgot', { login });
    busy = false;
    if (r.ok) done = r.data.message;
    else error = r.data.error;
  }
</script>

<AuthLayout title={t('forgot.title')} subtitle={t('forgot.subtitle')}>
  {#if done}
    <div class="space-y-4">
      <Alert kind="success">{done}</Alert>
      <p class="text-sm text-zinc-400">{t('forgot.check_spam')}</p>
    </div>
  {:else}
    <form class="space-y-4" onsubmit={submit}>
      {#if error}<Alert>{error}</Alert>{/if}
      <Field label={t('forgot.login')} name="login" autocomplete="username" required bind:value={login} />
      <p class="text-xs text-zinc-500">
        {t('forgot.only_verified')}
      </p>
      <div class="pt-2">
        <SubmitButton {busy}>{t('forgot.submit')}</SubmitButton>
      </div>
    </form>
  {/if}

  {#snippet footer()}
    <a href="/login" onclick={link} class="font-medium text-indigo-400 hover:text-indigo-300">{t('common.back_to_sign_in')}</a>
  {/snippet}
</AuthLayout>
