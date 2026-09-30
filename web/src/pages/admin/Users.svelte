<script>
  import { api } from '../../lib/api.js';
  import { toast } from '../../lib/state.svelte.js';
  import { timeAgo, fullDate } from '../../lib/format.js';
  import Modal from '../../components/Modal.svelte';
  import Field from '../../components/Field.svelte';
  import { card, btnApprove, btnDanger, btnGhost, btnPrimary, badge, statusBadge } from './ui.js';

  let { data, reload } = $props();

  let search = $state('');
  let expanded = $state(null);
  let busy = $state(false);

  const users = $derived(
    data.users.filter((u) => !search || u.username.includes(search.toLowerCase()) || u.email.includes(search.toLowerCase())),
  );
  const accessOf = $derived(new Map(data.access.map((a) => [`${a.user_id}:${a.site_id}`, a.status])));
  const approvalSites = $derived(data.sites.filter((s) => s.require_approval));

  function canManage(u) {
    return !u.is_admin && u.id !== data.me.id;
  }

  async function run(req, okText) {
    busy = true;
    const r = await req;
    busy = false;
    if (!r.ok) {
      toast(r.data.error, 'error');
      return false;
    }
    toast(okText);
    reload();
    return true;
  }

  const setAccess = (u, site, status) =>
    run(
      api('/admin/access', { user_id: u.id, site_id: site.id, status }, 'PUT'),
      status === 'approved' ? `${u.username} can now open ${site.name}.` : `${u.username} no longer has access to ${site.name}.`,
    );

  const setStatus = (u, status) =>
    run(api(`/admin/users/${u.id}/status`, { status }, 'PUT'), status === 'blocked' ? `${u.username} is blocked.` : `${u.username} is unblocked.`);

  function remove(u) {
    if (!confirm(`Delete ${u.username}? Their account and all their access are removed. This cannot be undone.`)) return;
    run(api(`/admin/users/${u.id}`, undefined, 'DELETE'), `${u.username} was deleted.`);
  }

  // Reset-password dialog
  let resetUser = $state(null);
  let resetOpen = $state(false);
  let newPassword = $state('');
  let resetError = $state('');

  function openReset(u) {
    resetUser = u;
    newPassword = '';
    resetError = '';
    resetOpen = true;
  }

  async function submitReset(e) {
    e.preventDefault();
    busy = true;
    const r = await api(`/admin/users/${resetUser.id}/password`, { password: newPassword }, 'PUT');
    busy = false;
    if (!r.ok) {
      resetError = r.data.error;
      return;
    }
    resetOpen = false;
    toast(`New password set for ${resetUser.username}. Their sessions were signed out.`);
    reload();
  }
</script>

<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
  <h2 class="text-lg font-semibold tracking-tight">Users</h2>
  <input
    type="search"
    placeholder="Search users…"
    bind:value={search}
    class="w-full rounded-lg border border-zinc-800 bg-zinc-900/60 px-3 py-1.5 text-sm outline-none placeholder:text-zinc-600 focus:border-indigo-500 sm:w-64"
  />
</div>

<ul class="space-y-2">
  {#each users as u (u.id)}
    {@const open = expanded === u.id}
    <li class="{card} overflow-hidden">
      <button
        type="button"
        class="flex w-full items-center gap-3 p-4 text-left transition hover:bg-zinc-800/30"
        onclick={() => (expanded = open ? null : u.id)}
        aria-expanded={open}
      >
        <div class="grid size-10 shrink-0 place-items-center rounded-full text-sm font-semibold uppercase {u.is_admin ? 'bg-indigo-500/20 text-indigo-300' : 'bg-zinc-800 text-zinc-300'}">
          {u.username.slice(0, 1)}
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-1.5">
            <span class="font-medium">{u.username}</span>
            {#if u.is_admin}<span class="{badge} bg-indigo-500/15 text-indigo-300 ring-indigo-400/20">Admin</span>{/if}
            {#if u.status === 'blocked'}<span class="{badge} {statusBadge.denied}">Blocked</span>{/if}
            {#if u.locked}<span class="{badge} {statusBadge.pending}">Locked</span>{/if}
          </div>
          <p class="mt-0.5 truncate text-xs text-zinc-500">
            {u.email ? (u.email_verified ? u.email : `${u.email} (not verified)`) : 'no email'} · last login {timeAgo(u.last_login)}
          </p>
        </div>
        <svg viewBox="0 0 24 24" class="size-4 shrink-0 text-zinc-500 transition {open ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
      </button>

      {#if open}
        <div class="space-y-5 border-t border-zinc-800 bg-zinc-950/40 p-4">
          <div>
            <h3 class="mb-2 text-xs font-semibold uppercase tracking-wider text-zinc-500">Site access</h3>
            {#if u.is_admin}
              <p class="text-sm text-zinc-400">Admins can open every site.</p>
            {:else if approvalSites.length === 0}
              <p class="text-sm text-zinc-400">No sites need approval. Add sites in the Sites tab.</p>
            {:else}
              <div class="divide-y divide-zinc-800/80 rounded-lg border border-zinc-800">
                {#each approvalSites as site (site.id)}
                  {@const status = accessOf.get(`${u.id}:${site.id}`)}
                  <div class="flex flex-wrap items-center gap-x-3 gap-y-2 px-3 py-2.5">
                    <div class="min-w-0 flex-1 text-sm">
                      <span class="font-medium">{site.name}</span>
                      <span class="text-zinc-500">{site.host}</span>
                    </div>
                    {#if status}
                      <span class="{badge} {statusBadge[status]}">{status}</span>
                    {:else}
                      <span class="text-xs text-zinc-500">no access</span>
                    {/if}
                    {#if status === 'approved'}
                      <button class={btnGhost} disabled={busy} onclick={() => setAccess(u, site, 'denied')}>Revoke</button>
                    {:else}
                      {#if status === 'pending'}
                        <button class={btnDanger} disabled={busy} onclick={() => setAccess(u, site, 'denied')}>Deny</button>
                      {/if}
                      <button class={btnApprove} disabled={busy} onclick={() => setAccess(u, site, 'approved')}>Give access</button>
                    {/if}
                  </div>
                {/each}
              </div>
            {/if}
          </div>

          <div class="flex flex-wrap items-center justify-between gap-3">
            <p class="text-xs text-zinc-500" title={fullDate(u.created_at)}>Joined {timeAgo(u.created_at)}</p>
            {#if canManage(u)}
              <div class="flex flex-wrap gap-2">
                <button class={btnGhost} disabled={busy} onclick={() => openReset(u)}>Reset password</button>
                {#if u.status === 'blocked'}
                  <button class={btnApprove} disabled={busy} onclick={() => setStatus(u, 'active')}>Unblock</button>
                {:else}
                  <button class={btnGhost} disabled={busy} onclick={() => setStatus(u, 'blocked')}>Block</button>
                {/if}
                {#if u.locked && u.status !== 'blocked'}
                  <button class={btnGhost} disabled={busy} onclick={() => setStatus(u, 'active')}>Unlock</button>
                {/if}
                <button class={btnDanger} disabled={busy} onclick={() => remove(u)}>Delete</button>
              </div>
            {:else}
              <p class="text-xs text-zinc-500">Admin accounts are managed via the container settings.</p>
            {/if}
          </div>
        </div>
      {/if}
    </li>
  {:else}
    <li class="{card} p-8 text-center text-sm text-zinc-500">No users match "{search}".</li>
  {/each}
</ul>

<Modal bind:open={resetOpen} title="Reset password for {resetUser?.username}">
  <form class="space-y-4" onsubmit={submitReset}>
    <p class="text-sm text-zinc-400">
      Set a new password and send it to them yourself. They're signed out everywhere and can log in with the new password.
    </p>
    <Field label="New password" type="password" autocomplete="new-password" required bind:value={newPassword}
      hint="At least 6 characters, 1 capital letter and 1 symbol." />
    {#if resetError}<p class="text-sm text-rose-400">{resetError}</p>{/if}
    <div class="flex justify-end gap-2">
      <button type="button" class={btnGhost} onclick={() => (resetOpen = false)}>Cancel</button>
      <button type="submit" class={btnPrimary} disabled={busy}>Set password</button>
    </div>
  </form>
</Modal>
