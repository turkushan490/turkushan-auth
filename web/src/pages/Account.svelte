<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Field from '../components/Field.svelte';
  import Alert from '../components/Alert.svelte';
  import PasswordRules from '../components/PasswordRules.svelte';
  import { api } from '../lib/api.js';
  import { passwordRules } from '../lib/password.js';
  import { app, link, refreshSession, toast, oauthErrors } from '../lib/state.svelte.js';
  import ProviderButtons from '../components/ProviderButtons.svelte';

  const user = $derived(app.session.user);

  let open = $state(''); // 'email' | 'password' | 'logins' | ''

  // Coming back from connecting Discord/Google.
  const query = new URLSearchParams(location.search);
  const providerNames = { discord: 'Discord', google: 'Google' };
  if (query.get('connected')) {
    toast(`${providerNames[query.get('connected')] || 'Account'} is connected.`);
    open = 'logins';
    history.replaceState({}, '', '/');
  } else if (query.get('oauth_error')) {
    toast(oauthErrors[query.get('oauth_error')] || 'That did not work.', 'error');
    open = 'logins';
    history.replaceState({}, '', '/');
  }

  const logins = $derived(user.logins || []);
  const canConnect = $derived((app.session.providers || []).filter((p) => !logins.includes(p)));

  async function disconnect(p) {
    if (!confirm(`Disconnect ${providerNames[p]}? You can't sign in with it anymore.`)) return;
    busy = true;
    const r = await api(`/account/logins/${p}`, undefined, 'DELETE');
    busy = false;
    toast(r.ok ? r.data.message : r.data.error, r.ok ? 'success' : 'error');
    if (r.ok) refreshSession();
  }
  let busy = $state(false);
  let error = $state('');

  function toggle(section) {
    open = open === section ? '' : section;
    error = '';
  }

  // Email
  let newEmail = $state('');
  let emailPassword = $state('');

  async function saveEmail(e) {
    e.preventDefault();
    error = '';
    busy = true;
    const r = await api('/account/email', { email: newEmail, password: emailPassword });
    busy = false;
    if (!r.ok) {
      error = r.data.error;
      return;
    }
    toast(r.data.message);
    emailPassword = '';
    newEmail = '';
    open = '';
    refreshSession();
  }

  async function resend() {
    busy = true;
    const r = await api('/account/email/resend', {});
    busy = false;
    toast(r.ok ? r.data.message : r.data.error, r.ok ? 'success' : 'error');
  }

  // Password
  let current = $state('');
  let next = $state('');
  let confirm = $state('');

  async function savePassword(e) {
    e.preventDefault();
    error = '';
    if (!passwordRules(next).every((r) => r.ok)) {
      error = 'Your new password does not meet all the rules yet.';
      return;
    }
    if (next !== confirm) {
      error = 'The new passwords do not match.';
      return;
    }
    busy = true;
    const r = await api('/account/password', { current, new: next });
    busy = false;
    if (!r.ok) {
      error = r.data.error;
      return;
    }
    toast(r.data.message);
    current = next = confirm = '';
    open = '';
    refreshSession();
  }

  const sectionBtn =
    'flex w-full items-center justify-between gap-3 rounded-lg px-1 py-2 text-left text-sm font-medium text-zinc-200 transition hover:text-white';
</script>

<AuthLayout title="You're signed in">
  <div class="space-y-5">
    <div class="flex items-center gap-3 rounded-xl border border-zinc-800 bg-zinc-950/50 p-4">
      <div class="grid size-11 shrink-0 place-items-center rounded-full bg-indigo-500/20 text-base font-semibold uppercase text-indigo-300">
        {user.username.slice(0, 1)}
      </div>
      <div class="min-w-0">
        <div class="flex items-center gap-2">
          <p class="truncate font-medium">{user.username}</p>
          {#if user.is_admin}
            <span class="rounded-full bg-indigo-500/15 px-2 py-0.5 text-xs font-medium text-indigo-300 ring-1 ring-indigo-400/20">Admin</span>
          {/if}
        </div>
        <p class="truncate text-sm text-zinc-400">
          {#if user.email}
            {user.email}
            {#if user.email_verified}<span class="text-emerald-400">· verified</span>{:else}<span class="text-amber-400/90">· not verified</span>{/if}
          {:else}
            No email address
          {/if}
        </p>
      </div>
    </div>

    {#if user.email && !user.email_verified}
      <div class="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3.5 py-3 text-sm text-amber-200">
        Check your inbox and click the link to verify {user.email}.
        <button type="button" class="ml-1 font-medium underline underline-offset-2 hover:text-amber-100 disabled:opacity-60" onclick={resend} disabled={busy}>Send again</button>
      </div>
    {/if}

    <p class="text-sm text-zinc-400">You can now open the sites you have access to. This login works on all of them.</p>

    <div class="divide-y divide-zinc-800 border-y border-zinc-800">
      <div class="py-1">
        <button type="button" class={sectionBtn} onclick={() => toggle('email')} aria-expanded={open === 'email'}>
          {user.email ? 'Change email address' : 'Add email address'}
          <svg viewBox="0 0 24 24" class="size-4 text-zinc-500 transition {open === 'email' ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
        </button>
        {#if open === 'email'}
          <form class="space-y-3 pb-4 pt-2" onsubmit={saveEmail}>
            {#if error}<Alert>{error}</Alert>{/if}
            <Field label="Email address" type="email" autocomplete="email" bind:value={newEmail}
              hint={user.email ? 'Leave empty to remove your address.' : "We'll send a link to confirm it."} />
            {#if user.email_verified}
              <Field label="Your password" type="password" autocomplete="current-password" required bind:value={emailPassword}
                hint="Needed because your email address can reset your password." />
            {/if}
            <button type="submit" disabled={busy}
              class="w-full rounded-lg bg-indigo-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-400 disabled:opacity-60">
              {busy ? 'Saving…' : user.email_verified ? 'Save' : 'Send verification link'}
            </button>
          </form>
        {/if}
      </div>
      <div class="py-1">
        <button type="button" class={sectionBtn} onclick={() => toggle('password')} aria-expanded={open === 'password'}>
          {user.has_password ? 'Change password' : 'Set a password'}
          <svg viewBox="0 0 24 24" class="size-4 text-zinc-500 transition {open === 'password' ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
        </button>
        {#if open === 'password'}
          <form class="space-y-3 pb-4 pt-2" onsubmit={savePassword}>
            {#if error}<Alert>{error}</Alert>{/if}
            {#if user.has_password}
              <Field label="Current password" type="password" autocomplete="current-password" required bind:value={current} />
            {/if}
            <Field label="New password" type="password" autocomplete="new-password" required bind:value={next}>
              <PasswordRules password={next} />
            </Field>
            <Field label="Confirm new password" type="password" autocomplete="new-password" required bind:value={confirm} />
            <button type="submit" disabled={busy}
              class="w-full rounded-lg bg-indigo-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-400 disabled:opacity-60">
              {busy ? 'Saving…' : 'Change password'}
            </button>
            <p class="text-xs text-zinc-500">Your other devices will be signed out.</p>
          </form>
        {/if}
      </div>
    </div>

    {#if logins.length || canConnect.length}
      <div class="-mt-5 border-b border-zinc-800 py-1">
        <button type="button" class={sectionBtn} onclick={() => toggle('logins')} aria-expanded={open === 'logins'}>
          Sign-in methods
          <svg viewBox="0 0 24 24" class="size-4 text-zinc-500 transition {open === 'logins' ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
        </button>
        {#if open === 'logins'}
          <div class="space-y-3 pb-4 pt-2">
            <ul class="divide-y divide-zinc-800 rounded-lg border border-zinc-800 text-sm">
              <li class="flex items-center justify-between gap-3 px-3 py-2.5">
                <span>Username and password</span>
                <span class="text-xs {user.has_password ? 'text-emerald-400' : 'text-zinc-500'}">{user.has_password ? 'on' : 'no password set'}</span>
              </li>
              {#each logins as p}
                <li class="flex items-center justify-between gap-3 px-3 py-2.5">
                  <span>{providerNames[p]} <span class="text-xs text-emerald-400">· connected</span></span>
                  <button type="button" class="text-xs font-medium text-zinc-400 hover:text-rose-300 disabled:opacity-60" disabled={busy} onclick={() => disconnect(p)}>Disconnect</button>
                </li>
              {/each}
            </ul>
            {#if canConnect.length}
              <ProviderButtons link only={canConnect} verb="Connect" />
            {/if}
          </div>
        {/if}
      </div>
    {/if}

    <div class="space-y-2">
      {#if user.staff}
        <a
          href="/admin"
          onclick={link}
          class="flex w-full items-center justify-center gap-2 rounded-lg bg-indigo-500 px-4 py-2.5 text-sm font-semibold text-white shadow-lg shadow-indigo-500/20 transition hover:bg-indigo-400"
        >
          <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M12 3l8 3v6c0 4.5-3.2 8.3-8 9-4.8-.7-8-4.5-8-9V6z" />
          </svg>
          Admin panel
        </a>
      {/if}
      <a
        href="/logout"
        onclick={link}
        class="flex w-full items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm font-medium text-zinc-100 transition hover:bg-zinc-700"
      >
        <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9" />
        </svg>
        Sign out
      </a>
    </div>
  </div>
</AuthLayout>
