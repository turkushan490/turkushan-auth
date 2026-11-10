<script>
  import { api } from '../../lib/api.js';
  import { toast } from '../../lib/state.svelte.js';
  import { confirmDialog } from '../../lib/confirm.svelte.js';
  import UserFilters from './UserFilters.svelte';
  import UserRow from './UserRow.svelte';
  import ResetPasswordModal from './ResetPasswordModal.svelte';
  import CreateUserModal from './CreateUserModal.svelte';
  import { card, btnGhost, btnPrimary } from './ui.js';
  import { can, standing, opens, groupsOf } from './rights.js';

  let { data, reload } = $props();

  // --- filters, kept in the address (#q=…&site=…) so a reload or a shared link keeps them ---
  const fromHash = new URLSearchParams(location.hash.slice(1));
  let search = $state(fromHash.get('q') || '');
  let siteFilter = $state(fromHash.get('site') || '');
  let groupFilter = $state(fromHash.get('group') || '');
  let loginFilter = $state(fromHash.get('login') || '');
  let statusFilter = $state(fromHash.get('status') || '');
  let sortBy = $state(fromHash.get('sort') || 'name');
  let expanded = $state(Number(fromHash.get('open')) || null);

  $effect(() => {
    const p = new URLSearchParams();
    for (const [k, v] of Object.entries({ q: search, site: siteFilter, group: groupFilter, login: loginFilter, status: statusFilter, open: expanded || '' })) {
      if (v) p.set(k, v);
    }
    if (sortBy !== 'name') p.set('sort', sortBy);
    const hash = p.toString();
    if (hash !== location.hash.slice(1)) history.replaceState(history.state, '', location.pathname + location.search + (hash ? '#' + hash : ''));
  });

  const filterSite = $derived(data.sites.find((s) => String(s.id) === siteFilter));
  const pendingUsers = $derived(new Set(data.access.filter((a) => a.status === 'pending').map((a) => a.user_id)));
  const identitiesByUser = $derived(Map.groupBy ? Map.groupBy(data.identities, (i) => i.user_id) : new Map());
  const loginsOf = (u) => identitiesByUser.get(u.id) || data.identities.filter((i) => i.user_id === u.id);
  const sitesOf = (u) => data.sites.filter((s) => opens(standing(data, u, s)));

  const users = $derived.by(() => {
    const q = search.trim().toLowerCase();
    const list = data.users.filter((u) => {
      if (q) {
        const hay = [u.username, u.email, ...groupsOf(data, u).map((g) => g.name.toLowerCase()), u.is_admin ? 'admins' : '', ...loginsOf(u).map((i) => i.display.toLowerCase())];
        if (!hay.some((h) => h.includes(q))) return false;
      }
      // With a site chosen: only the people who can open it right now.
      if (filterSite && !opens(standing(data, u, filterSite))) return false;
      if (groupFilter === 'admins' && !u.is_admin) return false;
      if (groupFilter === 'none' && (u.is_admin || groupsOf(data, u).length > 0)) return false;
      if (groupFilter && groupFilter !== 'admins' && groupFilter !== 'none' && !groupsOf(data, u).some((g) => String(g.id) === groupFilter)) return false;
      if (loginFilter === 'password' && !u.has_password) return false;
      if ((loginFilter === 'discord' || loginFilter === 'google') && !loginsOf(u).some((i) => i.provider === loginFilter)) return false;
      switch (statusFilter) {
        case 'pending':
          return pendingUsers.has(u.id);
        case 'blocked':
          return u.status === 'blocked';
        case 'locked':
          return u.locked;
        case 'noemail':
          return !u.email;
        case 'unverified':
          return u.email && !u.email_verified;
        case 'nosites':
          return sitesOf(u).length === 0;
      }
      return true;
    });
    const by = {
      name: (a, b) => a.username.localeCompare(b.username),
      newest: (a, b) => b.created_at - a.created_at,
      oldest: (a, b) => a.created_at - b.created_at,
      login: (a, b) => b.last_login - a.last_login,
    }[sortBy] || ((a, b) => a.username.localeCompare(b.username));
    return list.sort(by);
  });
  const filtered = $derived(search || siteFilter || groupFilter || statusFilter || loginFilter);

  // Long lists render in pages, so the panel stays quick with many users.
  const pageSize = 50;
  let shown = $state(pageSize);
  $effect(() => {
    // A new filter starts at the top again.
    void [search, siteFilter, groupFilter, loginFilter, statusFilter, sortBy];
    shown = pageSize;
  });

  function clearFilters() {
    search = siteFilter = groupFilter = statusFilter = loginFilter = '';
  }

  // --- actions ---
  let busy = $state(false);

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

  let resetUser = $state(null);
  let resetOpen = $state(false);
  let createOpen = $state(false);

  const actions = {
    setAccess: (u, site, status) =>
      run(
        api('/admin/access', { user_id: u.id, site_id: site.id, status }, 'PUT'),
        status === 'approved' ? `${u.username} can now open ${site.name}.` : `${u.username} is denied for ${site.name}.`,
      ),
    async setStatus(u, status) {
      if (status === 'blocked' && !(await confirmDialog({ title: `Block ${u.username}?`, text: 'They are signed out everywhere and can’t sign in until you unblock them.', ok: 'Block', danger: true }))) return;
      run(api(`/admin/users/${u.id}/status`, { status }, 'PUT'), status === 'blocked' ? `${u.username} is blocked.` : `${u.username} is unblocked.`);
    },
    async setGroup(u, g, member) {
      if (!member && !(await confirmDialog({ title: `Remove ${u.username} from ${g.name}?`, text: 'They lose the access and rights this group gives.', ok: 'Remove', danger: true }))) return;
      run(api(`/admin/groups/${g.id}/members`, { user_id: u.id, member }, 'PUT'), member ? `${u.username} added to ${g.name}.` : `${u.username} removed from ${g.name}.`);
    },
    async setAdmin(u, admin) {
      const question = admin
        ? { title: `Make ${u.username} an admin?`, text: 'Admins can do everything, including removing you as admin.', ok: 'Make admin' }
        : { title: `Remove ${u.username} as admin?`, text: 'They keep their groups, but lose everything only admins can do.', ok: 'Remove admin', danger: true };
      if (!(await confirmDialog(question))) return;
      run(api(`/admin/users/${u.id}/admin`, { admin }, 'PUT'), admin ? `${u.username} is now an admin.` : `${u.username} is no longer an admin.`);
    },
    async remove(u) {
      if (!(await confirmDialog({ title: `Delete ${u.username}?`, text: 'Their account and all their access are removed. This cannot be undone.', ok: 'Delete', danger: true }))) return;
      run(api(`/admin/users/${u.id}`, undefined, 'DELETE'), `${u.username} was deleted.`);
    },
    openReset(u) {
      resetUser = u;
      resetOpen = true;
    },
  };
</script>

<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
  <h2 class="text-lg font-semibold tracking-tight">Users</h2>
  {#if can(data, 'users')}
    <button class={btnPrimary} onclick={() => (createOpen = true)}>
      <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg>
      Create user
    </button>
  {/if}
</div>

<UserFilters {data} bind:search bind:siteFilter bind:groupFilter bind:loginFilter bind:statusFilter bind:sortBy />

<p class="mb-3 flex flex-wrap items-center gap-x-3 text-sm text-zinc-400" aria-live="polite">
  {users.length} of {data.users.length} {data.users.length === 1 ? 'user' : 'users'}
  {#if filterSite}<span>· who can open <span class="font-medium text-zinc-200">{filterSite.name}</span></span>{/if}
  {#if filtered}<button class="font-medium text-indigo-400 hover:text-indigo-300" onclick={clearFilters}>Clear filters</button>{/if}
</p>

<ul class="space-y-2">
  {#each users.slice(0, shown) as u (u.id)}
    <UserRow
      {data}
      {u}
      {filterSite}
      {busy}
      {actions}
      {loginsOf}
      {sitesOf}
      open={expanded === u.id}
      pending={pendingUsers.has(u.id)}
      ontoggle={() => (expanded = expanded === u.id ? null : u.id)}
    />
  {:else}
    <li class="{card} p-8 text-center text-sm text-zinc-500">No users match these filters.</li>
  {/each}
</ul>
{#if users.length > shown}
  <div class="mt-4 flex justify-center">
    <button class={btnGhost} onclick={() => (shown += pageSize)}>Show {Math.min(pageSize, users.length - shown)} more ({users.length - shown} left)</button>
  </div>
{/if}

<ResetPasswordModal bind:open={resetOpen} user={resetUser} {reload} />
<CreateUserModal bind:open={createOpen} {data} {reload} />
