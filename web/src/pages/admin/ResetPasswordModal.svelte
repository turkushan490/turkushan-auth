<script>
  import { api } from '../../lib/api.js';
  import { toast } from '../../lib/state.svelte.js';
  import Modal from '../../components/Modal.svelte';
  import Field from '../../components/Field.svelte';
  import PasswordRules from '../../components/PasswordRules.svelte';
  import { btnGhost, btnPrimary } from './ui.js';

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
    toast(`New password set for ${user.username}. Their sessions were signed out.`);
    reload();
  }
</script>

<Modal bind:open title="Reset password for {user?.username}">
  <form class="space-y-4" onsubmit={submit}>
    <p class="text-sm text-zinc-400">
      Set a new password and send it to them yourself. They're signed out everywhere and can log in with the new password.
    </p>
    <Field label="New password" type="password" autocomplete="new-password" required bind:value={password}>
      <PasswordRules {password} />
    </Field>
    {#if error}<p class="text-sm text-rose-400" role="alert">{error}</p>{/if}
    <div class="flex justify-end gap-2">
      <button type="button" class={btnGhost} onclick={() => (open = false)}>Cancel</button>
      <button type="submit" class={btnPrimary} disabled={busy}>Set password</button>
    </div>
  </form>
</Modal>
