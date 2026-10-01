<script>
  import { api } from '../../lib/api.js';
  import { toast } from '../../lib/state.svelte.js';
  import { timeAgo, fullDate } from '../../lib/format.js';
  import Modal from '../../components/Modal.svelte';
  import Field from '../../components/Field.svelte';
  import PasswordRules from '../../components/PasswordRules.svelte';
  import ProviderIcon from '../../components/ProviderIcon.svelte';
  import { card, btnApprove, btnDanger, btnGhost, btnPrimary, badge, statusBadge, input, label } from './ui.js';
  import { can, canApprove, standing, opens, groupsOf, groupsOpening, standingText, standingBadge } from './rights.js';

  let { data, reload } = $props();

  // --- search, filters, sorting ---
  let search = $state('');
  let siteFilter = $state('');
  let groupFilter = $state(''); // group id, 'admins' or 'none'
  let statusFilter = $state('');
  let loginFilter = $state(''); // '' | password | discord | google
  let sortBy = $state('name');
  let expanded = $state(null);
  let busy = $state(false);

  const filterSite = $derived(data.sites.find((s) => String(s.id) === siteFilter));
  const pendingUsers = $derived(new Set(data.access.filter((a) => a.status === 'pending').map((a) => a.user_id)));
  // Discord/Google accounts connected to a user.
  const loginsOf = (u) => data.identities.filter((i) => i.user_id === u.id);
  const providerNames = { discord: 'Discord', google: 'Google' };
  const sitesOf = (u) => data.sites.filter((s) => opens(standing(data, u, s)));

  const statusFilters = [
    { id: '', name: 'Any status' },
    { id: 'pending', name: 'Has open request' },
    { id: 'blocked', name: 'Blocked' },
    { id: 'locked', name: 'Locked' },
    { id: 'noemail', name: 'No email' },
    { id: 'unverified', name: 'Email not verified' },
    { id: 'nosites', name: "Can't open any site" },
  ];
  const sorts = [
    { id: 'name', name: 'Name (A–Z)' },
    { id: 'newest', name: 'Newest first' },
    { id: 'oldest', name: 'Oldest first' },
    { id: 'login', name: 'Last login' },
  ];

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
    }[sortBy];
    return list.sort(by);
  });
  const filtered = $derived(search || siteFilter || groupFilter || statusFilter || loginFilter);

  function clearFilters() {
    search = siteFilter = groupFilter = statusFilter = loginFilter = '';
  }

  // --- actions ---
  const isMe = (u) => u.id === data.me.id;
  const canManage = (u) => can(data, 'users') && !isMe(u) && (!u.is_admin || data.me.admin);

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
      status === 'approved' ? `${u.username} can now open ${site.name}.` : `${u.username} is denied for ${site.name}.`,
    );
  const setStatus = (u, status) =>
    run(api(`/admin/users/${u.id}/status`, { status }, 'PUT'), status === 'blocked' ? `${u.username} is blocked.` : `${u.username} is unblocked.`);
  const setGroup = (u, g, member) =>
    run(api(`/admin/groups/${g.id}/members`, { user_id: u.id, member }, 'PUT'), member ? `${u.username} added to ${g.name}.` : `${u.username} removed from ${g.name}.`);

  function setAdmin(u, admin) {
    if (admin && !confirm(`Make ${u.username} an admin? Admins can do everything, including removing you as admin.`)) return;
    run(api(`/admin/users/${u.id}/admin`, { admin }, 'PUT'), admin ? `${u.username} is now an admin.` : `${u.username} is no longer an admin.`);
  }

  function remove(u) {
    if (!confirm(`Delete ${u.username}? Their account and all their access are removed. This cannot be undone.`)) return;
    run(api(`/admin/users/${u.id}`, undefined, 'DELETE'), `${u.username} was deleted.`);
  }

  // --- reset password dialog ---
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

  // --- create user dialog ---
  let createOpen = $state(false);
  let createError = $state('');
  let draft = $state({ username: '', email: '', password: '', invite: false, group_ids: [], admin: false });

  function openCreate() {
    draft = { username: '', email: '', password: '', invite: false, group_ids: [], admin: false };
    createError = '';
    createOpen = true;
  }

  function toggleDraftGroup(id) {
    draft.group_ids = draft.group_ids.includes(id) ? draft.group_ids.filter((g) => g !== id) : [...draft.group_ids, id];
  }

  async function submitCreate(e) {
    e.preventDefault();
    createError = '';
    busy = true;
    const r = await api('/admin/users', { ...draft, password: draft.invite ? '' : draft.password });
    busy = false;
    if (!r.ok) {
      createError = r.data.error;
      return;
    }
    createOpen = false;
    if (r.data.warning) toast(r.data.warning, 'error');
    else toast(draft.invite ? `Invite sent to ${draft.email}.` : `${draft.username} created. Tell them their password.`);
    reload();
  }

  const select = 'rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-sm text-zinc-200 outline-none focus:border-indigo-500';
  const chip = `${badge} bg-zinc-800 text-zinc-300 ring-zinc-700`;
</script>

<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
  <h2 class="text-lg font-semibold tracking-tight">Users</h2>
  {#if can(data, 'users')}
    <button class={btnPrimary} onclick={openCreate}>
      <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg>
      Create user
    </button>
  {/if}
</div>

<div class="{card} mb-4 grid gap-2 p-3 sm:grid-cols-2 lg:grid-cols-6">
  <input
    type="search"
    placeholder="Search name, email or group…"
    bind:value={search}
    class="rounded-lg border border-zinc-800 bg-zinc-950/60 px-3 py-1.5 text-sm outline-none placeholder:text-zinc-600 focus:border-indigo-500 sm:col-span-2 lg:col-span-1"
  />
  <select bind:value={siteFilter} class={select} aria-label="Filter by site">
    <option value="">All sites</option>
    {#each data.sites as s (s.id)}<option value={String(s.id)}>{s.name}</option>{/each}
  </select>
  <select bind:value={groupFilter} class={select} aria-label="Filter by group">
    <option value="">All groups</option>
    <option value="admins">Admins</option>
    {#each data.groups as g (g.id)}<option value={String(g.id)}>{g.name}</option>{/each}
    <option value="none">Not in any group</option>
  </select>
  <select bind:value={loginFilter} class={select} aria-label="Filter by login type">
    <option value="">Any login</option>
    <option value="password">Username &amp; password</option>
    <option value="discord">Discord</option>
    <option value="google">Google</option>
  </select>
  <select bind:value={statusFilter} class={select} aria-label="Filter by status">
    {#each statusFilters as f}<option value={f.id}>{f.name}</option>{/each}
  </select>
  <select bind:value={sortBy} class={select} aria-label="Sort">
    {#each sorts as s}<option value={s.id}>Sort: {s.name}</option>{/each}
  </select>
</div>

<p class="mb-3 flex flex-wrap items-center gap-x-3 text-sm text-zinc-400">
  {users.length} of {data.users.length} {data.users.length === 1 ? 'user' : 'users'}
  {#if filterSite}<span>· who can open <span class="font-medium text-zinc-200">{filterSite.name}</span></span>{/if}
  {#if filtered}<button class="font-medium text-indigo-400 hover:text-indigo-300" onclick={clearFilters}>Clear filters</button>{/if}
</p>

<ul class="space-y-2">
  {#each users as u (u.id)}
    {@const open = expanded === u.id}
    {@const myGroups = groupsOf(data, u)}
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
            {#each loginsOf(u) as i (i.provider)}
              <span class="inline-flex" title="{providerNames[i.provider]}{i.display ? `: ${i.display}` : ''}">
                <ProviderIcon provider={i.provider} size="size-4" color />
                <span class="sr-only">Connected to {providerNames[i.provider]}</span>
              </span>
            {/each}
            {#if isMe(u)}<span class="text-xs text-zinc-500">(you)</span>{/if}
            {#if u.is_admin}<span class="{badge} bg-indigo-500/15 text-indigo-300 ring-indigo-400/20">Admin</span>{/if}
            {#if u.status === 'blocked'}<span class="{badge} {statusBadge.denied}">Blocked</span>{/if}
            {#if u.locked}<span class="{badge} {statusBadge.pending}">Locked</span>{/if}
            {#if pendingUsers.has(u.id)}<span class="{badge} {statusBadge.pending}">Request open</span>{/if}
          </div>
          <p class="mt-0.5 truncate text-xs text-zinc-500">
            {u.email ? (u.email_verified ? u.email : `${u.email} (not verified)`) : 'no email'} · last login {timeAgo(u.last_login)}
          </p>
          <div class="mt-1.5 flex flex-wrap items-center gap-1">
            {#if filterSite}
              {@const st = standing(data, u, filterSite)}
              <span class="{badge} {standingBadge[st]}">{standingText[st]}</span>
            {/if}
            {#each myGroups as g (g.id)}
              <span class="{badge} bg-indigo-500/10 text-indigo-300 ring-indigo-400/20">{g.name}</span>
            {/each}
            {#if !filterSite}
              {#if u.is_admin}
                <span class={chip}>all sites</span>
              {:else}
                {#each sitesOf(u) as s (s.id)}
                  <span class={chip}>{s.name}</span>
                {:else}
                  {#if data.sites.length > 0}<span class="text-xs text-zinc-600">no sites yet</span>{/if}
                {/each}
              {/if}
            {/if}
          </div>
        </div>
        <svg viewBox="0 0 24 24" class="size-4 shrink-0 text-zinc-500 transition {open ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
      </button>

      {#if open}
        <div class="space-y-5 border-t border-zinc-800 bg-zinc-950/40 p-4">
          <div>
            <h3 class="mb-2 text-xs font-semibold uppercase tracking-wider text-zinc-500">Groups</h3>
            <div class="flex flex-wrap items-center gap-1.5">
              {#if u.is_admin}<span class="{badge} bg-indigo-500/15 text-indigo-300 ring-indigo-400/20">Admins</span>{/if}
              {#each myGroups as g (g.id)}
                {@const byRule = data.memberships.find((m) => m.user_id === u.id && m.group_id === g.id)?.source === 'rule'}
                <span class="{badge} gap-1 bg-indigo-500/10 text-indigo-300 ring-indigo-400/20">
                  {g.name}{#if byRule}<span class="text-indigo-400/60">· by rule</span>{/if}
                  {#if can(data, 'groups')}
                    <button type="button" class="-mr-1 rounded-full px-1 text-indigo-300/70 hover:text-white" aria-label="Remove from {g.name}" disabled={busy} onclick={() => setGroup(u, g, false)}>×</button>
                  {/if}
                </span>
              {:else}
                {#if !u.is_admin}<span class="text-sm text-zinc-500">Not in a group.</span>{/if}
              {/each}
              {#if can(data, 'groups') && data.groups.some((g) => !myGroups.includes(g))}
                <select
                  class="rounded-full border border-dashed border-zinc-700 bg-zinc-900 px-2 py-0.5 text-xs text-zinc-400 outline-none hover:border-zinc-500"
                  aria-label="Add {u.username} to a group"
                  disabled={busy}
                  onchange={(e) => {
                    const g = data.groups.find((g) => String(g.id) === e.currentTarget.value);
                    e.currentTarget.value = '';
                    if (g) setGroup(u, g, true);
                  }}
                >
                  <option value="">+ Add to group</option>
                  {#each data.groups.filter((g) => !myGroups.includes(g)) as g (g.id)}<option value={String(g.id)}>{g.name}</option>{/each}
                </select>
              {/if}
            </div>
          </div>

          <div>
            <h3 class="mb-2 text-xs font-semibold uppercase tracking-wider text-zinc-500">Site access</h3>
            {#if u.is_admin}
              <p class="text-sm text-zinc-400">Admins can open every site.</p>
            {:else if data.sites.length === 0}
              <p class="text-sm text-zinc-400">No sites yet.</p>
            {:else}
              <div class="divide-y divide-zinc-800/80 rounded-lg border border-zinc-800">
                {#each data.sites as site (site.id)}
                  {@const st = standing(data, u, site)}
                  {@const direct = data.access.find((a) => a.user_id === u.id && a.site_id === site.id)?.status}
                  <div class="flex flex-wrap items-center gap-x-3 gap-y-2 px-3 py-2.5">
                    <div class="min-w-0 flex-1 text-sm">
                      <span class="font-medium">{site.name}</span>
                      <span class="text-zinc-500">{site.host}</span>
                    </div>
                    {#if st}
                      <span class="{badge} {standingBadge[st]}">
                        {st === 'group' ? `via ${groupsOpening(data, u, site).map((g) => g.name).join(', ')}` : st === 'open' ? 'open for everyone' : standingText[st]}
                      </span>
                    {:else}
                      <span class="text-xs text-zinc-500">no access</span>
                    {/if}
                    {#if site.require_approval && canApprove(data, site.id)}
                      {#if direct === 'approved'}
                        <button class={btnGhost} disabled={busy} onclick={() => setAccess(u, site, 'denied')}>Revoke</button>
                      {:else if direct === 'denied'}
                        <button class={btnApprove} disabled={busy} onclick={() => setAccess(u, site, 'approved')}>Give access</button>
                      {:else}
                        <button class={btnDanger} disabled={busy} onclick={() => setAccess(u, site, 'denied')}>Deny</button>
                        {#if st !== 'group'}
                          <button class={btnApprove} disabled={busy} onclick={() => setAccess(u, site, 'approved')}>Give access</button>
                        {/if}
                      {/if}
                    {/if}
                  </div>
                {/each}
              </div>
              <p class="mt-2 text-xs text-zinc-500">Deny always wins, also over a group.</p>
            {/if}
          </div>

          <div>
            <h3 class="mb-2 text-xs font-semibold uppercase tracking-wider text-zinc-500">Signs in with</h3>
            <div class="flex flex-wrap items-center gap-1.5">
              {#if u.has_password}<span class={chip}>Username &amp; password</span>{/if}
              {#each loginsOf(u) as i (i.provider)}
                <span class="{chip} gap-1.5">
                  <ProviderIcon provider={i.provider} size="size-3.5" color />
                  {providerNames[i.provider]}{i.display ? `: ${i.display}` : ''}
                </span>
              {/each}
              {#if !u.has_password && loginsOf(u).length === 0}
                <span class="text-sm text-zinc-500">No password yet (invited, waiting for them to choose one).</span>
              {/if}
            </div>
          </div>

          <div class="flex flex-wrap items-center justify-between gap-3">
            <p class="text-xs text-zinc-500" title={fullDate(u.created_at)}>Joined {timeAgo(u.created_at)}</p>
            <div class="flex flex-wrap gap-2">
              {#if data.me.admin && !isMe(u)}
                {#if u.is_admin}
                  <button class={btnGhost} disabled={busy} onclick={() => setAdmin(u, false)}>Remove admin</button>
                {:else if u.status === 'active'}
                  <button class={btnGhost} disabled={busy} onclick={() => setAdmin(u, true)}>Make admin</button>
                {/if}
              {/if}
              {#if canManage(u)}
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
              {:else if isMe(u)}
                <span class="text-xs text-zinc-500">Change your own password and email on your account page.</span>
              {/if}
            </div>
          </div>
        </div>
      {/if}
    </li>
  {:else}
    <li class="{card} p-8 text-center text-sm text-zinc-500">No users match these filters.</li>
  {/each}
</ul>

<Modal bind:open={resetOpen} title="Reset password for {resetUser?.username}">
  <form class="space-y-4" onsubmit={submitReset}>
    <p class="text-sm text-zinc-400">
      Set a new password and send it to them yourself. They're signed out everywhere and can log in with the new password.
    </p>
    <Field label="New password" type="password" autocomplete="new-password" required bind:value={newPassword}>
      <PasswordRules password={newPassword} />
    </Field>
    {#if resetError}<p class="text-sm text-rose-400">{resetError}</p>{/if}
    <div class="flex justify-end gap-2">
      <button type="button" class={btnGhost} onclick={() => (resetOpen = false)}>Cancel</button>
      <button type="submit" class={btnPrimary} disabled={busy}>Set password</button>
    </div>
  </form>
</Modal>

<Modal bind:open={createOpen} title="Create user">
  <form class="space-y-4" onsubmit={submitCreate}>
    <div>
      <label class={label} for="new-username">Username</label>
      <input id="new-username" class={input} bind:value={draft.username} autocapitalize="off" spellcheck="false" autocomplete="off" required />
    </div>
    <div>
      <label class="{label} flex items-baseline justify-between" for="new-email">
        Email {#if !draft.invite}<span class="text-xs font-normal text-zinc-500">optional</span>{/if}
      </label>
      <input id="new-email" type="email" class={input} bind:value={draft.email} autocomplete="off" required={draft.invite} />
    </div>

    <div class="grid grid-cols-2 gap-1 rounded-lg border border-zinc-800 bg-zinc-950/60 p-1" role="radiogroup" aria-label="How they get their password">
      <button type="button" role="radio" aria-checked={!draft.invite} onclick={() => (draft.invite = false)}
        class="rounded-md px-3 py-1.5 text-sm font-medium transition {!draft.invite ? 'bg-indigo-500 text-white' : 'text-zinc-400 hover:text-zinc-200'}">
        I set the password
      </button>
      <button type="button" role="radio" aria-checked={draft.invite} onclick={() => (draft.invite = true)} disabled={!data.mail_ready}
        title={data.mail_ready ? '' : 'Set up email first (Settings → Email)'}
        class="rounded-md px-3 py-1.5 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-40 {draft.invite ? 'bg-indigo-500 text-white' : 'text-zinc-400 hover:text-zinc-200'}">
        Send invite mail
      </button>
    </div>
    {#if draft.invite}
      <p class="text-sm text-zinc-400">They get a mail with a link to choose their own password. The link works for 3 days.</p>
    {:else}
      <Field label="Password" type="password" autocomplete="new-password" required bind:value={draft.password} hint="Tell them this password yourself. They can change it on their account page.">
        <PasswordRules password={draft.password} />
      </Field>
    {/if}

    {#if (can(data, 'groups') && data.groups.length > 0) || data.me.admin}
      <div>
        <span class={label}>Groups</span>
        <div class="flex flex-wrap gap-1.5">
          {#if can(data, 'groups')}
            {#each data.groups as g (g.id)}
              <button type="button" aria-pressed={draft.group_ids.includes(g.id)} onclick={() => toggleDraftGroup(g.id)}
                class="rounded-full px-2.5 py-1 text-xs font-medium ring-1 transition {draft.group_ids.includes(g.id) ? 'bg-indigo-500 text-white ring-indigo-400' : 'bg-zinc-800 text-zinc-300 ring-zinc-700 hover:bg-zinc-700'}">
                {g.name}
              </button>
            {/each}
          {/if}
          {#if data.me.admin}
            <button type="button" aria-pressed={draft.admin} onclick={() => (draft.admin = !draft.admin)}
              class="rounded-full px-2.5 py-1 text-xs font-medium ring-1 transition {draft.admin ? 'bg-indigo-500 text-white ring-indigo-400' : 'bg-zinc-800 text-zinc-300 ring-zinc-700 hover:bg-zinc-700'}">
              Admins
            </button>
          {/if}
        </div>
        {#if draft.admin}<p class="mt-1.5 text-xs text-amber-300/90">Admins can do everything, including managing other admins.</p>{/if}
      </div>
    {/if}

    {#if createError}<p class="text-sm text-rose-400">{createError}</p>{/if}
    <div class="flex justify-end gap-2">
      <button type="button" class={btnGhost} onclick={() => (createOpen = false)}>Cancel</button>
      <button type="submit" class={btnPrimary} disabled={busy}>{draft.invite ? 'Create and send invite' : 'Create user'}</button>
    </div>
  </form>
</Modal>
