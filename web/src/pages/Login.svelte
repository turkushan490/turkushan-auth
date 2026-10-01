<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Field from '../components/Field.svelte';
  import SubmitButton from '../components/SubmitButton.svelte';
  import Alert from '../components/Alert.svelte';
  import { api } from '../lib/api.js';
  import { app, link, currentRD, withRD, siteInfo, oauthErrors } from '../lib/state.svelte.js';
  import ProviderButtons from '../components/ProviderButtons.svelte';

  const rd = currentRD();
  let site = $state({ known: false });
  siteInfo(rd).then((s) => (site = s));
  const flash = app.flash;
  app.flash = '';

  let username = $state('');
  let password = $state('');
  let error = $state(oauthErrors[new URLSearchParams(location.search).get('oauth_error')] || '');
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

<AuthLayout title="Sign in" subtitle={site.known ? `To continue to ${site.name}.` : rd ? 'Sign in to continue to the site you opened.' : 'Welcome back.'}>
  <form class="space-y-4" onsubmit={submit}>
    {#if flash}
      <Alert kind="success">{flash}</Alert>
    {/if}
    {#if error}
      <Alert>{error}</Alert>
    {/if}

    <Field label="Username" name="username" autocomplete="username" required bind:value={username} />
    <Field label="Password" name="password" type="password" autocomplete="current-password" required bind:value={password} />
    <div class="-mt-1 text-right">
      <a href="/forgot" onclick={link} class="text-xs font-medium text-zinc-400 hover:text-zinc-200">Forgot password?</a>
    </div>

    <div class="pt-2">
      <SubmitButton {busy}>Sign in</SubmitButton>
    </div>
  </form>

  {#if app.session?.providers?.length}
    <div class="my-5 flex items-center gap-3 text-xs text-zinc-500">
      <span class="h-px flex-1 bg-zinc-800"></span>or<span class="h-px flex-1 bg-zinc-800"></span>
    </div>
    <ProviderButtons {rd} />
  {/if}

  {#snippet footer()}
    No account yet?
    <a href={withRD('/register', rd)} onclick={link} class="font-medium text-indigo-400 hover:text-indigo-300">Create one</a>
  {/snippet}
</AuthLayout>
