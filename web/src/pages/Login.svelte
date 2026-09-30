<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Field from '../components/Field.svelte';
  import SubmitButton from '../components/SubmitButton.svelte';
  import Alert from '../components/Alert.svelte';
  import { api } from '../lib/api.js';
  import { app, link, currentRD, withRD } from '../lib/state.svelte.js';

  const rd = currentRD();
  const flash = app.flash;
  app.flash = '';

  let username = $state('');
  let password = $state('');
  let error = $state('');
  let busy = $state(false);

  async function submit(e) {
    e.preventDefault();
    error = '';
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

<AuthLayout title="Sign in" subtitle={rd ? 'Sign in to continue to the site you opened.' : 'Welcome back.'}>
  <form class="space-y-4" onsubmit={submit}>
    {#if flash}
      <Alert kind="success">{flash}</Alert>
    {/if}
    {#if error}
      <Alert>{error}</Alert>
    {/if}

    <Field label="Username" name="username" autocomplete="username" required bind:value={username} />
    <Field label="Password" name="password" type="password" autocomplete="current-password" required bind:value={password} />

    <div class="pt-2">
      <SubmitButton {busy}>Sign in</SubmitButton>
    </div>
  </form>

  {#snippet footer()}
    No account yet?
    <a href={withRD('/register', rd)} onclick={link} class="font-medium text-indigo-400 hover:text-indigo-300">Create one</a>
  {/snippet}
</AuthLayout>
