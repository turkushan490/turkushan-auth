<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Field from '../components/Field.svelte';
  import SubmitButton from '../components/SubmitButton.svelte';
  import Alert from '../components/Alert.svelte';
  import PasswordRules from '../components/PasswordRules.svelte';
  import { api } from '../lib/api.js';
  import { passwordRules } from '../lib/password.js';
  import { app, link, navigate, refreshSession } from '../lib/state.svelte.js';

  const token = new URLSearchParams(location.search).get('token') || '';

  let password = $state('');
  let confirm = $state('');
  let error = $state('');
  let busy = $state(false);

  async function submit(e) {
    e.preventDefault();
    error = '';
    if (!passwordRules(password).every((r) => r.ok)) {
      error = 'Your password does not meet all the rules yet.';
      return;
    }
    if (confirm !== password) {
      error = 'The passwords do not match.';
      return;
    }
    busy = true;
    const r = await api('/password/reset', { token, password });
    busy = false;
    if (!r.ok) {
      error = r.data.error;
      return;
    }
    await refreshSession();
    app.flash = 'Password changed. Sign in with your new password.';
    navigate('/login');
  }
</script>

<AuthLayout title="Choose a new password" subtitle="You'll be signed out on all your devices.">
  {#if !token}
    <div class="space-y-4">
      <Alert>This link is incomplete. Open the link from the mail again, or request a new one.</Alert>
      <a href="/forgot" onclick={link} class="block text-center text-sm font-medium text-indigo-400 hover:text-indigo-300">Request a new link</a>
    </div>
  {:else}
    <form class="space-y-4" onsubmit={submit}>
      {#if error}
        <Alert>
          {error}
          {#if error.includes('Request a new')}<a href="/forgot" onclick={link} class="ml-1 underline underline-offset-2">Request a new link</a>{/if}
        </Alert>
      {/if}
      <Field label="New password" type="password" autocomplete="new-password" required bind:value={password}>
        <PasswordRules {password} />
      </Field>
      <Field label="Confirm new password" type="password" autocomplete="new-password" required bind:value={confirm} />
      <div class="pt-2">
        <SubmitButton {busy}>Save new password</SubmitButton>
      </div>
    </form>
  {/if}
</AuthLayout>
