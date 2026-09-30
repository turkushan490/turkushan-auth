<script>
  import { api } from '../../lib/api.js';
  import { link, toast } from '../../lib/state.svelte.js';
  import Toggle from '../../components/Toggle.svelte';
  import CodeBlock from '../../components/CodeBlock.svelte';
  import SiteForm from './SiteForm.svelte';
  import { card, btnGhost, btnPrimary, btnDanger } from './ui.js';

  let { data, reload } = $props();

  let adding = $state(false);
  let editing = $state(null);
  let showConfig = $state(null);

  async function create(values) {
    const r = await api('/admin/sites', values);
    if (!r.ok) return r.data.error;
    adding = false;
    toast(`${values.name} added. Copy its NPM config below.`);
    await reload();
    showConfig = data.sites.find((s) => s.name === values.name.trim())?.id ?? null;
    return '';
  }

  async function update(site, values) {
    const r = await api(`/admin/sites/${site.id}`, values, 'PUT');
    if (!r.ok) return r.data.error;
    editing = null;
    toast(`${values.name} saved.`);
    reload();
    return '';
  }

  async function flip(site, field, value) {
    const r = await api(`/admin/sites/${site.id}`, { ...site, [field]: value }, 'PUT');
    if (!r.ok) {
      toast(r.data.error, 'error');
    } else {
      toast(`${site.name} updated.`);
    }
    reload();
  }

  async function remove(site) {
    if (!confirm(`Remove ${site.name}? Everyone's access to it is removed too. Also remove its config in NPM, or visitors will get an error.`)) return;
    const r = await api(`/admin/sites/${site.id}`, undefined, 'DELETE');
    if (!r.ok) return toast(r.data.error, 'error');
    toast(`${site.name} removed.`);
    reload();
  }
</script>

<div class="mb-4 flex items-center justify-between gap-3">
  <h2 class="text-lg font-semibold tracking-tight">Sites</h2>
  {#if !adding}
    <button class={btnPrimary} onclick={() => (adding = true)}>
      <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg>
      Add site
    </button>
  {/if}
</div>

{#if !data.settings.portal_url}
  <div class="mb-4 rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-200">
    Set the <a href="/admin/settings" onclick={link} class="font-medium underline underline-offset-2">portal address</a> first, so the NPM config below is complete.
  </div>
{/if}

{#if adding}
  <div class="{card} mb-4 p-5">
    <h3 class="mb-4 font-semibold">New site</h3>
    <SiteForm base={data.settings.base_domain} submitText="Add site" onsave={create} oncancel={() => (adding = false)} />
  </div>
{/if}

{#if data.sites.length === 0 && !adding}
  <div class="{card} flex flex-col items-center px-6 py-12 text-center">
    <p class="font-medium">No sites yet</p>
    <p class="mt-1 max-w-sm text-sm text-zinc-500">
      Add a site like manga.{data.settings.base_domain}, then paste its NPM config into the proxy host's Advanced tab.
    </p>
  </div>
{/if}

<ul class="space-y-3">
  {#each data.sites as site (site.id)}
    <li class="{card} p-5">
      {#if editing === site.id}
        <h3 class="mb-4 font-semibold">Edit {site.name}</h3>
        <SiteForm initial={site} base={data.settings.base_domain} onsave={(v) => update(site, v)} oncancel={() => (editing = null)} />
      {:else}
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0">
            <h3 class="font-semibold">{site.name}</h3>
            <a href="https://{site.host}" target="_blank" rel="noopener noreferrer" class="text-sm text-indigo-400 hover:text-indigo-300">{site.host} ↗</a>
            <p class="mt-1 font-mono text-xs text-zinc-500">→ {site.upstream}</p>
          </div>
          <div class="flex gap-2">
            <button class={btnGhost} onclick={() => (editing = site.id)}>Edit</button>
            <button class={btnDanger} onclick={() => remove(site)}>Remove</button>
          </div>
        </div>

        <div class="mt-4 grid gap-4 rounded-lg border border-zinc-800 bg-zinc-950/40 p-4 sm:grid-cols-2">
          <Toggle checked={site.require_approval} label="Needs approval" description={site.require_approval ? 'Only users you approve.' : 'Everyone who is signed in.'}
            onchange={(v) => flip(site, 'require_approval', v)} />
          <Toggle checked={site.require_email} label="Needs a verified email" description={site.require_email ? 'Verified email required.' : 'No email needed.'}
            onchange={(v) => flip(site, 'require_email', v)} />
        </div>

        <button
          type="button"
          class="mt-4 flex items-center gap-1.5 text-sm font-medium text-zinc-300 hover:text-zinc-100"
          onclick={() => (showConfig = showConfig === site.id ? null : site.id)}
          aria-expanded={showConfig === site.id}
        >
          <svg viewBox="0 0 24 24" class="size-4 transition {showConfig === site.id ? 'rotate-90' : ''}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 6l6 6-6 6" /></svg>
          NPM config
        </button>
        {#if showConfig === site.id}
          <div class="mt-3 space-y-3">
            <ol class="list-decimal space-y-1 pl-5 text-sm text-zinc-400">
              <li>In NPM, open the proxy host for <span class="font-medium text-zinc-200">{site.host}</span>.</li>
              <li>Go to the <span class="font-medium text-zinc-200">Advanced</span> tab and replace everything there with this config.</li>
              <li>Save, then open {site.host} in a private window to test.</li>
            </ol>
            <CodeBlock code={site.snippet} title="NPM → Proxy Host → Advanced → Custom Nginx Configuration" />
          </div>
        {/if}
      {/if}
    </li>
  {/each}
</ul>
