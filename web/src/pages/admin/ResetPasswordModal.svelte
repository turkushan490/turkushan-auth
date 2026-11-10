<script>
  import { api } from '../../lib/api.js';
  import { toast } from '../../lib/state.svelte.js';
  import Modal from '../../components/Modal.svelte';
  import Field from '../../components/Field.svelte';
  import PasswordRules from '../../components/PasswordRules.svelte';
  import { btnGhost, btnPrimary } from './ui.js';
  import { t } from '../../lib/i18n.svelte.js';

  let { open = $bindable(false), user, reload } = $props();

  let password = $state('');
  let error = $state('');
  let busy = $state(false);

  $effect(() => {
    if (open) {
      password = '';
      error = '';
    }
  });

  async function submit(e) {
    e.preventDefault();
    busy = true;
    const r = await api(`/admin/users/${user.id}/password`, { password }, 'PUT');
    busy = false;
    if (!r.ok) {
      error = r.data.error;
      return;
    }
    open = false;
    toast(t('users.reset_done', { name: user.username }));
    reload();
  }
</script>

<Modal bind:open title={t('users.reset_title', { name: user?.username })}>
  <form class="space-y-4" onsubmit={submit}>
    <p class="text-sm text-zinc-400">
      {t('users.reset_text')}
    </p>
    <Field label={t('users.new_password')} type="password" autocomplete="new-password" required bind:value={password}>
      <PasswordRules {password} />
    </Field>
    {#if error}<p class="text-sm text-rose-400" role="alert">{error}</p>{/if}
    <div class="flex justify-end gap-2">
      <button type="button" class={btnGhost} onclick={() => (open = false)}>{t('users.cancel')}</button>
      <button type="submit" class={btnPrimary} disabled={busy}>{t('users.set_password')}</button>
    </div>
  </form>
</Modal>
