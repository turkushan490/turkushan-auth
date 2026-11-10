<script>
  import { api } from '../../lib/api.js';
  import { confirmDialog } from '../../lib/confirm.svelte.js';
  import { btnGhost, input } from './ui.js';
  import { t } from '../../lib/i18n.svelte.js';

  // The auto-add rules of one group. run(request, okText) comes from Groups.svelte.
  let { g, rules, canManage, busy, run } = $props();

  // Built at render, so the labels follow a language change.
  const ruleKinds = $derived.by(() => {
    const discordHint = t('rules.discord_hint');
    return [
      { id: 'everyone', name: t('rules.kind.everyone'), hint: t('rules.kind.everyone_hint') },
      { id: 'email', name: t('rules.kind.email'), hint: t('rules.kind.email_hint'), placeholder: 'name@example.com' },
      { id: 'domain', name: t('rules.kind.domain'), hint: t('rules.kind.domain_hint'), placeholder: 'example.com' },
      { id: 'method', name: t('rules.kind.method'), hint: t('rules.kind.method_hint') },
      { id: 'discord_server', name: t('rules.kind.discord_server'), hint: discordHint, placeholder: t('rules.server_id') },
      { id: 'discord_role', name: t('rules.kind.discord_role'), hint: discordHint, placeholder: t('rules.server_id') },
    ];
  });
  let ruleRole = $state('');
  let ruleMethod = $state('discord');
  let ruleKind = $state('everyone');
  let ruleValue = $state('');
  const kind = $derived(ruleKinds.find((k) => k.id === ruleKind));

  async function addRule(e) {
    e.preventDefault();
    const value = ruleKind === 'method' ? ruleMethod : ruleKind === 'discord_role' ? `${ruleValue.trim()}:${ruleRole.trim()}` : ruleValue;
    if (await run(api(`/admin/groups/${g.id}/rules`, { kind: ruleKind, value }), t('rules.added'))) ruleValue = ruleRole = '';
  }

  async function removeRule(r) {
    if (!(await confirmDialog({ title: t('rules.remove_title'), text: t('rules.remove_text', { group: g.name }), ok: t('rules.remove'), danger: true }))) return;
    run(api(`/admin/rules/${r.id}`, undefined, 'DELETE'), t('rules.removed'));
  }

  // Called at render, so it follows a language change.
  function ruleText(r) {
    switch (r.kind) {
      case 'everyone':
        return t('rules.text.everyone');
      case 'email':
        return t('rules.text.email', { value: r.value });
      case 'domain':
        return t('rules.text.domain', { value: r.value });
      case 'method':
        if (r.value === 'discord') return t('rules.text.method_discord');
        if (r.value === 'google') return t('rules.text.method_google');
        if (r.value === 'password') return t('rules.text.method_password');
        return t('rules.text.method_other', { value: r.value });
      case 'discord_server':
        return t('rules.text.discord_server', { value: r.value });
      case 'discord_role':
        return t('rules.text.discord_role', { role: r.value.split(':')[1], server: r.value.split(':')[0] });
      default:
        return r.kind;
    }
  }

  const select = 'rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 outline-none focus:border-indigo-500';
</script>

{#if rules.length}
  <ul class="mb-3 divide-y divide-zinc-800/80 rounded-lg border border-zinc-800">
    {#each rules as r (r.id)}
      <li class="flex items-center justify-between gap-3 px-3 py-2 text-sm">
        <span>{ruleText(r)}</span>
        {#if canManage}<button class={btnGhost} disabled={busy} onclick={() => removeRule(r)}>{t('rules.remove')}</button>{/if}
      </li>
    {/each}
  </ul>
{:else}
  <p class="mb-3 text-sm text-zinc-500">{t('rules.none')}</p>
{/if}
{#if canManage}
  <form class="flex flex-col gap-2 sm:flex-row" onsubmit={addRule}>
    <select bind:value={ruleKind} class={select} aria-label={t('rules.kind_aria')}>
      {#each ruleKinds as k}<option value={k.id}>{k.name}</option>{/each}
    </select>
    {#if ruleKind === 'method'}
      <select bind:value={ruleMethod} class={select} aria-label={t('rules.kind.method')}>
        <option value="discord">Discord</option>
        <option value="google">Google</option>
        <option value="password">{t('rules.method_password')}</option>
      </select>
    {:else if kind.placeholder}
      <input class={input} bind:value={ruleValue} placeholder={kind.placeholder} autocapitalize="off" spellcheck="false" required aria-label={kind.placeholder} />
    {/if}
    {#if ruleKind === 'discord_role'}
      <input class={input} bind:value={ruleRole} placeholder={t('rules.role_id')} autocapitalize="off" spellcheck="false" required aria-label={t('rules.role_id')} />
    {/if}
    <button type="submit" class="{btnGhost} shrink-0" disabled={busy}>{t('rules.add')}</button>
  </form>
  <p class="mt-2 text-xs text-zinc-500">{kind.hint} {t('rules.help')}</p>
{/if}
