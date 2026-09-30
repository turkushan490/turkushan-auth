<script>
  import { untrack } from 'svelte';
  import Toggle from '../../components/Toggle.svelte';
  import { btnGhost, btnPrimary, input, label } from './ui.js';

  // onsave(values) resolves to an error message, or '' when saved.
  let { initial = {}, base, submitText = 'Save', onsave, oncancel } = $props();

  // The form edits a copy; later changes to `initial` (a reload) don't overwrite what's typed.
  const start = untrack(() => ({ ...initial }));
  let name = $state(start.name ?? '');
  let host = $state(start.host ?? '');
  let upstream = $state(start.upstream ?? '');
  let requireApproval = $state(start.require_approval ?? true);
  let requireEmail = $state(start.require_email ?? false);
  let error = $state('');
  let busy = $state(false);

  async function submit(e) {
    e.preventDefault();
    busy = true;
    error = await onsave({ name, host, upstream, require_approval: requireApproval, require_email: requireEmail });
    busy = false;
  }
</script>

<form class="space-y-4" onsubmit={submit}>
  <div class="grid gap-4 sm:grid-cols-2">
    <div>
      <label class={label} for="site-name">Name</label>
      <input id="site-name" class={input} bind:value={name} placeholder="LANraragi" required />
    </div>
    <div>
      <label class={label} for="site-host">Hostname</label>
      <input id="site-host" class={input} bind:value={host} placeholder="manga.{base}" autocapitalize="off" spellcheck="false" required />
    </div>
  </div>
  <div>
    <label class={label} for="site-upstream">Where the app runs</label>
    <input id="site-upstream" class={input} bind:value={upstream} placeholder="http://192.168.0.6:3000" autocapitalize="off" spellcheck="false" required />
    <p class="mt-1.5 text-xs text-zinc-500">The address NPM forwards to, same as the Forward Hostname/IP and Port in NPM.</p>
  </div>
  <div class="space-y-4 rounded-lg border border-zinc-800 bg-zinc-950/40 p-4">
    <Toggle bind:checked={requireApproval} label="Needs approval" description="New users wait until you approve them. Off = everyone who is signed in can open it." />
    <Toggle bind:checked={requireEmail} label="Needs a verified email" description="Users must have a verified email address before they can open this site." />
  </div>
  {#if error}<p class="text-sm text-rose-400">{error}</p>{/if}
  <div class="flex justify-end gap-2">
    {#if oncancel}<button type="button" class={btnGhost} onclick={oncancel}>Cancel</button>{/if}
    <button type="submit" class={btnPrimary} disabled={busy}>{submitText}</button>
  </div>
</form>
