<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Field from '../components/Field.svelte';
  import SubmitButton from '../components/SubmitButton.svelte';
  import Alert from '../components/Alert.svelte';
  import { api } from '../lib/api.js';
  import { link } from '../lib/state.svelte.js';

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

<AuthLayout title="Forgot your password?" subtitle="We'll mail you a link to choose a new one.">
  {#if done}
    <div class="space-y-4">
      <Alert kind="success">{done}</Alert>
      <p class="text-sm text-zinc-400">Check your spam folder too. The link works for 1 hour.</p>
    </div>
  {:else}
    <form class="space-y-4" onsubmit={submit}>
      {#if error}<Alert>{error}</Alert>{/if}
      <Field label="Username or email" name="login" autocomplete="username" required bind:value={login} />
      <p class="text-xs text-zinc-500">
        This only works if your account has a verified email address. Otherwise, ask the admin to reset your password.
      </p>
      <div class="pt-2">
        <SubmitButton {busy}>Send reset link</SubmitButton>
      </div>
    </form>
  {/if}

  {#snippet footer()}
    <a href="/login" onclick={link} class="font-medium text-indigo-400 hover:text-indigo-300">Back to sign in</a>
  {/snippet}
</AuthLayout>
