<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Field from '../components/Field.svelte';
  import SubmitButton from '../components/SubmitButton.svelte';
  import Alert from '../components/Alert.svelte';
  import PasswordRules from '../components/PasswordRules.svelte';
  import { api } from '../lib/api.js';
  import { passwordRules } from '../lib/password.js';
  import { link } from '../lib/state.svelte.js';

  // First sign-in with Discord/Google: pick a username (and maybe a password), or skip.
  const p = new URLSearchParams(location.search).get('p') || '';

  let info = $state(null);
  let expired = $state('');
  let username = $state('');
  let withPassword = $state(false);
  let password = $state('');
  let error = $state('');
  let busy = $state('');

  api('/oauth/pending?p=' + encodeURIComponent(p)).then((r) => {
    if (!r.ok) {
      expired = r.data.error;
      return;
    }
    info = r.data;
    username = r.data.suggested_username;
  });

  async function finish(body, which) {
    error = '';
    busy = which;
    const r = await api('/oauth/complete', { p, ...body });
    if (r.ok) {
      window.location.href = r.data.redirect;
      return;
    }
    busy = '';
    error = r.data.error;
  }

  function submit(e) {
    e.preventDefault();
    if (withPassword && !passwordRules(password).every((r) => r.ok)) {
      error = 'Your password does not meet all the rules yet.';
      return;
    }
    finish({ username, password: withPassword ? password : '' }, 'save');
  }
</script>

<AuthLayout title={info ? 'Almost there' : expired ? 'Sign-up expired' : 'One moment…'} subtitle={info ? `You're signing in with ${info.label}${info.display ? ` as ${info.display}` : ''}.` : ''}>
  {#if expired}
    <div class="space-y-4">
      <Alert>{expired}</Alert>
      <a href="/login" onclick={link} class="flex w-full items-center justify-center rounded-lg bg-indigo-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-400">Back to sign in</a>
    </div>
  {:else if info}
    <form class="space-y-4" onsubmit={submit}>
      {#if error}<Alert>{error}</Alert>{/if}

      <Field label="Choose a username" name="username" autocomplete="username" required bind:value={username}
        hint="3–32 characters: letters, digits, . _ -" />

      <div class="rounded-lg border border-zinc-800 bg-zinc-950/40 p-3">
        <label class="flex cursor-pointer items-start gap-3">
          <input type="checkbox" bind:checked={withPassword} class="mt-0.5 size-4 accent-indigo-500" />
          <span>
            <span class="block text-sm font-medium text-zinc-200">Also set a password</span>
            <span class="block text-xs text-zinc-500">Optional. Then you can also sign in with your username, without {info.label}.</span>
          </span>
        </label>
        {#if withPassword}
          <div class="mt-3">
            <Field label="Password" type="password" autocomplete="new-password" required bind:value={password}>
              <PasswordRules {password} />
            </Field>
          </div>
        {/if}
      </div>

      <SubmitButton busy={busy === 'save'} disabled={busy !== ''}>Create my account</SubmitButton>
      <button type="button" disabled={busy !== ''} onclick={() => finish({}, 'skip')}
        class="w-full rounded-lg px-4 py-2 text-sm font-medium text-zinc-400 transition hover:text-zinc-200 disabled:opacity-60">
        {busy === 'skip' ? 'One moment…' : `Skip, use "${info.suggested_username}" and no password`}
      </button>
    </form>
  {/if}
</AuthLayout>
