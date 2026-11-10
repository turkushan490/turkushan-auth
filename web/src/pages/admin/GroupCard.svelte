<script>
  import { api } from '../../lib/api.js';
  import { confirmDialog } from '../../lib/confirm.svelte.js';
  import Toggle from '../../components/Toggle.svelte';
  import RuleEditor from './RuleEditor.svelte';
  import { card, btnDanger, btnGhost, btnPrimary, badge, input } from './ui.js';
  import { can, permLabels } from './rights.js';
  import { t, tn } from '../../lib/i18n.svelte.js';

  // run(request, okText) comes from Groups.svelte: busy state, toast and reload.
  let { data, g, open, ontoggle, busy, run } = $props();

  const manageGroups = $derived(can(data, 'groups'));
  const members = $derived.by(() => {
    const by = new Map(data.memberships.filter((m) => m.group_id === g.id).map((m) => [m.user_id, m.source]));
    return data.users.filter((u) => by.has(u.id)).map((u) => ({ ...u, source: by.get(u.id) }));
  });
  const rules = $derived(data.rules.filter((r) => r.group_id === g.id));
  const siteRight = (site) => data.group_sites.find((gs) => gs.group_id === g.id && gs.site_id === site.id) || { can_open: false, can_approve: false };

  const summary = $derived.by(() => {
    const parts = g.perms.map((p) => permLabels[p]?.name || p);
    const opens = data.group_sites.filter((gs) => gs.group_id === g.id && gs.can_open).length;
    const approves = data.group_sites.filter((gs) => gs.group_id === g.id && gs.can_approve).length;
    if (opens) parts.unshift(tn('groups.opens_sites', opens));
    if (approves) parts.push(tn('groups.approves_sites', approves));
    return parts;
  });

  async function setMember(u, member) {
    if (!member && !(await confirmDialog({ title: t('groups.remove_member_title', { user: u.username, group: g.name }), text: t('groups.remove_member_text'), ok: t('groups.remove'), danger: true }))) return;
    run(api(`/admin/groups/${g.id}/members`, { user_id: u.id, member }, 'PUT'), member ? t('groups.member_added', { user: u.username, group: g.name }) : t('groups.member_removed', { user: u.username, group: g.name }));
  }

  function togglePerm(perm, on) {
    const perms = on ? [...g.perms, perm] : g.perms.filter((p) => p !== perm);
    run(api(`/admin/groups/${g.id}/perms`, { perms }, 'PUT'), t(on ? 'groups.perm_on' : 'groups.perm_off', { group: g.name, perm: permLabels[perm].name }));
  }

  function setSiteRight(site, field, on) {
    const cur = siteRight(site);
    run(api(`/admin/groups/${g.id}/sites`, { site_id: site.id, can_open: cur.can_open, can_approve: cur.can_approve, [field]: on }, 'PUT'), t('groups.site_updated', { group: g.name, site: site.name }));
  }

  // --- rename / delete ---
  let editing = $state(false);
  let editName = $state('');
  let editDescription = $state('');

  function startEdit() {
    editing = true;
    editName = g.name;
    editDescription = g.description;
  }

  async function saveEdit(e) {
    e.preventDefault();
    if (await run(api(`/admin/groups/${g.id}`, { name: editName, description: editDescription }, 'PUT'), t('groups.saved'))) editing = false;
  }

  async function remove() {
    const n = members.length;
    const text = n ? tn('groups.delete_text', n) : '';
    if (!(await confirmDialog({ title: t('groups.delete_title', { name: g.name }), text, ok: t('groups.delete'), danger: true }))) return;
    run(api(`/admin/groups/${g.id}`, undefined, 'DELETE'), t('groups.deleted', { name: g.name }));
  }

  const sectionTitle = 'mb-2 text-xs font-semibold uppercase tracking-wider text-zinc-500';
</script>

<li class="{card} overflow-hidden">
  <button type="button" class="flex w-full items-center gap-3 p-4 text-start transition hover:bg-zinc-800/30 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-indigo-500" onclick={ontoggle} aria-expanded={open}>
    <div class="grid size-10 shrink-0 place-items-center rounded-full bg-zinc-800 text-sm font-semibold uppercase text-zinc-300" aria-hidden="true">{g.name.slice(0, 1)}</div>
    <div class="min-w-0 flex-1">
      <span class="font-medium">{g.name}</span>
      <p class="mt-0.5 truncate text-xs text-zinc-500">
        {tn('groups.member_count', members.length)}{g.description ? ` · ${g.description}` : ''}
      </p>
      <div class="mt-1.5 flex flex-wrap gap-1">
        {#each summary as part}
          <span class="{badge} bg-zinc-800 text-zinc-300 ring-zinc-700">{part}</span>
        {:else}
          <span class="text-xs text-zinc-500">{t('groups.no_rights')}</span>
        {/each}
        {#if rules.length}<span class="{badge} bg-sky-500/10 text-sky-300 ring-sky-500/25">{tn('groups.rule_count', rules.length)}</span>{/if}
      </div>
    </div>
    <svg viewBox="0 0 24 24" class="size-4 shrink-0 text-zinc-500 transition {open ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
  </button>

  {#if open}
    <div class="space-y-6 border-t border-zinc-800 bg-zinc-950/40 p-4">
      <section>
        <h4 class={sectionTitle}>{t('groups.members')}</h4>
        <div class="flex flex-wrap items-center gap-1.5">
          {#each members as u (u.id)}
            <span class="{badge} gap-1 bg-indigo-500/10 text-indigo-300 ring-indigo-400/20">
              {u.username}{#if u.source === 'rule'}<span class="text-indigo-400/60">{t('groups.by_rule')}</span>{/if}
              {#if manageGroups}
                <button type="button" class="-me-1 rounded-full px-1 text-indigo-300/70 hover:text-white" aria-label={t('groups.remove_member_aria', { name: u.username })} disabled={busy} onclick={() => setMember(u, false)}>×</button>
              {/if}
            </span>
          {:else}
            <span class="text-sm text-zinc-500">{t('groups.nobody')}</span>
          {/each}
          {#if manageGroups && data.users.some((u) => !members.some((m) => m.id === u.id))}
            <select class="rounded-full border border-dashed border-zinc-700 bg-zinc-900 px-2 py-0.5 text-xs text-zinc-400 outline-none hover:border-zinc-500 focus:border-indigo-500" aria-label={t('groups.add_member_aria', { name: g.name })} disabled={busy}
              onchange={(e) => {
                const u = data.users.find((u) => String(u.id) === e.currentTarget.value);
                e.currentTarget.value = '';
                if (u) setMember(u, true);
              }}>
              <option value="">{t('groups.add_member')}</option>
              {#each data.users.filter((u) => !members.some((m) => m.id === u.id)) as u (u.id)}<option value={String(u.id)}>{u.username}</option>{/each}
            </select>
          {/if}
        </div>
      </section>

      <section>
        <h4 class={sectionTitle}>{t('groups.sites')}</h4>
        {#if data.sites.length === 0}
          <p class="text-sm text-zinc-500">{t('groups.no_sites')}</p>
        {:else}
          <div class="overflow-hidden rounded-lg border border-zinc-800">
            <div class="grid grid-cols-[1fr_5rem_5rem] items-center gap-2 border-b border-zinc-800 bg-zinc-900/60 px-3 py-2 text-xs font-medium text-zinc-400">
              <span>{t('groups.site')}</span><span class="text-center">{t('groups.can_open')}</span><span class="text-center">{t('groups.can_approve')}</span>
            </div>
            {#each data.sites as site (site.id)}
              {@const sr = siteRight(site)}
              <div class="grid grid-cols-[1fr_5rem_5rem] items-center gap-2 border-b border-zinc-800/60 px-3 py-2 text-sm last:border-0">
                <div class="min-w-0">
                  <span class="font-medium">{site.name}</span>
                  <span class="block truncate text-xs text-zinc-500">{site.host}</span>
                </div>
                <div class="grid place-items-center">
                  <input type="checkbox" class="size-4 accent-indigo-500" checked={sr.can_open} disabled={busy || !data.me.admin} aria-label={t('groups.can_open_aria', { group: g.name, site: site.name })}
                    onchange={(e) => setSiteRight(site, 'can_open', e.currentTarget.checked)} />
                </div>
                <div class="grid place-items-center">
                  <input type="checkbox" class="size-4 accent-indigo-500" checked={sr.can_approve} disabled={busy || !data.me.admin} aria-label={t('groups.can_approve_aria', { group: g.name, site: site.name })}
                    onchange={(e) => setSiteRight(site, 'can_approve', e.currentTarget.checked)} />
                </div>
              </div>
            {/each}
          </div>
          <p class="mt-2 text-xs text-zinc-500">
            <span class="font-medium text-zinc-400">{t('groups.can_open_label')}</span> {t('groups.can_open_help')}
            <span class="font-medium text-zinc-400">{t('groups.can_approve_label')}</span> {t('groups.can_approve_help')}
          </p>
        {/if}
      </section>

      <section>
        <h4 class={sectionTitle}>{t('groups.rights')}</h4>
        <div class="grid gap-x-6 gap-y-4 rounded-lg border border-zinc-800 p-4 sm:grid-cols-2">
          {#each data.all_perms as perm}
            <Toggle checked={g.perms.includes(perm)} label={permLabels[perm].name} description={permLabels[perm].hint} disabled={busy || !data.me.admin}
              onchange={(v) => togglePerm(perm, v)} />
          {/each}
        </div>
        {#if !data.me.admin}<p class="mt-2 text-xs text-zinc-500">{t('groups.rights_admin_only')}</p>{/if}
      </section>

      <section>
        <h4 class={sectionTitle}>{t('groups.rules')}</h4>
        <RuleEditor {g} {rules} canManage={manageGroups} {busy} {run} />
      </section>

      {#if manageGroups}
        <section class="border-t border-zinc-800 pt-4">
          {#if editing}
            <form class="space-y-3" onsubmit={saveEdit}>
              <div class="grid gap-3 sm:grid-cols-2">
                <input class={input} bind:value={editName} maxlength="40" aria-label={t('groups.name_aria')} required />
                <input class={input} bind:value={editDescription} maxlength="200" placeholder={t('groups.description')} aria-label={t('groups.description')} />
              </div>
              <div class="flex justify-end gap-2">
                <button type="button" class={btnGhost} onclick={() => (editing = false)}>{t('groups.cancel')}</button>
                <button type="submit" class={btnPrimary} disabled={busy}>{t('groups.save')}</button>
              </div>
            </form>
          {:else}
            <div class="flex justify-end gap-2">
              <button class={btnGhost} disabled={busy} onclick={startEdit}>{t('groups.rename')}</button>
              <button class={btnDanger} disabled={busy} onclick={remove}>{t('groups.delete')}</button>
            </div>
          {/if}
        </section>
      {/if}
    </div>
  {/if}
</li>
