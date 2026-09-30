<script>
  import { untrack } from 'svelte';
  import { api } from '../../lib/api.js';
  import { toast } from '../../lib/state.svelte.js';
  import { card, btnGhost, btnPrimary, btnDanger, input, label } from './ui.js';

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
      'Email settings saved.',
      'mail',
    );
    if (ok) smtpPassword = '';
  }

  async function testMail(e) {
    e.preventDefault();
    busy = 'testmail';
    const r = await api('/admin/settings/test-mail', { to: testTo });
    busy = '';
    toast(r.ok ? `Test mail sent to ${testTo}. Check the inbox (and spam).` : r.data.error, r.ok ? 'success' : 'error');
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
    save({ portal_url: portalURL }, 'Portal address saved. The NPM configs are updated.', 'portal');
  }

  async function saveWebhook(e) {
    e.preventDefault();
    if (await save({ discord_webhook: webhook }, 'Discord webhook saved.', 'webhook')) webhook = '';
  }

  function removeWebhook() {
    if (confirm('Remove the Discord webhook? You will no longer get notifications.')) {
      save({ discord_webhook: '' }, 'Discord webhook removed.', 'webhook');
    }
  }

  async function testWebhook() {
    busy = 'test';
    const r = await api('/admin/settings/test-discord', {});
    busy = '';
    toast(r.ok ? 'Test message sent. Check your Discord channel.' : r.data.error, r.ok ? 'success' : 'error');
  }
</script>

<div class="space-y-6">
  <section class="{card} p-5 sm:p-6">
    <h2 class="font-semibold">Portal address for NPM</h2>
    <p class="mt-1 text-sm text-zinc-400">
      How Nginx Proxy Manager reaches this container. It's filled into the NPM config of every site.
    </p>
    <form class="mt-4 flex flex-col gap-2 sm:flex-row" onsubmit={savePortal}>
      <label class="sr-only" for="portal-url">Portal address</label>
      <input id="portal-url" class={input} bind:value={portalURL} placeholder="http://192.168.0.6:3010" autocapitalize="off" spellcheck="false" />
      <button type="submit" class="{btnPrimary} shrink-0" disabled={busy !== ''}>Save</button>
    </form>
  </section>

  <section class="{card} p-5 sm:p-6">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h2 class="font-semibold">Email</h2>
        <p class="mt-1 text-sm text-zinc-400">Used for email verification and "forgot password" mails.</p>
      </div>
      {#if data.settings.smtp.ready}
        <span class="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/15 px-2.5 py-1 text-xs font-medium text-emerald-300 ring-1 ring-emerald-500/25">
          <span class="size-1.5 rounded-full bg-emerald-400"></span> Set up
        </span>
      {:else}
        <span class="rounded-full bg-zinc-800 px-2.5 py-1 text-xs font-medium text-zinc-400">Not set</span>
      {/if}
    </div>

    <form class="mt-4 space-y-4" onsubmit={saveMail}>
      <div class="flex flex-wrap items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-950/40 px-3 py-2.5 text-sm text-zinc-400">
        Using Resend?
        <button type="button" class={btnGhost} onclick={useResend}>Fill in Resend</button>
        <span class="text-xs text-zinc-500">then paste your Resend API key as the password.</span>
      </div>
      <div class="grid gap-4 sm:grid-cols-[1fr_8rem]">
        <div>
          <label class={label} for="smtp-host">Mail server</label>
          <input id="smtp-host" class={input} bind:value={smtpHost} placeholder="smtp.resend.com" autocapitalize="off" spellcheck="false" />
        </div>
        <div>
          <label class={label} for="smtp-port">Port</label>
          <input id="smtp-port" class={input} type="number" min="1" max="65535" bind:value={smtpPort} />
        </div>
      </div>
      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label class={label} for="smtp-user">Username</label>
          <input id="smtp-user" class={input} bind:value={smtpUser} placeholder="resend" autocomplete="off" autocapitalize="off" spellcheck="false" />
        </div>
        <div>
          <label class={label} for="smtp-pass">Password / API key</label>
          <input id="smtp-pass" class={input} type="password" bind:value={smtpPassword} autocomplete="new-password"
            placeholder={data.settings.smtp.password_set ? '•••••••• saved, leave empty to keep' : 're_…'} />
        </div>
      </div>
      <div>
        <label class={label} for="smtp-from">Send as</label>
        <input id="smtp-from" class={input} bind:value={smtpFrom} placeholder="noreply@{data.settings.base_domain}" autocapitalize="off" spellcheck="false" />
        <p class="mt-1.5 text-xs text-zinc-500">
          Must be on a domain you verified at your mail provider. Add a name like <span class="font-mono">{data.settings.base_domain} &lt;noreply@{data.settings.base_domain}&gt;</span>.
        </p>
      </div>
      <div class="flex justify-end">
        <button type="submit" class={btnPrimary} disabled={busy !== ''}>{busy === 'mail' ? 'Saving…' : 'Save'}</button>
      </div>
    </form>

    {#if data.settings.smtp.ready}
      <form class="mt-4 flex flex-col gap-2 border-t border-zinc-800 pt-4 sm:flex-row" onsubmit={testMail}>
        <label class="sr-only" for="test-to">Send a test mail to</label>
        <input id="test-to" type="email" class={input} bind:value={testTo} placeholder="Send a test mail to…" required />
        <button type="submit" class="{btnGhost} shrink-0" disabled={busy !== ''}>{busy === 'testmail' ? 'Sending…' : 'Send test mail'}</button>
      </form>
    {/if}
  </section>

  <section class="{card} p-5 sm:p-6">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h2 class="font-semibold">Discord notifications</h2>
        <p class="mt-1 text-sm text-zinc-400">Get a message when someone creates an account or asks for access to a site.</p>
      </div>
      {#if data.settings.discord_webhook_set}
        <span class="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/15 px-2.5 py-1 text-xs font-medium text-emerald-300 ring-1 ring-emerald-500/25">
          <span class="size-1.5 rounded-full bg-emerald-400"></span> Connected
        </span>
      {:else}
        <span class="rounded-full bg-zinc-800 px-2.5 py-1 text-xs font-medium text-zinc-400">Not set</span>
      {/if}
    </div>

    <form class="mt-4 space-y-2" onsubmit={saveWebhook}>
      <label class={label} for="webhook">{data.settings.discord_webhook_set ? 'Replace webhook URL' : 'Webhook URL'}</label>
      <div class="flex flex-col gap-2 sm:flex-row">
        <input id="webhook" type="password" class={input} bind:value={webhook} placeholder="https://discord.com/api/webhooks/…" autocomplete="off" required />
        <button type="submit" class="{btnPrimary} shrink-0" disabled={busy !== ''}>Save</button>
      </div>
      <p class="text-xs text-zinc-500">In Discord: channel settings → Integrations → Webhooks → New Webhook → Copy Webhook URL.</p>
    </form>

    {#if data.settings.discord_webhook_set}
      <div class="mt-4 flex gap-2 border-t border-zinc-800 pt-4">
        <button class={btnGhost} onclick={testWebhook} disabled={busy !== ''}>{busy === 'test' ? 'Sending…' : 'Send test message'}</button>
        <button class={btnDanger} onclick={removeWebhook} disabled={busy !== ''}>Remove</button>
      </div>
    {/if}
  </section>

  <section class="{card} p-5 sm:p-6">
    <h2 class="font-semibold">Portal</h2>
    <dl class="mt-3 grid gap-x-6 gap-y-2 text-sm sm:grid-cols-[auto_1fr]">
      <dt class="text-zinc-500">Portal URL</dt>
      <dd class="font-mono text-zinc-300">{data.settings.app_url}</dd>
      <dt class="text-zinc-500">Login works on</dt>
      <dd class="font-mono text-zinc-300">{data.settings.base_domain} and *.{data.settings.base_domain}</dd>
    </dl>
    <p class="mt-3 text-xs text-zinc-500">These come from the container settings in Unraid (APP_URL, COOKIE_DOMAIN).</p>
  </section>
</div>
