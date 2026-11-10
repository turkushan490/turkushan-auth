<script>
  import AuthLayout from '../components/AuthLayout.svelte';
  import Field from '../components/Field.svelte';
  import Alert from '../components/Alert.svelte';
  import PasswordRules from '../components/PasswordRules.svelte';
  import { api } from '../lib/api.js';
  import { passwordRules } from '../lib/password.js';
  import { app, link, refreshSession, toast, oauthError } from '../lib/state.svelte.js';
  import { t } from '../lib/i18n.svelte.js';
  import ProviderButtons from '../components/ProviderButtons.svelte';
  import { confirmDialog } from '../lib/confirm.svelte.js';
  import LanguagePicker from '../components/LanguagePicker.svelte';

  const user = $derived(app.session.user);

  let open = $state(''); // 'email' | 'password' | 'logins' | ''

  // Coming back from connecting Discord/Google.
  const query = new URLSearchParams(location.search);
  const providerNames = { discord: 'Discord', google: 'Google' };
  if (query.get('connected')) {
    const name = providerNames[query.get('connected')];
    toast(name ? t('account.connected', { name }) : t('account.connected_generic'));
    open = 'logins';
    history.replaceState({}, '', '/');
  } else if (query.get('oauth_error')) {
    toast(oauthError(query.get('oauth_error')), 'error');
    open = 'logins';
    history.replaceState({}, '', '/');
  }

  const logins = $derived(user.logins || []);
  const canConnect = $derived((app.session.providers || []).filter((p) => !logins.includes(p)));

  async function disconnect(p) {
    if (!(await confirmDialog({ title: t('account.disconnect_title', { name: providerNames[p] }), text: t('account.disconnect_text'), ok: t('account.disconnect'), danger: true }))) return;
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
  let confirmPw = $state('');

  async function savePassword(e) {
    e.preventDefault();
    error = '';
    if (!passwordRules(next).every((r) => r.ok)) {
      error = t('account.pw_rules');
      return;
    }
    if (next !== confirmPw) {
      error = t('account.pw_mismatch');
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
    current = next = confirmPw = '';
    open = '';
    refreshSession();
  }

  const sectionBtn =
    'flex w-full items-center justify-between gap-3 rounded-lg px-1 py-2 text-start text-sm font-medium text-zinc-200 transition hover:text-white';
</script>

<AuthLayout title={t('account.title')}>
  <div class="space-y-5">
    <div class="flex items-center gap-3 rounded-xl border border-zinc-800 bg-zinc-950/50 p-4">
      <div class="grid size-11 shrink-0 place-items-center rounded-full bg-indigo-500/20 text-base font-semibold uppercase text-indigo-300">
        {user.username.slice(0, 1)}
      </div>
      <div class="min-w-0">
        <div class="flex items-center gap-2">
          <p class="truncate font-medium">{user.username}</p>
          {#if user.is_admin}
            <span class="rounded-full bg-indigo-500/15 px-2 py-0.5 text-xs font-medium text-indigo-300 ring-1 ring-indigo-400/20">{t('account.admin')}</span>
          {/if}
        </div>
        <p class="truncate text-sm text-zinc-400">
          {#if user.email}
            {user.email}
            {#if user.email_verified}<span class="text-emerald-400">· {t('account.verified')}</span>{:else}<span class="text-amber-400/90">· {t('account.not_verified')}</span>{/if}
          {:else}
            {t('account.no_email')}
          {/if}
        </p>
      </div>
    </div>

    {#if user.email && !user.email_verified}
      <div class="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3.5 py-3 text-sm text-amber-200">
        {t('account.verify_prompt', { email: user.email })}
        <button type="button" class="ms-1 font-medium underline underline-offset-2 hover:text-amber-100 disabled:opacity-60" onclick={resend} disabled={busy}>{t('account.send_again')}</button>
      </div>
    {/if}

    <p class="text-sm text-zinc-400">{t('account.intro')}</p>

    <div class="divide-y divide-zinc-800 border-y border-zinc-800">
      <div class="py-1">
        <button type="button" class={sectionBtn} onclick={() => toggle('email')} aria-expanded={open === 'email'}>
          {user.email ? t('account.change_email') : t('account.add_email')}
          <svg viewBox="0 0 24 24" class="size-4 text-zinc-500 transition {open === 'email' ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
        </button>
        {#if open === 'email'}
          <form class="space-y-3 pb-4 pt-2" onsubmit={saveEmail}>
            {#if error}<Alert>{error}</Alert>{/if}
            <Field label={t('account.email')} type="email" autocomplete="email" bind:value={newEmail}
              hint={user.email ? t('account.email_hint_remove') : t('account.email_hint_new')} />
            {#if user.email_verified}
              <Field label={t('account.your_password')} type="password" autocomplete="current-password" required bind:value={emailPassword}
                hint={t('account.your_password_hint')} />
            {/if}
            <button type="submit" disabled={busy}
              class="w-full rounded-lg bg-indigo-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-400 disabled:opacity-60">
              {busy ? t('common.saving') : user.email_verified ? t('common.save') : t('common.send_verification')}
            </button>
          </form>
        {/if}
      </div>
      <div class="py-1">
        <button type="button" class={sectionBtn} onclick={() => toggle('password')} aria-expanded={open === 'password'}>
          {user.has_password ? t('account.change_password') : t('account.set_password')}
          <svg viewBox="0 0 24 24" class="size-4 text-zinc-500 transition {open === 'password' ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
        </button>
        {#if open === 'password'}
          <form class="space-y-3 pb-4 pt-2" onsubmit={savePassword}>
            {#if error}<Alert>{error}</Alert>{/if}
            {#if user.has_password}
              <Field label={t('account.current_password')} type="password" autocomplete="current-password" required bind:value={current} />
            {/if}
            <Field label={t('common.new_password')} type="password" autocomplete="new-password" required bind:value={next}>
              <PasswordRules password={next} />
            </Field>
            <Field label={t('common.confirm_new_password')} type="password" autocomplete="new-password" required bind:value={confirmPw} />
            <button type="submit" disabled={busy}
              class="w-full rounded-lg bg-indigo-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-400 disabled:opacity-60">
              {busy ? t('common.saving') : t('account.change_password')}
            </button>
            <p class="text-xs text-zinc-500">{t('account.other_devices')}</p>
          </form>
        {/if}
      </div>
    </div>

    {#if logins.length || canConnect.length}
      <div class="-mt-5 border-b border-zinc-800 py-1">
        <button type="button" class={sectionBtn} onclick={() => toggle('logins')} aria-expanded={open === 'logins'}>
          {t('account.logins')}
          <svg viewBox="0 0 24 24" class="size-4 text-zinc-500 transition {open === 'logins' ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
        </button>
        {#if open === 'logins'}
          <div class="space-y-3 pb-4 pt-2">
            <ul class="divide-y divide-zinc-800 rounded-lg border border-zinc-800 text-sm">
              <li class="flex items-center justify-between gap-3 px-3 py-2.5">
                <span>{t('account.username_password')}</span>
                <span class="text-xs {user.has_password ? 'text-emerald-400' : 'text-zinc-500'}">{user.has_password ? t('account.login_on') : t('account.no_password')}</span>
              </li>
              {#each logins as p}
                <li class="flex items-center justify-between gap-3 px-3 py-2.5">
                  <span>{providerNames[p]} <span class="text-xs text-emerald-400">· {t('account.login_connected')}</span></span>
                  <button type="button" class="text-xs font-medium text-zinc-400 hover:text-rose-300 disabled:opacity-60" disabled={busy} onclick={() => disconnect(p)}>{t('account.disconnect')}</button>
                </li>
              {/each}
            </ul>
            {#if canConnect.length}
              <ProviderButtons link only={canConnect} />
            {/if}
          </div>
        {/if}
      </div>
    {/if}

    <div class="space-y-1.5">
      <p class="text-sm font-medium text-zinc-300">{t('lang.label')}</p>
      <LanguagePicker compact={false} />
    </div>

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
          {t('account.admin_panel')}
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
        {t('account.sign_out')}
      </a>
    </div>
  </div>
</AuthLayout>
