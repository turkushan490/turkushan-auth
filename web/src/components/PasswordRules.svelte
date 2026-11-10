<script>
  import { passwordRules } from '../lib/password.js';
  import { t } from '../lib/i18n.svelte.js';

  let { password } = $props();
  const rules = $derived(passwordRules(password));
</script>

<ul class="mt-2 space-y-1 text-xs">
  {#each rules as rule}
    <li class="flex items-center gap-2 {rule.ok ? 'text-emerald-400' : 'text-zinc-500'}">
      {#if rule.ok}
        <svg viewBox="0 0 24 24" class="size-3.5" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12l5 5L20 7" /></svg>
      {:else}
        <svg viewBox="0 0 24 24" class="size-3.5" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="12" cy="12" r="8" /></svg>
      {/if}
      {rule.text}<span class="sr-only">{rule.ok ? t('pw.done') : t('pw.not_yet')}</span>
    </li>
  {/each}
</ul>
