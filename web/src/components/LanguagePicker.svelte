<script>
  import { tick } from 'svelte';
  import { i18n, offered, setLanguage, t } from '../lib/i18n.svelte.js';
  import { byCode } from '../locales/index.js';
  import { api } from '../lib/api.js';
  import { app } from '../lib/state.svelte.js';

  // compact: a small globe button (sign-in pages); otherwise a full-width field (account page).
  let { compact = true } = $props();

  let open = $state(false);
  let query = $state('');
  let active = $state(0);
  let root = $state();
  let searchBox = $state();
  const listId = 'lang-list-' + Math.random().toString(36).slice(2, 8);

  const all = $derived(offered(app.session?.appearance));
  // Matches the native name, the English name and the code: "türk", "turkish" and "tr" all find Türkçe.
  const fold = (s) => s.toLocaleLowerCase().normalize('NFD').replace(/\p{M}/gu, '');
  const matches = $derived.by(() => {
    const q = fold(query.trim());
    if (!q) return all;
    return all.filter((l) => [l.name, l.english, l.code].some((s) => fold(s).includes(q)));
  });
  $effect(() => {
    void query;
    active = 0;
  });

  async function toggle() {
    open = !open;
    if (open) {
      query = '';
      active = Math.max(0, matches.findIndex((l) => l.code === i18n.lang));
      await tick();
      searchBox?.focus();
      document.getElementById(`${listId}-${active}`)?.scrollIntoView({ block: 'nearest' });
    }
  }

  async function choose(code) {
    open = false;
    await setLanguage(code, { remember: true });
    // Signed in: remember it on the account too, for the mails.
    if (app.session?.authenticated) {
      api('/account/language', { lang: code }, 'PUT');
      if (app.session.user) app.session.user.lang = code;
    }
  }

  function onkeydown(e) {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      active = Math.min(active + 1, matches.length - 1);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      active = Math.max(active - 1, 0);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (matches[active]) choose(matches[active].code);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      open = false;
      root?.querySelector('button')?.focus();
      return;
    } else {
      return;
    }
    tick().then(() => document.getElementById(`${listId}-${active}`)?.scrollIntoView({ block: 'nearest' }));
  }

  function onfocusout(e) {
    if (open && !root.contains(e.relatedTarget)) open = false;
  }
</script>

{#if all.length > 1}
  <div class="relative {compact ? '' : 'w-full'}" bind:this={root} {onfocusout}>
    <button
      type="button"
      onclick={toggle}
      aria-haspopup="listbox"
      aria-expanded={open}
      class="{compact
        ? 'inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs font-medium text-zinc-400 hover:bg-zinc-800/70 hover:text-zinc-100'
        : 'flex w-full items-center justify-between gap-2 rounded-lg border border-zinc-700/80 bg-zinc-950/60 px-3 py-2.5 text-sm text-zinc-100 hover:border-zinc-600'} transition focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500"
    >
      <span class="inline-flex items-center gap-2">
        <svg viewBox="0 0 24 24" class="size-4 shrink-0" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <circle cx="12" cy="12" r="9" /><path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18" />
        </svg>
        <span class="sr-only">{t('lang.label')}:</span>
        {byCode[i18n.lang]?.name || 'English'}
      </span>
      <svg viewBox="0 0 24 24" class="size-3.5 shrink-0 transition {open ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
    </button>

    {#if open}
      <div class="absolute z-50 mt-1.5 w-64 overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900 shadow-2xl shadow-black/60 {compact ? 'end-0' : 'start-0'}">
        <div class="border-b border-zinc-800 p-2">
          <input
            bind:this={searchBox}
            bind:value={query}
            {onkeydown}
            type="search"
            role="combobox"
            aria-expanded="true"
            aria-controls={listId}
            aria-activedescendant={matches[active] ? `${listId}-${active}` : undefined}
            aria-label={t('lang.search')}
            placeholder={t('lang.search')}
            autocomplete="off"
            spellcheck="false"
            class="w-full rounded-lg border border-zinc-700/80 bg-zinc-950/60 px-3 py-2 text-sm text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-indigo-500"
          />
        </div>
        <ul id={listId} role="listbox" aria-label={t('lang.label')} class="max-h-72 overflow-y-auto py-1">
          {#each matches as l, i (l.code)}
            <!-- Keyboard: arrows and Enter in the search box above (combobox pattern). -->
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <li
              id="{listId}-{i}"
              role="option"
              aria-selected={l.code === i18n.lang}
              class="flex cursor-pointer items-center justify-between gap-3 px-3 py-2 text-sm {i === active ? 'bg-zinc-800 text-zinc-50' : 'text-zinc-300'}"
              onmousemove={() => (active = i)}
              onmousedown={(e) => e.preventDefault()}
              onclick={() => choose(l.code)}
            >
              <span dir="auto">{l.name}</span>
              <span class="flex items-center gap-2 text-xs text-zinc-500">
                {#if l.name !== l.english}{l.english}{/if}
                {#if l.code === i18n.lang}
                  <svg viewBox="0 0 24 24" class="size-3.5 text-indigo-400" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12l5 5L20 7" /></svg>
                {/if}
              </span>
            </li>
          {:else}
            <li class="px-3 py-3 text-sm text-zinc-500">{t('lang.none')}</li>
          {/each}
        </ul>
      </div>
    {/if}
  </div>
{/if}
