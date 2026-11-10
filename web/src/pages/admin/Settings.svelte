<script>
  import { untrack } from 'svelte';
  import { api, apiDownload } from '../../lib/api.js';
  import { confirmDialog } from '../../lib/confirm.svelte.js';
  import { toast } from '../../lib/state.svelte.js';
  import { card, btnGhost, btnPrimary, btnDanger, input, label } from './ui.js';
  import { t } from '../../lib/i18n.svelte.js';

  let { data, reload } = $props();

  let portalURL = $state(untrack(() => data.settings.portal_url));
  let webhook = $state('');
  let busy = $state('');

  // Email (SMTP). The saved password is never sent to the browser; empty keeps it.
  const smtp0 = untrack(() => data.settings.smtp);
  let smtpHost = $state(smtp0.host);
  let smtpPort = $state(smtp0.port || 465);
  let smtpUser = $state(smtp0.username);
  let smtpPassword = $state('');
  let smtpFrom = $state(smtp0.from || `noreply@${untrack(() => data.settings.base_domain)}`);
  let testTo = $state('');

  function useResend() {
    smtpHost = 'smtp.resend.com';
    smtpPort = 465;
    smtpUser = 'resend';
  }

  async function saveMail(e) {
    e.preventDefault();
    const ok = await save(
      { smtp_host: smtpHost, smtp_port: Number(smtpPort), smtp_username: smtpUser, smtp_password: smtpPassword, smtp_from: smtpFrom },
      t('settings.mail.saved'),
      'mail',
    );
    if (ok) smtpPassword = '';
  }

  // Sign in with Discord / Google. Secrets are never sent to the browser; empty keeps the saved one.
  const oauth0 = untrack(() => data.settings.oauth);
  let oauthDraft = $state(Object.fromEntries(Object.entries(oauth0).map(([name, o]) => [name, { client_id: o.client_id, client_secret: '' }])));
  let howTo = $state('');
  const oauthOrder = ['discord', 'google'];

  async function saveLogin(e, name) {
    e.preventDefault();
    const label = data.settings.oauth[name].label;
    if (await save({ oauth: { [name]: oauthDraft[name] } }, t('settings.oauth.on_toast', { name: label }), 'oauth-' + name)) oauthDraft[name].client_secret = '';
  }

  async function removeLogin(name) {
    const label = data.settings.oauth[name].label;
    if (!(await confirmDialog({ title: t('settings.oauth.off_title', { name: label }), text: t('settings.oauth.off_text', { name: label }), ok: t('settings.oauth.turn_off'), danger: true }))) return;
    if (await save({ oauth: { [name]: { remove: true } } }, t('settings.oauth.off_toast', { name: label }), 'oauth-' + name)) oauthDraft[name] = { client_id: '', client_secret: '' };
  }

  async function copy(text) {
    try {
      await navigator.clipboard.writeText(text);
      toast(t('settings.copied'));
    } catch {
      toast(t('settings.copy_failed'), 'error');
    }
  }

  async function testMail(e) {
    e.preventDefault();
    busy = 'testmail';
    const r = await api('/admin/settings/test-mail', { to: testTo });
    busy = '';
    toast(r.ok ? t('settings.mail.test_sent', { to: testTo }) : r.data.error, r.ok ? 'success' : 'error');
  }

  async function save(body, okText, which) {
    busy = which;
    const r = await api('/admin/settings', body, 'PUT');
    busy = '';
    if (!r.ok) {
      toast(r.data.error, 'error');
      return false;
    }
    toast(okText);
    reload();
    return true;
  }

  function savePortal(e) {
    e.preventDefault();
    save({ portal_url: portalURL }, t('settings.portal.saved'), 'portal');
  }

  async function saveWebhook(e) {
    e.preventDefault();
    if (await save({ discord_webhook: webhook }, t('settings.webhook.saved'), 'webhook')) webhook = '';
  }

  async function removeWebhook() {
    if (await confirmDialog({ title: t('settings.webhook.remove_title'), text: t('settings.webhook.remove_text'), ok: t('settings.remove'), danger: true })) {
      save({ discord_webhook: '' }, t('settings.webhook.removed'), 'webhook');
    }
  }

  async function downloadBackup() {
    busy = 'backup';
    const r = await apiDownload('/admin/backup', 'turkushan-auth-backup.db');
    busy = '';
    if (!r.ok) toast(r.error, 'error');
  }

  async function testWebhook() {
    busy = 'test';
    const r = await api('/admin/settings/test-discord', {});
    busy = '';
    toast(r.ok ? t('settings.webhook.test_sent') : r.data.error, r.ok ? 'success' : 'error');
  }
</script>

<div class="space-y-6">
  <section class="{card} p-5 sm:p-6">
    <h2 class="font-semibold">{t('settings.portal.title')}</h2>
    <p class="mt-1 text-sm text-zinc-400">
      {t('settings.portal.intro')}
    </p>
    <form class="mt-4 flex flex-col gap-2 sm:flex-row" onsubmit={savePortal}>
      <label class="sr-only" for="portal-url">{t('settings.portal.label')}</label>
      <input id="portal-url" class={input} bind:value={portalURL} placeholder="http://192.168.0.6:3010" autocapitalize="off" spellcheck="false" />
      <button type="submit" class="{btnPrimary} shrink-0" disabled={busy !== ''}>{t('settings.save')}</button>
    </form>
  </section>

  {#if !data.me.admin}
    <section class="{card} p-5 text-sm text-zinc-400 sm:p-6">
      <h2 class="font-semibold text-zinc-100">{t('settings.admin_only_title')}</h2>
      <p class="mt-1">{t('settings.admin_only_text')}</p>
    </section>
  {:else}
  <section class="{card} p-5 sm:p-6">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h2 class="font-semibold">{t('settings.mail.title')}</h2>
        <p class="mt-1 text-sm text-zinc-400">{t('settings.mail.intro')}</p>
      </div>
      {#if data.settings.smtp.ready}
        <span class="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/15 px-2.5 py-1 text-xs font-medium text-emerald-300 ring-1 ring-emerald-500/25">
          <span class="size-1.5 rounded-full bg-emerald-400"></span> {t('settings.status.set_up')}
        </span>
      {:else}
        <span class="rounded-full bg-zinc-800 px-2.5 py-1 text-xs font-medium text-zinc-400">{t('settings.status.not_set')}</span>
      {/if}
    </div>

    <form class="mt-4 space-y-4" onsubmit={saveMail}>
      <div class="flex flex-wrap items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-950/40 px-3 py-2.5 text-sm text-zinc-400">
        {t('settings.mail.resend_q')}
        <button type="button" class={btnGhost} onclick={useResend}>{t('settings.mail.resend_fill')}</button>
        <span class="text-xs text-zinc-500">{t('settings.mail.resend_then')}</span>
      </div>
      <div class="grid gap-4 sm:grid-cols-[1fr_8rem]">
        <div>
          <label class={label} for="smtp-host">{t('settings.mail.server')}</label>
          <input id="smtp-host" class={input} bind:value={smtpHost} placeholder="smtp.resend.com" autocapitalize="off" spellcheck="false" />
        </div>
        <div>
          <label class={label} for="smtp-port">{t('settings.mail.port')}</label>
          <input id="smtp-port" class={input} type="number" min="1" max="65535" bind:value={smtpPort} />
        </div>
      </div>
      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label class={label} for="smtp-user">{t('settings.mail.username')}</label>
          <input id="smtp-user" class={input} bind:value={smtpUser} placeholder="resend" autocomplete="off" autocapitalize="off" spellcheck="false" />
        </div>
        <div>
          <label class={label} for="smtp-pass">{t('settings.mail.password')}</label>
          <input id="smtp-pass" class={input} type="password" bind:value={smtpPassword} autocomplete="new-password"
            placeholder={data.settings.smtp.password_set ? t('settings.secret_saved') : 're_…'} />
        </div>
      </div>
      <div>
        <label class={label} for="smtp-from">{t('settings.mail.from')}</label>
        <input id="smtp-from" class={input} bind:value={smtpFrom} placeholder="noreply@{data.settings.base_domain}" autocapitalize="off" spellcheck="false" />
        <p class="mt-1.5 text-xs text-zinc-500">
          {t('settings.mail.from_help')} <span class="font-mono">{data.settings.base_domain} &lt;noreply@{data.settings.base_domain}&gt;</span>{t('settings.sentence_end')}
        </p>
      </div>
      <div class="flex justify-end">
        <button type="submit" class={btnPrimary} disabled={busy !== ''}>{busy === 'mail' ? t('settings.saving') : t('settings.save')}</button>
      </div>
    </form>

    {#if data.settings.smtp.ready}
      <form class="mt-4 flex flex-col gap-2 border-t border-zinc-800 pt-4 sm:flex-row" onsubmit={testMail}>
        <label class="sr-only" for="test-to">{t('settings.mail.test_label')}</label>
        <input id="test-to" type="email" class={input} bind:value={testTo} placeholder={t('settings.mail.test_placeholder')} required />
        <button type="submit" class="{btnGhost} shrink-0" disabled={busy !== ''}>{busy === 'testmail' ? t('settings.sending') : t('settings.mail.test_button')}</button>
      </form>
    {/if}
  </section>

  <section class="{card} p-5 sm:p-6">
    <h2 class="font-semibold">{t('settings.oauth.title')}</h2>
    <p class="mt-1 text-sm text-zinc-400">
      {t('settings.oauth.intro')}
    </p>

    <div class="mt-4 space-y-4">
      {#each oauthOrder.filter((n) => data.settings.oauth[n]) as name}
        {@const o = data.settings.oauth[name]}
        {@const on = o.client_id && o.secret_set}
        <form class="rounded-lg border border-zinc-800 p-4" onsubmit={(e) => saveLogin(e, name)}>
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h3 class="font-medium">{o.label}</h3>
            {#if on}
              <span class="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/15 px-2.5 py-1 text-xs font-medium text-emerald-300 ring-1 ring-emerald-500/25">
                <span class="size-1.5 rounded-full bg-emerald-400"></span> {t('settings.status.on')}
              </span>
            {:else}
              <span class="rounded-full bg-zinc-800 px-2.5 py-1 text-xs font-medium text-zinc-400">{t('settings.status.off')}</span>
            {/if}
          </div>

          <div class="mt-3">
            <span class={label}>{t('settings.oauth.redirect')} <span class="text-xs font-normal text-zinc-500">{t('settings.oauth.paste_at', { name: o.label })}</span></span>
            <div class="flex gap-2">
              <input class="{input} font-mono text-xs" value={o.redirect_uri} readonly onfocus={(e) => e.currentTarget.select()} aria-label={t('settings.oauth.redirect_aria', { name: o.label })} />
              <button type="button" class="{btnGhost} shrink-0" onclick={() => copy(o.redirect_uri)}>{t('settings.copy')}</button>
            </div>
          </div>
          <div class="mt-3 grid gap-3 sm:grid-cols-2">
            <div>
              <label class={label} for="oauth-id-{name}">{t('settings.oauth.client_id')}</label>
              <input id="oauth-id-{name}" class={input} bind:value={oauthDraft[name].client_id} autocomplete="off" autocapitalize="off" spellcheck="false" required />
            </div>
            <div>
              <label class={label} for="oauth-secret-{name}">{t('settings.oauth.client_secret')}</label>
              <input id="oauth-secret-{name}" type="password" class={input} bind:value={oauthDraft[name].client_secret} autocomplete="new-password"
                placeholder={o.secret_set ? t('settings.secret_saved') : ''} required={!o.secret_set} />
            </div>
          </div>

          <div class="mt-3 flex flex-wrap items-center justify-between gap-2">
            <button type="button" class="text-sm font-medium text-indigo-400 hover:text-indigo-300" onclick={() => (howTo = howTo === name ? '' : name)}>
              {howTo === name ? t('settings.oauth.hide') : t('settings.oauth.how')}
            </button>
            <div class="flex gap-2">
              {#if o.client_id}<button type="button" class={btnDanger} onclick={() => removeLogin(name)} disabled={busy !== ''}>{t('settings.oauth.turn_off')}</button>{/if}
              <button type="submit" class={btnPrimary} disabled={busy !== ''}>{busy === 'oauth-' + name ? t('settings.saving') : t('settings.save')}</button>
            </div>
          </div>

          {#if howTo === name}
            <ol class="mt-3 list-decimal space-y-1.5 rounded-lg bg-zinc-950/50 p-4 ps-8 text-sm text-zinc-300">
              {#if name === 'discord'}
                <li>{t('settings.discord.step1_a')} <span class="font-mono text-xs">discord.com/developers/applications</span> {t('settings.discord.step1_b')} <b>New Application</b>{t('settings.sentence_end')}</li>
                <li>{t('settings.discord.step2_a')} <b>OAuth2</b> {t('settings.discord.step2_b')} <b>Redirects</b>{t('settings.discord.step2_c')} <b>Add Redirect</b> {t('settings.discord.step2_d')}</li>
                <li>{t('settings.discord.step3_a')} <b>Client ID</b>{t('settings.discord.step3_b')} <b>Reset Secret</b> {t('settings.discord.step3_c')} <b>Client Secret</b>{t('settings.sentence_end')}</li>
                <li>{t('settings.discord.step4')}</li>
              {:else}
                <li>{t('settings.google.step1_a')} <span class="font-mono text-xs">console.cloud.google.com</span>{t('settings.google.step1_b')} <b>APIs &amp; Services → OAuth consent screen</b>{t('settings.google.step1_c')} <b>External</b>{t('settings.google.step1_d')}</li>
                <li>{t('settings.google.step2_a')} <b>Credentials → Create credentials → OAuth client ID</b>{t('settings.google.step2_b')} <b>Web application</b>{t('settings.sentence_end')}</li>
                <li>{t('settings.google.step3_a')} <b>Authorized redirect URIs</b>{t('settings.google.step3_b')}</li>
                <li>{t('settings.google.step4_a')} <b>Client ID</b> {t('settings.google.step4_b')} <b>Client secret</b>{t('settings.google.step4_c')}</li>
              {/if}
            </ol>
          {/if}
        </form>
      {/each}
    </div>
  </section>
  {/if}

  <section class="{card} p-5 sm:p-6">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h2 class="font-semibold">{t('settings.webhook.title')}</h2>
        <p class="mt-1 text-sm text-zinc-400">{t('settings.webhook.intro')}</p>
      </div>
      {#if data.settings.discord_webhook_set}
        <span class="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/15 px-2.5 py-1 text-xs font-medium text-emerald-300 ring-1 ring-emerald-500/25">
          <span class="size-1.5 rounded-full bg-emerald-400"></span> {t('settings.status.connected')}
        </span>
      {:else}
        <span class="rounded-full bg-zinc-800 px-2.5 py-1 text-xs font-medium text-zinc-400">{t('settings.status.not_set')}</span>
      {/if}
    </div>

    <form class="mt-4 space-y-2" onsubmit={saveWebhook}>
      <label class={label} for="webhook">{data.settings.discord_webhook_set ? t('settings.webhook.replace_label') : t('settings.webhook.label')}</label>
      <div class="flex flex-col gap-2 sm:flex-row">
        <input id="webhook" type="password" class={input} bind:value={webhook} placeholder="https://discord.com/api/webhooks/…" autocomplete="off" required />
        <button type="submit" class="{btnPrimary} shrink-0" disabled={busy !== ''}>{t('settings.save')}</button>
      </div>
      <p class="text-xs text-zinc-500">{t('settings.webhook.help')}</p>
    </form>

    {#if data.settings.discord_webhook_set}
      <div class="mt-4 flex gap-2 border-t border-zinc-800 pt-4">
        <button class={btnGhost} onclick={testWebhook} disabled={busy !== ''}>{busy === 'test' ? t('settings.sending') : t('settings.webhook.test_button')}</button>
        <button class={btnDanger} onclick={removeWebhook} disabled={busy !== ''}>{t('settings.remove')}</button>
      </div>
    {/if}
  </section>

  <section class="{card} p-5 sm:p-6">
    <h2 class="font-semibold">{t('settings.info.title')}</h2>
    <dl class="mt-3 grid gap-x-6 gap-y-2 text-sm sm:grid-cols-[auto_1fr]">
      <dt class="text-zinc-500">{t('settings.info.url')}</dt>
      <dd class="font-mono text-zinc-300">{data.settings.app_url}</dd>
      <dt class="text-zinc-500">{t('settings.info.login_on')}</dt>
      <dd class="font-mono text-zinc-300">{t('settings.info.login_on_value', { domain: data.settings.base_domain })}</dd>
    </dl>
    <p class="mt-3 text-xs text-zinc-500">{t('settings.info.help')}</p>
  </section>

  {#if data.me.admin}
    <section class="{card} p-5 sm:p-6">
      <h2 class="font-semibold">{t('settings.backup.title')}</h2>
      <p class="mt-1 text-sm text-zinc-400">
        {t('settings.backup.intro')}
      </p>
      <div class="mt-4 flex flex-wrap items-center gap-3">
        <button class={btnGhost} onclick={downloadBackup} disabled={busy !== ''}>{busy === 'backup' ? t('settings.backup.preparing') : t('settings.backup.download')}</button>
        <span class="text-xs text-zinc-500">{t('settings.backup.cli')} <span class="font-mono">docker exec turkushan-auth /turkushan-auth backup</span></span>
      </div>
    </section>
  {/if}
</div>
