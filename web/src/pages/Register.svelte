<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Field from '../components/Field.svelte';
  import SubmitButton from '../components/SubmitButton.svelte';
  import Alert from '../components/Alert.svelte';
  import PasswordRules from '../components/PasswordRules.svelte';
  import { api } from '../lib/api.js';
  import { passwordRules } from '../lib/password.js';
  import { app, link, currentRD, withRD, siteInfo } from '../lib/state.svelte.js';
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
      error = 'Your password does not meet all the rules yet.';
      return;
    }
    if (confirm !== password) {
      error = 'The passwords do not match.';
      return;
    }
    busy = true;
    const r = await api('/register', { username, email, password, rd });
    if (r.ok) {
      window.location.href = r.data.redirect;
      return;
    }
    busy = false;
    error = r.data.error;
  }
</script>

<AuthLayout
  title="Create account"
  subtitle={site.known ? `To open ${site.name}.` : 'After signing up, an admin decides which sites you can open.'}
>
  <form class="space-y-4" onsubmit={submit}>
    {#if error}
      <Alert>{error}</Alert>
    {/if}
    {#if site.require_email}
      <Alert kind="info">
        <span class="font-semibold">{site.name} needs a verified email address.</span>
        Fill in your email below. You'll get a mail with a link to confirm it.
      </Alert>
    {/if}

    <Field
      label="Username"
      name="username"
      autocomplete="username"
      required
      bind:value={username}
      hint="3–32 characters: letters, digits, . _ -"
    />
    <Field
      label="Email"
      name="email"
      type="email"
      autocomplete="email"
      optional={!site.require_email}
      bind:value={email}
      hint={site.require_email
        ? `Needed to open ${site.name}. You can also add it later, but you can't open the site without it.`
        : "Needed for some sites and to reset your password yourself. You'll get a mail to confirm it."}
    />
    <Field label="Password" name="password" type="password" autocomplete="new-password" required bind:value={password}>
      <PasswordRules {password} />
    </Field>
    <Field label="Confirm password" name="confirm" type="password" autocomplete="new-password" required bind:value={confirm}>
      {#if mismatch}
        <p class="mt-1.5 text-xs text-rose-400">The passwords do not match.</p>
      {/if}
    </Field>

    <div class="pt-2">
      <SubmitButton {busy}>Create account</SubmitButton>
    </div>
  </form>

  {#if app.session?.providers?.length}
    <div class="my-5 flex items-center gap-3 text-xs text-zinc-500">
      <span class="h-px flex-1 bg-zinc-800"></span>or<span class="h-px flex-1 bg-zinc-800"></span>
    </div>
    <ProviderButtons {rd} />
  {/if}

  {#snippet footer()}
    Already have an account?
    <a href={withRD('/login', rd)} onclick={link} class="font-medium text-indigo-400 hover:text-indigo-300">Sign in</a>
  {/snippet}
</AuthLayout>
