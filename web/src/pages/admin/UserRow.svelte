<script>
  import { timeAgo, fullDate } from '../../lib/format.js';
  import ProviderIcon from '../../components/ProviderIcon.svelte';
  import { card, btnApprove, btnDanger, btnGhost, badge, statusBadge } from './ui.js';
  import { can, canApprove, standing, groupsOf, groupsOpening, standingText, standingBadge } from './rights.js';

  // actions: { setAccess, setStatus, setGroup, setAdmin, remove, openReset } from Users.svelte.
  let { data, u, open, ontoggle, filterSite, pending, busy, actions, loginsOf, sitesOf } = $props();

  const providerNames = { discord: 'Discord', google: 'Google' };
  const isMe = $derived(u.id === data.me.id);
  const canManage = $derived(can(data, 'users') && !isMe && (!u.is_admin || data.me.admin));
  const myGroups = $derived(groupsOf(data, u));
  const chip = `${badge} bg-zinc-800 text-zinc-300 ring-zinc-700`;
</script>

<li class="{card} overflow-hidden">
  <button
    type="button"
    class="flex w-full items-center gap-3 p-4 text-left transition hover:bg-zinc-800/30 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-indigo-500"
    onclick={ontoggle}
    aria-expanded={open}
  >
    <div class="grid size-10 shrink-0 place-items-center rounded-full text-sm font-semibold uppercase {u.is_admin ? 'bg-indigo-500/20 text-indigo-300' : 'bg-zinc-800 text-zinc-300'}" aria-hidden="true">
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
        {#if isMe}<span class="text-xs text-zinc-500">(you)</span>{/if}
        {#if u.is_admin}<span class="{badge} bg-indigo-500/15 text-indigo-300 ring-indigo-400/20">Admin</span>{/if}
        {#if u.status === 'blocked'}<span class="{badge} {statusBadge.denied}">Blocked</span>{/if}
        {#if u.locked}<span class="{badge} {statusBadge.pending}">Locked</span>{/if}
        {#if pending}<span class="{badge} {statusBadge.pending}">Request open</span>{/if}
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
              {#if data.sites.length > 0}<span class="text-xs text-zinc-500">no sites yet</span>{/if}
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
                <button type="button" class="-mr-1 rounded-full px-1 text-indigo-300/70 hover:text-white" aria-label="Remove from {g.name}" disabled={busy} onclick={() => actions.setGroup(u, g, false)}>×</button>
              {/if}
            </span>
          {:else}
            {#if !u.is_admin}<span class="text-sm text-zinc-500">Not in a group.</span>{/if}
          {/each}
          {#if can(data, 'groups') && data.groups.some((g) => !myGroups.includes(g))}
            <select
              class="rounded-full border border-dashed border-zinc-700 bg-zinc-900 px-2 py-0.5 text-xs text-zinc-400 outline-none hover:border-zinc-500 focus:border-indigo-500"
              aria-label="Add {u.username} to a group"
              disabled={busy}
              onchange={(e) => {
                const g = data.groups.find((g) => String(g.id) === e.currentTarget.value);
                e.currentTarget.value = '';
                if (g) actions.setGroup(u, g, true);
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
                    <button class={btnGhost} disabled={busy} onclick={() => actions.setAccess(u, site, 'denied')}>Revoke</button>
                  {:else if direct === 'denied'}
                    <button class={btnApprove} disabled={busy} onclick={() => actions.setAccess(u, site, 'approved')}>Give access</button>
                  {:else}
                    <button class={btnDanger} disabled={busy} onclick={() => actions.setAccess(u, site, 'denied')}>Deny</button>
                    {#if st !== 'group'}
                      <button class={btnApprove} disabled={busy} onclick={() => actions.setAccess(u, site, 'approved')}>Give access</button>
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
          {#if data.me.admin && !isMe}
            {#if u.is_admin}
              <button class={btnGhost} disabled={busy} onclick={() => actions.setAdmin(u, false)}>Remove admin</button>
            {:else if u.status === 'active'}
              <button class={btnGhost} disabled={busy} onclick={() => actions.setAdmin(u, true)}>Make admin</button>
            {/if}
          {/if}
          {#if canManage}
            <button class={btnGhost} disabled={busy} onclick={() => actions.openReset(u)}>Reset password</button>
            {#if u.status === 'blocked'}
              <button class={btnApprove} disabled={busy} onclick={() => actions.setStatus(u, 'active')}>Unblock</button>
            {:else}
              <button class={btnGhost} disabled={busy} onclick={() => actions.setStatus(u, 'blocked')}>Block</button>
            {/if}
            {#if u.locked && u.status !== 'blocked'}
              <button class={btnGhost} disabled={busy} onclick={() => actions.setStatus(u, 'active')}>Unlock</button>
            {/if}
            <button class={btnDanger} disabled={busy} onclick={() => actions.remove(u)}>Delete</button>
          {:else if isMe}
            <span class="text-xs text-zinc-500">Change your own password and email on your account page.</span>
          {/if}
        </div>
      </div>
    </div>
  {/if}
</li>
