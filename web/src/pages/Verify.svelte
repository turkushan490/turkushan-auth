<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Alert from '../components/Alert.svelte';
  import { api } from '../lib/api.js';
  import { app, link, refreshSession } from '../lib/state.svelte.js';

  const token = new URLSearchParams(location.search).get('token') || '';

  let result = $state(null); // { ok, text }

  // The page POSTs the token itself, so mail scanners that only open the link don't use it up.
  (async () => {
    if (!token) {
      result = { ok: false, text: 'This link is incomplete. Open the link from the mail again.' };
      return;
    }
    const r = await api('/verify-email', { token });
    result = r.ok ? { ok: true, text: `${r.data.email} is verified.` } : { ok: false, text: r.data.error };
    if (r.ok) refreshSession();
  })();
</script>

<AuthLayout title={result ? (result.ok ? 'Email verified' : 'Could not verify') : 'Verifying…'}>
  {#if !result}
    <p class="text-sm text-zinc-400">One moment…</p>
  {:else}
    <div class="space-y-4">
      <Alert kind={result.ok ? 'success' : 'error'}>{result.text}</Alert>
      {#if result.ok}
        <p class="text-sm text-zinc-400">You can now open sites that need a verified email, and reset your password yourself if you ever forget it.</p>
      {/if}
      <a href="/" onclick={link} class="flex w-full items-center justify-center rounded-lg bg-indigo-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-400">
        {app.session?.authenticated ? 'Go to your account' : 'Sign in'}
      </a>
    </div>
  {/if}
</AuthLayout>
