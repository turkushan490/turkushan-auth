<script>
  import { api } from '../../lib/api.js';
  import { confirmDialog } from '../../lib/confirm.svelte.js';
  import { card, badge } from './ui.js';

  // run(request, okText) comes from Groups.svelte: busy state, toast and reload.
  let { data, open, ontoggle, busy, run } = $props();

  const admins = $derived(data.users.filter((u) => u.is_admin));
  const sectionTitle = 'mb-2 text-xs font-semibold uppercase tracking-wider text-zinc-500';

  async function setAdmin(u, admin) {
    const question = admin
      ? { title: `Make ${u.username} an admin?`, text: 'Admins can do everything, including removing you as admin.', ok: 'Make admin' }
      : { title: `Remove ${u.username} as admin?`, text: 'They keep their groups, but lose everything only admins can do.', ok: 'Remove admin', danger: true };
    if (!(await confirmDialog(question))) return;
    run(api(`/admin/users/${u.id}/admin`, { admin }, 'PUT'), admin ? `${u.username} is now an admin.` : `${u.username} is no longer an admin.`);
  }
</script>

<li class="{card} overflow-hidden">
  <button type="button" class="flex w-full items-center gap-3 p-4 text-left transition hover:bg-zinc-800/30 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-indigo-500" onclick={ontoggle} aria-expanded={open}>
    <div class="grid size-10 shrink-0 place-items-center rounded-full bg-indigo-500/20 text-indigo-300" aria-hidden="true">
      <svg viewBox="0 0 24 24" class="size-5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3l8 3v6c0 4.5-3.2 8.3-8 9-4.8-.7-8-4.5-8-9V6z" /></svg>
    </div>
    <div class="min-w-0 flex-1">
      <div class="flex flex-wrap items-center gap-1.5">
        <span class="font-medium">Admins</span>
        <span class="{badge} bg-zinc-800 text-zinc-400 ring-zinc-700">built in</span>
      </div>
      <p class="mt-0.5 text-xs text-zinc-500">{admins.length} member{admins.length === 1 ? '' : 's'} · all rights, every site</p>
    </div>
    <svg viewBox="0 0 24 24" class="size-4 shrink-0 text-zinc-500 transition {open ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
  </button>
  {#if open}
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
          <select class="rounded-full border border-dashed border-zinc-700 bg-zinc-900 px-2 py-0.5 text-xs text-zinc-400 outline-none hover:border-zinc-500 focus:border-indigo-500" aria-label="Make someone admin" disabled={busy}
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
