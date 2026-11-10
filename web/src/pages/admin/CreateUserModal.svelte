<script>
  import { api } from '../../lib/api.js';
  import { toast } from '../../lib/state.svelte.js';
  import Modal from '../../components/Modal.svelte';
  import Field from '../../components/Field.svelte';
  import PasswordRules from '../../components/PasswordRules.svelte';
  import { btnGhost, btnPrimary, input, label } from './ui.js';
  import { can } from './rights.js';

  let { open = $bindable(false), data, reload } = $props();

  const empty = () => ({ username: '', email: '', password: '', invite: false, group_ids: [], admin: false });
  let draft = $state(empty());
  let error = $state('');
  let busy = $state(false);

  $effect(() => {
    if (open) {
      draft = empty();
      error = '';
    }
  });

  function toggleGroup(id) {
    draft.group_ids = draft.group_ids.includes(id) ? draft.group_ids.filter((g) => g !== id) : [...draft.group_ids, id];
  }

  async function submit(e) {
    e.preventDefault();
    error = '';
    busy = true;
    const r = await api('/admin/users', { ...draft, password: draft.invite ? '' : draft.password });
    busy = false;
    if (!r.ok) {
      error = r.data.error;
      return;
    }
    open = false;
    if (r.data.warning) toast(r.data.warning, 'error');
    else toast(draft.invite ? `Invite sent to ${draft.email}.` : `${draft.username} created. Tell them their password.`);
    reload();
  }
</script>

<Modal bind:open title="Create user">
  <form class="space-y-4" onsubmit={submit}>
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
        class="rounded-md px-3 py-1.5 text-sm font-medium transition {!draft.invite ? 'bg-indigo-500 text-[var(--accent-fg,#fff)]' : 'text-zinc-400 hover:text-zinc-200'}">
        I set the password
      </button>
      <button type="button" role="radio" aria-checked={draft.invite} onclick={() => (draft.invite = true)} disabled={!data.mail_ready}
        title={data.mail_ready ? '' : 'Set up email first (Settings → Email)'}
        class="rounded-md px-3 py-1.5 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-40 {draft.invite ? 'bg-indigo-500 text-[var(--accent-fg,#fff)]' : 'text-zinc-400 hover:text-zinc-200'}">
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
              <button type="button" aria-pressed={draft.group_ids.includes(g.id)} onclick={() => toggleGroup(g.id)}
                class="rounded-full px-2.5 py-1 text-xs font-medium ring-1 transition {draft.group_ids.includes(g.id) ? 'bg-indigo-500 text-[var(--accent-fg,#fff)] ring-indigo-400' : 'bg-zinc-800 text-zinc-300 ring-zinc-700 hover:bg-zinc-700'}">
                {g.name}
              </button>
            {/each}
          {/if}
          {#if data.me.admin}
            <button type="button" aria-pressed={draft.admin} onclick={() => (draft.admin = !draft.admin)}
              class="rounded-full px-2.5 py-1 text-xs font-medium ring-1 transition {draft.admin ? 'bg-indigo-500 text-[var(--accent-fg,#fff)] ring-indigo-400' : 'bg-zinc-800 text-zinc-300 ring-zinc-700 hover:bg-zinc-700'}">
              Admins
            </button>
          {/if}
        </div>
        {#if draft.admin}<p class="mt-1.5 text-xs text-amber-300/90">Admins can do everything, including managing other admins.</p>{/if}
      </div>
    {/if}

    {#if error}<p class="text-sm text-rose-400" role="alert">{error}</p>{/if}
    <div class="flex justify-end gap-2">
      <button type="button" class={btnGhost} onclick={() => (open = false)}>Cancel</button>
      <button type="submit" class={btnPrimary} disabled={busy}>{draft.invite ? 'Create and send invite' : 'Create user'}</button>
    </div>
  </form>
</Modal>
