<script>
  import { api } from '../../lib/api.js';
  import { confirmDialog } from '../../lib/confirm.svelte.js';
  import { btnGhost, input } from './ui.js';

  // The auto-add rules of one group. run(request, okText) comes from Groups.svelte.
  let { g, rules, canManage, busy, run } = $props();

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

  async function addRule(e) {
    e.preventDefault();
    const value = ruleKind === 'method' ? ruleMethod : ruleKind === 'discord_role' ? `${ruleValue.trim()}:${ruleRole.trim()}` : ruleValue;
    if (await run(api(`/admin/groups/${g.id}/rules`, { kind: ruleKind, value }), 'Rule added. Matching users are in the group now.')) ruleValue = ruleRole = '';
  }

  async function removeRule(r) {
    if (!(await confirmDialog({ title: 'Remove this rule?', text: `People it added to ${g.name} are taken out again, unless they were added by hand.`, ok: 'Remove', danger: true }))) return;
    run(api(`/admin/rules/${r.id}`, undefined, 'DELETE'), 'Rule removed.');
  }

  const ruleText = (r) =>
    ({
      everyone: 'Everyone',
      email: `Email is ${r.value}`,
      domain: `Email ends with @${r.value}`,
      method: `Can sign in with ${{ discord: 'Discord', google: 'Google', password: 'a password' }[r.value] || r.value}`,
      discord_server: `In Discord server ${r.value}`,
      discord_role: `Has Discord role ${r.value.split(':')[1]} in server ${r.value.split(':')[0]}`,
    })[r.kind] || r.kind;

  const select = 'rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 outline-none focus:border-indigo-500';
</script>

{#if rules.length}
  <ul class="mb-3 divide-y divide-zinc-800/80 rounded-lg border border-zinc-800">
    {#each rules as r (r.id)}
      <li class="flex items-center justify-between gap-3 px-3 py-2 text-sm">
        <span>{ruleText(r)}</span>
        {#if canManage}<button class={btnGhost} disabled={busy} onclick={() => removeRule(r)}>Remove</button>{/if}
      </li>
    {/each}
  </ul>
{:else}
  <p class="mb-3 text-sm text-zinc-500">No rules. Members are added by hand.</p>
{/if}
{#if canManage}
  <form class="flex flex-col gap-2 sm:flex-row" onsubmit={addRule}>
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
