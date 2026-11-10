<script>
  import { api } from '../../lib/api.js';
  import { toast } from '../../lib/state.svelte.js';
  import AdminsCard from './AdminsCard.svelte';
  import GroupCard from './GroupCard.svelte';
  import { card, btnGhost, btnPrimary, input, label } from './ui.js';
  import { can } from './rights.js';

  let { data, reload } = $props();

  let expanded = $state(null); // group id or 'admins'
  let busy = $state(false);
  let creating = $state(false);
  let newName = $state('');
  let newDescription = $state('');

  const manageGroups = $derived(can(data, 'groups'));

  // Shared by the cards: one request at a time, a toast, and fresh data afterwards.
  async function run(req, okText) {
    busy = true;
    const r = await req;
    busy = false;
    if (!r.ok) {
      toast(r.data.error, 'error');
      reload();
      return false;
    }
    if (okText) toast(okText);
    await reload();
    return true;
  }

  async function create(e) {
    e.preventDefault();
    if (await run(api('/admin/groups', { name: newName, description: newDescription }), `Group ${newName} created. Now choose what it may do.`)) {
      expanded = data.groups.find((g) => g.name.toLowerCase() === newName.trim().toLowerCase())?.id ?? null;
      creating = false;
      newName = newDescription = '';
    }
  }

  const toggle = (id) => (expanded = expanded === id ? null : id);
</script>

<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
  <div>
    <h2 class="text-lg font-semibold tracking-tight">Groups</h2>
    <p class="text-sm text-zinc-400">A group gives its members access to sites and rights in this panel. A user can be in several groups.</p>
  </div>
  {#if manageGroups && !creating}
    <button class={btnPrimary} onclick={() => (creating = true)}>
      <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg>
      New group
    </button>
  {/if}
</div>

{#if creating}
  <form class="{card} mb-4 space-y-4 p-5" onsubmit={create}>
    <h3 class="font-semibold">New group</h3>
    <div class="grid gap-4 sm:grid-cols-2">
      <div>
        <label class={label} for="group-name">Name</label>
        <input id="group-name" class={input} bind:value={newName} maxlength="40" placeholder="Friends" required />
      </div>
      <div>
        <label class={label} for="group-desc">Description <span class="text-xs font-normal text-zinc-500">optional</span></label>
        <input id="group-desc" class={input} bind:value={newDescription} maxlength="200" placeholder="Who is this group for?" />
      </div>
    </div>
    <div class="flex justify-end gap-2">
      <button type="button" class={btnGhost} onclick={() => (creating = false)}>Cancel</button>
      <button type="submit" class={btnPrimary} disabled={busy}>Create group</button>
    </div>
  </form>
{/if}

<ul class="space-y-2">
  <AdminsCard {data} {busy} {run} open={expanded === 'admins'} ontoggle={() => toggle('admins')} />
  {#each data.groups as g (g.id)}
    <GroupCard {data} {g} {busy} {run} open={expanded === g.id} ontoggle={() => toggle(g.id)} />
  {:else}
    <li class="{card} p-8 text-center text-sm text-zinc-500">
      No groups yet. Make one, like "Friends", tick the sites it may open, and add people.
    </li>
  {/each}
</ul>
