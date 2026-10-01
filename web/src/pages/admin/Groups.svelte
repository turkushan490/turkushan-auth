<script>
  import { api } from '../../lib/api.js';
  import { toast } from '../../lib/state.svelte.js';
  import Toggle from '../../components/Toggle.svelte';
  import { card, btnDanger, btnGhost, btnPrimary, badge, input, label } from './ui.js';
  import { can, permLabels } from './rights.js';

  let { data, reload } = $props();

  let expanded = $state(null); // group id or 'admins'
  let busy = $state(false);
  let creating = $state(false);
  let newName = $state('');
  let newDescription = $state('');
  let editing = $state(null);
  let editName = $state('');
  let editDescription = $state('');

  const manageGroups = $derived(can(data, 'groups'));
  const admins = $derived(data.users.filter((u) => u.is_admin));
  const membersOf = (g) => {
    const by = new Map(data.memberships.filter((m) => m.group_id === g.id).map((m) => [m.user_id, m.source]));
    return data.users.filter((u) => by.has(u.id)).map((u) => ({ ...u, source: by.get(u.id) }));
  };
  const siteRight = (g, site) => data.group_sites.find((gs) => gs.group_id === g.id && gs.site_id === site.id) || { can_open: false, can_approve: false };
  const rulesOf = (g) => data.rules.filter((r) => r.group_id === g.id);

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

  function startEdit(g) {
    editing = g.id;
    editName = g.name;
    editDescription = g.description;
  }

  async function saveEdit(e, g) {
    e.preventDefault();
    if (await run(api(`/admin/groups/${g.id}`, { name: editName, description: editDescription }, 'PUT'), 'Group saved.')) editing = null;
  }

  function remove(g) {
    const n = membersOf(g).length;
    if (!confirm(`Delete the group ${g.name}? ${n ? `Its ${n} member${n === 1 ? '' : 's'} lose${n === 1 ? 's' : ''} the access and rights it gives.` : ''}`)) return;
    run(api(`/admin/groups/${g.id}`, undefined, 'DELETE'), `Group ${g.name} deleted.`);
  }

  const setMember = (g, u, member) =>
    run(api(`/admin/groups/${g.id}/members`, { user_id: u.id, member }, 'PUT'), member ? `${u.username} added to ${g.name}.` : `${u.username} removed from ${g.name}.`);

  function setAdmin(u, admin) {
    if (admin && !confirm(`Make ${u.username} an admin? Admins can do everything, including removing you as admin.`)) return;
    run(api(`/admin/users/${u.id}/admin`, { admin }, 'PUT'), admin ? `${u.username} is now an admin.` : `${u.username} is no longer an admin.`);
  }

  function togglePerm(g, perm, on) {
    const perms = on ? [...g.perms, perm] : g.perms.filter((p) => p !== perm);
    run(api(`/admin/groups/${g.id}/perms`, { perms }, 'PUT'), `${g.name}: ${permLabels[perm].name} ${on ? 'on' : 'off'}.`);
  }

  function setSiteRight(g, site, field, on) {
    const cur = siteRight(g, site);
    run(api(`/admin/groups/${g.id}/sites`, { site_id: site.id, can_open: cur.can_open, can_approve: cur.can_approve, [field]: on }, 'PUT'), `${g.name} updated for ${site.name}.`);
  }

  // --- auto-add rules ---
  const discordHint =
    'Checked every time they sign in with Discord. To get an ID: Discord → Settings → Advanced → Developer Mode on, then right-click the server or role → Copy ID.';
  const ruleKinds = [
    { id: 'everyone', name: 'Everyone', hint: 'Every account, also new ones.' },
    { id: 'email', name: 'Email address', hint: 'One specific address, once it is verified.', placeholder: 'name@example.com' },
    { id: 'domain', name: 'Email domain', hint: 'Everyone with a verified address on this domain.', placeholder: 'example.com' },
    { id: 'method', name: 'Login method', hint: 'Everyone who has this way of signing in on their account.' },
    { id: 'discord_server', name: 'Discord server', hint: discordHint, placeholder: 'Server ID' },
    { id: 'discord_role', name: 'Discord role', hint: discordHint, placeholder: 'Server ID' },
  ];
  let ruleRole = $state('');
  let ruleMethod = $state('discord');
  let ruleKind = $state('everyone');
  let ruleValue = $state('');
  const kind = $derived(ruleKinds.find((k) => k.id === ruleKind));

  async function addRule(e, g) {
    e.preventDefault();
    const value = ruleKind === 'method' ? ruleMethod : ruleKind === 'discord_role' ? `${ruleValue.trim()}:${ruleRole.trim()}` : ruleValue;
    if (await run(api(`/admin/groups/${g.id}/rules`, { kind: ruleKind, value }), 'Rule added. Matching users are in the group now.')) ruleValue = ruleRole = '';
  }

  const removeRule = (r) => run(api(`/admin/rules/${r.id}`, undefined, 'DELETE'), 'Rule removed.');

  const ruleText = (r) =>
    ({
      everyone: 'Everyone',
      email: `Email is ${r.value}`,
      domain: `Email ends with @${r.value}`,
      method: `Can sign in with ${{ discord: 'Discord', google: 'Google', password: 'a password' }[r.value] || r.value}`,
      discord_server: `In Discord server ${r.value}`,
      discord_role: `Has Discord role ${r.value.split(':')[1]} in server ${r.value.split(':')[0]}`,
    })[r.kind] || r.kind;

  const summary = (g) => {
    const parts = g.perms.map((p) => permLabels[p]?.name || p);
    const opens = data.group_sites.filter((gs) => gs.group_id === g.id && gs.can_open).length;
    const approves = data.group_sites.filter((gs) => gs.group_id === g.id && gs.can_approve).length;
    if (opens) parts.unshift(`opens ${opens} site${opens === 1 ? '' : 's'}`);
    if (approves) parts.push(`approves for ${approves} site${approves === 1 ? '' : 's'}`);
    return parts;
  };

  const select = 'rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 outline-none focus:border-indigo-500';
  const sectionTitle = 'mb-2 text-xs font-semibold uppercase tracking-wider text-zinc-500';
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
  <!-- Built-in Admins group -->
  <li class="{card} overflow-hidden">
    <button type="button" class="flex w-full items-center gap-3 p-4 text-left transition hover:bg-zinc-800/30" onclick={() => (expanded = expanded === 'admins' ? null : 'admins')} aria-expanded={expanded === 'admins'}>
      <div class="grid size-10 shrink-0 place-items-center rounded-full bg-indigo-500/20 text-indigo-300">
        <svg viewBox="0 0 24 24" class="size-5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3l8 3v6c0 4.5-3.2 8.3-8 9-4.8-.7-8-4.5-8-9V6z" /></svg>
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="font-medium">Admins</span>
          <span class="{badge} bg-zinc-800 text-zinc-400 ring-zinc-700">built in</span>
        </div>
        <p class="mt-0.5 text-xs text-zinc-500">{admins.length} member{admins.length === 1 ? '' : 's'} · all rights, every site</p>
      </div>
      <svg viewBox="0 0 24 24" class="size-4 shrink-0 text-zinc-500 transition {expanded === 'admins' ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
    </button>
    {#if expanded === 'admins'}
      <div class="space-y-3 border-t border-zinc-800 bg-zinc-950/40 p-4">
        <h4 class={sectionTitle}>Members</h4>
        <div class="flex flex-wrap items-center gap-1.5">
          {#each admins as u (u.id)}
            <span class="{badge} gap-1 bg-indigo-500/10 text-indigo-300 ring-indigo-400/20">
              {u.username}{#if u.id === data.me.id}<span class="text-indigo-400/60">· you</span>{/if}
              {#if data.me.admin && u.id !== data.me.id}
                <button type="button" class="-mr-1 rounded-full px-1 text-indigo-300/70 hover:text-white" aria-label="Remove {u.username} as admin" disabled={busy} onclick={() => setAdmin(u, false)}>×</button>
              {/if}
            </span>
          {/each}
          {#if data.me.admin}
            <select class="rounded-full border border-dashed border-zinc-700 bg-zinc-900 px-2 py-0.5 text-xs text-zinc-400 outline-none hover:border-zinc-500" aria-label="Make someone admin" disabled={busy}
              onchange={(e) => {
                const u = data.users.find((u) => String(u.id) === e.currentTarget.value);
                e.currentTarget.value = '';
                if (u) setAdmin(u, true);
              }}>
              <option value="">+ Add admin</option>
              {#each data.users.filter((u) => !u.is_admin && u.status === 'active') as u (u.id)}<option value={String(u.id)}>{u.username}</option>{/each}
            </select>
          {/if}
        </div>
        <p class="text-xs text-zinc-500">Admins can do everything in this panel and open every site. You can't remove yourself; another admin has to do that.</p>
      </div>
    {/if}
  </li>

  {#each data.groups as g (g.id)}
    {@const open = expanded === g.id}
    {@const members = membersOf(g)}
    <li class="{card} overflow-hidden">
      <button type="button" class="flex w-full items-center gap-3 p-4 text-left transition hover:bg-zinc-800/30" onclick={() => (expanded = open ? null : g.id)} aria-expanded={open}>
        <div class="grid size-10 shrink-0 place-items-center rounded-full bg-zinc-800 text-sm font-semibold uppercase text-zinc-300">{g.name.slice(0, 1)}</div>
        <div class="min-w-0 flex-1">
          <span class="font-medium">{g.name}</span>
          <p class="mt-0.5 truncate text-xs text-zinc-500">
            {members.length} member{members.length === 1 ? '' : 's'}{g.description ? ` · ${g.description}` : ''}
          </p>
          <div class="mt-1.5 flex flex-wrap gap-1">
            {#each summary(g) as part}
              <span class="{badge} bg-zinc-800 text-zinc-300 ring-zinc-700">{part}</span>
            {:else}
              <span class="text-xs text-zinc-600">no rights yet</span>
            {/each}
            {#if rulesOf(g).length}<span class="{badge} bg-sky-500/10 text-sky-300 ring-sky-500/25">{rulesOf(g).length} auto-add rule{rulesOf(g).length === 1 ? '' : 's'}</span>{/if}
          </div>
        </div>
        <svg viewBox="0 0 24 24" class="size-4 shrink-0 text-zinc-500 transition {open ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
      </button>

      {#if open}
        <div class="space-y-6 border-t border-zinc-800 bg-zinc-950/40 p-4">
          <section>
            <h4 class={sectionTitle}>Members</h4>
            <div class="flex flex-wrap items-center gap-1.5">
              {#each members as u (u.id)}
                <span class="{badge} gap-1 bg-indigo-500/10 text-indigo-300 ring-indigo-400/20">
                  {u.username}{#if u.source === 'rule'}<span class="text-indigo-400/60">· by rule</span>{/if}
                  {#if manageGroups}
                    <button type="button" class="-mr-1 rounded-full px-1 text-indigo-300/70 hover:text-white" aria-label="Remove {u.username}" disabled={busy} onclick={() => setMember(g, u, false)}>×</button>
                  {/if}
                </span>
              {:else}
                <span class="text-sm text-zinc-500">Nobody yet.</span>
              {/each}
              {#if manageGroups && data.users.some((u) => !members.some((m) => m.id === u.id))}
                <select class="rounded-full border border-dashed border-zinc-700 bg-zinc-900 px-2 py-0.5 text-xs text-zinc-400 outline-none hover:border-zinc-500" aria-label="Add a member to {g.name}" disabled={busy}
                  onchange={(e) => {
                    const u = data.users.find((u) => String(u.id) === e.currentTarget.value);
                    e.currentTarget.value = '';
                    if (u) setMember(g, u, true);
                  }}>
                  <option value="">+ Add member</option>
                  {#each data.users.filter((u) => !members.some((m) => m.id === u.id)) as u (u.id)}<option value={String(u.id)}>{u.username}</option>{/each}
                </select>
              {/if}
            </div>
          </section>

          <section>
            <h4 class={sectionTitle}>Sites</h4>
            {#if data.sites.length === 0}
              <p class="text-sm text-zinc-500">No sites yet. Add them in the Sites tab.</p>
            {:else}
              <div class="overflow-hidden rounded-lg border border-zinc-800">
                <div class="grid grid-cols-[1fr_5rem_5rem] items-center gap-2 border-b border-zinc-800 bg-zinc-900/60 px-3 py-2 text-xs font-medium text-zinc-400">
                  <span>Site</span><span class="text-center">Can open</span><span class="text-center">Can approve</span>
                </div>
                {#each data.sites as site (site.id)}
                  {@const sr = siteRight(g, site)}
                  <div class="grid grid-cols-[1fr_5rem_5rem] items-center gap-2 border-b border-zinc-800/60 px-3 py-2 text-sm last:border-0">
                    <div class="min-w-0">
                      <span class="font-medium">{site.name}</span>
                      <span class="block truncate text-xs text-zinc-500">{site.host}</span>
                    </div>
                    <div class="grid place-items-center">
                      <input type="checkbox" class="size-4 accent-indigo-500" checked={sr.can_open} disabled={busy || !data.me.admin} aria-label="{g.name} can open {site.name}"
                        onchange={(e) => setSiteRight(g, site, 'can_open', e.currentTarget.checked)} />
                    </div>
                    <div class="grid place-items-center">
                      <input type="checkbox" class="size-4 accent-indigo-500" checked={sr.can_approve} disabled={busy || !data.me.admin} aria-label="{g.name} can approve requests for {site.name}"
                        onchange={(e) => setSiteRight(g, site, 'can_approve', e.currentTarget.checked)} />
                    </div>
                  </div>
                {/each}
              </div>
              <p class="mt-2 text-xs text-zinc-500">
                <span class="font-medium text-zinc-400">Can open:</span> members get in without asking.
                <span class="font-medium text-zinc-400">Can approve:</span> members may approve or deny requests for that site.
              </p>
            {/if}
          </section>

          <section>
            <h4 class={sectionTitle}>Rights in the admin panel</h4>
            <div class="grid gap-x-6 gap-y-4 rounded-lg border border-zinc-800 p-4 sm:grid-cols-2">
              {#each data.all_perms as perm}
                <Toggle checked={g.perms.includes(perm)} label={permLabels[perm].name} description={permLabels[perm].hint} disabled={busy || !data.me.admin}
                  onchange={(v) => togglePerm(g, perm, v)} />
              {/each}
            </div>
            {#if !data.me.admin}<p class="mt-2 text-xs text-zinc-500">Only admins can change what a group may do.</p>{/if}
          </section>

          <section>
            <h4 class={sectionTitle}>Auto-add rules</h4>
            {#if rulesOf(g).length}
              <ul class="mb-3 divide-y divide-zinc-800/80 rounded-lg border border-zinc-800">
                {#each rulesOf(g) as r (r.id)}
                  <li class="flex items-center justify-between gap-3 px-3 py-2 text-sm">
                    <span>{ruleText(r)}</span>
                    {#if manageGroups}<button class={btnGhost} disabled={busy} onclick={() => removeRule(r)}>Remove</button>{/if}
                  </li>
                {/each}
              </ul>
            {:else}
              <p class="mb-3 text-sm text-zinc-500">No rules. Members are added by hand.</p>
            {/if}
            {#if manageGroups}
              <form class="flex flex-col gap-2 sm:flex-row" onsubmit={(e) => addRule(e, g)}>
                <select bind:value={ruleKind} class={select} aria-label="Kind of rule">
                  {#each ruleKinds as k}<option value={k.id}>{k.name}</option>{/each}
                </select>
                {#if ruleKind === 'method'}
                  <select bind:value={ruleMethod} class={select} aria-label="Login method">
                    <option value="discord">Discord</option>
                    <option value="google">Google</option>
                    <option value="password">Username and password</option>
                  </select>
                {:else if kind.placeholder}
                  <input class={input} bind:value={ruleValue} placeholder={kind.placeholder} autocapitalize="off" spellcheck="false" required aria-label={kind.placeholder} />
                {/if}
                {#if ruleKind === 'discord_role'}
                  <input class={input} bind:value={ruleRole} placeholder="Role ID" autocapitalize="off" spellcheck="false" required aria-label="Role ID" />
                {/if}
                <button type="submit" class="{btnGhost} shrink-0" disabled={busy}>Add rule</button>
              </form>
              <p class="mt-2 text-xs text-zinc-500">{kind.hint} Rules add people automatically and take them out again when they no longer match. People you add by hand always stay.</p>
            {/if}
          </section>

          {#if manageGroups}
            <section class="border-t border-zinc-800 pt-4">
              {#if editing === g.id}
                <form class="space-y-3" onsubmit={(e) => saveEdit(e, g)}>
                  <div class="grid gap-3 sm:grid-cols-2">
                    <input class={input} bind:value={editName} maxlength="40" aria-label="Group name" required />
                    <input class={input} bind:value={editDescription} maxlength="200" placeholder="Description" aria-label="Description" />
                  </div>
                  <div class="flex justify-end gap-2">
                    <button type="button" class={btnGhost} onclick={() => (editing = null)}>Cancel</button>
                    <button type="submit" class={btnPrimary} disabled={busy}>Save</button>
                  </div>
                </form>
              {:else}
                <div class="flex justify-end gap-2">
                  <button class={btnGhost} disabled={busy} onclick={() => startEdit(g)}>Rename</button>
                  <button class={btnDanger} disabled={busy} onclick={() => remove(g)}>Delete group</button>
                </div>
              {/if}
            </section>
          {/if}
        </div>
      {/if}
    </li>
  {:else}
    <li class="{card} p-8 text-center text-sm text-zinc-500">
      No groups yet. Make one, like "Friends", tick the sites it may open, and add people.
    </li>
  {/each}
</ul>
