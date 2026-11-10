<script>
  import { api } from '../../lib/api.js';
  import { link, toast } from '../../lib/state.svelte.js';
  import Toggle from '../../components/Toggle.svelte';
  import CodeBlock from '../../components/CodeBlock.svelte';
  import SiteForm from './SiteForm.svelte';
  import { confirmDialog } from '../../lib/confirm.svelte.js';
  import { card, btnGhost, btnPrimary, btnDanger } from './ui.js';
  import { t } from '../../lib/i18n.svelte.js';

  let { data, reload } = $props();

  let adding = $state(false);
  let editing = $state(null);
  let showConfig = $state(null);

  async function create(values) {
    const r = await api('/admin/sites', values);
    if (!r.ok) return r.data.error;
    adding = false;
    toast(t('sites.added', { name: values.name }));
    await reload();
    showConfig = data.sites.find((s) => s.name === values.name.trim())?.id ?? null;
    return '';
  }

  async function update(site, values) {
    const r = await api(`/admin/sites/${site.id}`, values, 'PUT');
    if (!r.ok) return r.data.error;
    editing = null;
    toast(t('sites.saved', { name: values.name }));
    reload();
    return '';
  }

  async function flip(site, field, value) {
    const r = await api(`/admin/sites/${site.id}`, { ...site, [field]: value }, 'PUT');
    if (!r.ok) {
      toast(r.data.error, 'error');
    } else {
      toast(t('sites.updated', { name: site.name }));
    }
    reload();
  }

  async function remove(site) {
    if (!(await confirmDialog({ title: t('sites.remove_title', { name: site.name }), text: t('sites.remove_text'), ok: t('sites.remove'), danger: true }))) return;
    const r = await api(`/admin/sites/${site.id}`, undefined, 'DELETE');
    if (!r.ok) return toast(r.data.error, 'error');
    toast(t('sites.removed', { name: site.name }));
    reload();
  }
</script>

<div class="mb-4 flex items-center justify-between gap-3">
  <h2 class="text-lg font-semibold tracking-tight">{t('sites.title')}</h2>
  {#if !adding}
    <button class={btnPrimary} onclick={() => (adding = true)}>
      <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg>
      {t('sites.add')}
    </button>
  {/if}
</div>

{#if !data.settings.portal_url}
  <div class="mb-4 rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-200">
    {t('sites.portal_missing_before')} <a href="/admin/settings" onclick={link} class="font-medium underline underline-offset-2">{t('sites.portal_missing_link')}</a> {t('sites.portal_missing_after')}
  </div>
{/if}

{#if adding}
  <div class="{card} mb-4 p-5">
    <h3 class="mb-4 font-semibold">{t('sites.new')}</h3>
    <SiteForm base={data.settings.base_domain} canOpen={data.me.admin} submitText={t('sites.add')} onsave={create} oncancel={() => (adding = false)} />
  </div>
{/if}

{#if data.sites.length === 0 && !adding}
  <div class="{card} flex flex-col items-center px-6 py-12 text-center">
    <p class="font-medium">{t('sites.empty_title')}</p>
    <p class="mt-1 max-w-sm text-sm text-zinc-500">
      {t('sites.empty_text', { example: `manga.${data.settings.base_domain}` })}
    </p>
  </div>
{/if}

<ul class="space-y-3">
  {#each data.sites as site (site.id)}
    <li class="{card} p-5">
      {#if editing === site.id}
        <h3 class="mb-4 font-semibold">{t('sites.edit_title', { name: site.name })}</h3>
        <SiteForm initial={site} base={data.settings.base_domain} canOpen={data.me.admin} onsave={(v) => update(site, v)} oncancel={() => (editing = null)} />
      {:else}
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0">
            <h3 class="font-semibold">{site.name}</h3>
            <a href="https://{site.host}" target="_blank" rel="noopener noreferrer" class="text-sm text-indigo-400 hover:text-indigo-300">{site.host} ↗</a>
            <p class="mt-1 font-mono text-xs text-zinc-500">→ {site.upstream}</p>
          </div>
          <div class="flex gap-2">
            <button class={btnGhost} onclick={() => (editing = site.id)}>{t('sites.edit')}</button>
            <button class={btnDanger} onclick={() => remove(site)}>{t('sites.remove')}</button>
          </div>
        </div>

        <div class="mt-4 grid gap-4 rounded-lg border border-zinc-800 bg-zinc-950/40 p-4 sm:grid-cols-2">
          <Toggle checked={site.require_approval} label={t('sites.needs_approval')} disabled={!data.me.admin && site.require_approval} description={site.require_approval ? t('sites.approval_on') : t('sites.approval_off')}
            onchange={(v) => flip(site, 'require_approval', v)} />
          <Toggle checked={site.require_email} label={t('sites.needs_email')} description={site.require_email ? t('sites.email_on') : t('sites.email_off')}
            onchange={(v) => flip(site, 'require_email', v)} />
        </div>

        <button
          type="button"
          class="mt-4 flex items-center gap-1.5 text-sm font-medium text-zinc-300 hover:text-zinc-100"
          onclick={() => (showConfig = showConfig === site.id ? null : site.id)}
          aria-expanded={showConfig === site.id}
        >
          <svg viewBox="0 0 24 24" class="size-4 transition {showConfig === site.id ? 'rotate-90' : ''}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 6l6 6-6 6" /></svg>
          {t('sites.npm_config')}
        </button>
        {#if showConfig === site.id}
          <div class="mt-3 space-y-3">
            <ol class="list-decimal space-y-1 ps-5 text-sm text-zinc-400">
              <li>{t('sites.step1_before')} <span class="font-medium text-zinc-200">{site.host}</span>{t('sites.step1_after')}</li>
              <li>{t('sites.step2_before')} <span class="font-medium text-zinc-200">Advanced</span> {t('sites.step2_after')}</li>
              <li>{t('sites.step3', { host: site.host })}</li>
            </ol>
            <CodeBlock code={site.snippet} title="NPM → Proxy Host → Advanced → Custom Nginx Configuration" />
          </div>
        {/if}
      {/if}
    </li>
  {/each}
</ul>
