<script>
  import { onDestroy, untrack } from 'svelte';
  import { api, apiUpload } from '../../lib/api.js';
  import { app, refreshSession, setLeaveGuard, toast } from '../../lib/state.svelte.js';
  import { fonts, accentPresets, defaultAccent, isHex, accentReadability } from '../../lib/appearance.js';
  import { confirmDialog } from '../../lib/confirm.svelte.js';
  import Backdrop from '../../components/Backdrop.svelte';
  import Logo from '../../components/Logo.svelte';
  import { card, btnGhost, btnPrimary, btnDanger, input, label } from './ui.js';
  import { t, tn } from '../../lib/i18n.svelte.js';
  import { languages } from '../../locales/index.js';

  const saved = untrack(() => app.session.appearance);
  const baseDomain = untrack(() => app.session.brand);

  // The draft is shown live on the whole page (app.preview) until it's saved or the tab is left.
  let draft = $state({
    site_name: saved.site_name === baseDomain ? '' : saved.site_name,
    page_title: saved.page_title === saved.site_name ? '' : saved.page_title,
    accent: saved.accent || '',
    font: saved.font || '',
    bg_type: saved.bg_type || 'default',
    bg_color: saved.bg_color || '#0f172a',
    bg_color2: saved.bg_color2 || '#1e1b4b',
    overlay: saved.overlay ?? 50,
    logo_url: saved.logo_url,
    bg_image_url: saved.bg_image_url,
    default_lang: saved.default_lang === 'en' ? '' : saved.default_lang || '',
    languages: [...(saved.languages || [])],
  });
  let busy = $state('');

  const preview = $derived({
    ...draft,
    site_name: draft.site_name.trim() || baseDomain,
    page_title: draft.page_title.trim() || draft.site_name.trim() || baseDomain,
  });
  $effect(() => {
    app.preview = preview;
  });
  onDestroy(() => (app.preview = null));

  // Unsaved changes: ask before leaving the tab or closing the page.
  const fields = ['site_name', 'page_title', 'accent', 'font', 'bg_type', 'bg_color', 'bg_color2', 'overlay', 'default_lang', 'languages'];
  const snapshot = () => JSON.stringify(fields.map((f) => (Array.isArray(draft[f]) ? JSON.stringify([...draft[f]].sort()) : String(draft[f] ?? ''))));
  let savedSnapshot = $state(untrack(snapshot));
  const dirty = $derived(snapshot() !== savedSnapshot);
  onDestroy(setLeaveGuard(() => (dirty ? t('appearance.leave') : '')));

  // Accent readability (WCAG): button text needs 4.5:1, accent text on the page 3:1.
  const readable = $derived(accentReadability(isHex(draft.accent) ? draft.accent : defaultAccent));

  const bgTypes = $derived([
    { id: 'default', name: t('appearance.bg.default') },
    { id: 'color', name: t('appearance.bg.color') },
    { id: 'gradient', name: t('appearance.bg.gradient') },
    { id: 'image', name: t('appearance.bg.image') },
  ]);

  // Languages. default_lang '' = English; languages [] = all of them are offered.
  // The server refuses a default that isn't in a non-empty offered list, so the
  // default select only lists offered languages and follows when one is unticked.
  const langValue = (code) => (code === 'en' ? '' : code);
  const offeredNow = $derived(draft.languages.length ? languages.filter((l) => draft.languages.includes(l.code)) : languages);
  let langFilter = $state('');
  const shownLangs = $derived.by(() => {
    const q = langFilter.trim().toLowerCase();
    if (!q) return languages;
    return languages.filter((l) => l.name.toLowerCase().includes(q) || l.english.toLowerCase().includes(q) || l.code.toLowerCase().includes(q));
  });

  function fixDefault() {
    if (!draft.languages.length) return;
    if (draft.languages.includes(draft.default_lang || 'en')) return;
    draft.default_lang = draft.languages.includes('en') ? '' : langValue(languages.find((l) => draft.languages.includes(l.code)).code);
  }

  function toggleLang(code, on) {
    draft.languages = on ? [...draft.languages.filter((c) => c !== code), code] : draft.languages.filter((c) => c !== code);
    fixDefault();
  }

  async function save() {
    busy = 'save';
    const r = await api(
      '/admin/appearance',
      {
        site_name: draft.site_name,
        page_title: draft.page_title,
        accent: isHex(draft.accent) ? draft.accent : '',
        font: draft.font,
        bg_type: draft.bg_type === 'image' && !draft.bg_image_url ? 'default' : draft.bg_type,
        bg_color: draft.bg_color,
        bg_color2: draft.bg_color2,
        overlay: Number(draft.overlay),
        default_lang: draft.default_lang,
        languages: languages.map((l) => l.code).filter((c) => draft.languages.includes(c)),
      },
      'PUT',
    );
    busy = '';
    if (!r.ok) return toast(r.data.error, 'error');
    savedSnapshot = snapshot();
    await refreshSession();
    toast(t('appearance.saved'));
  }

  async function upload(kind, e) {
    const file = e.currentTarget.files?.[0];
    e.currentTarget.value = '';
    if (!file) return;
    busy = kind;
    const r = await apiUpload(`/admin/appearance/${kind}`, file);
    busy = '';
    if (!r.ok) return toast(r.data.error, 'error');
    if (kind === 'logo') {
      draft.logo_url = r.data.appearance.logo_url;
    } else {
      draft.bg_image_url = r.data.appearance.bg_image_url;
      draft.bg_type = 'image';
    }
    await refreshSession();
    toast(kind === 'logo' ? t('appearance.logo_uploaded') : t('appearance.bg_uploaded'));
  }

  async function remove(kind) {
    busy = kind;
    const r = await api(`/admin/appearance/${kind}`, undefined, 'DELETE');
    busy = '';
    if (!r.ok) return toast(r.data.error, 'error');
    if (kind === 'logo') {
      draft.logo_url = '';
    } else {
      draft.bg_image_url = '';
      if (draft.bg_type === 'image') draft.bg_type = 'default';
    }
    await refreshSession();
    toast(kind === 'logo' ? t('appearance.logo_removed') : t('appearance.bg_removed'));
  }

  async function resetAll() {
    if (!(await confirmDialog({ title: t('appearance.reset_title'), text: t('appearance.reset_text'), ok: t('appearance.reset'), danger: true }))) return;
    Object.assign(draft, { site_name: '', page_title: '', accent: '', font: '', bg_type: 'default', overlay: 50 });
    save();
  }

  const colorInput = 'h-9 w-12 shrink-0 cursor-pointer rounded-lg border border-zinc-700 bg-zinc-950 p-1';
  const uploadBtn = `${btnGhost} cursor-pointer`;
</script>

<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
  <div>
    <h2 class="text-lg font-semibold tracking-tight">{t('appearance.title')}</h2>
    <p class="text-sm text-zinc-400">{t('appearance.intro')}</p>
  </div>
  <div class="flex gap-2">
    <button class={btnGhost} onclick={resetAll} disabled={busy !== ''}>{t('appearance.reset')}</button>
    <button class={btnPrimary} onclick={save} disabled={busy !== ''}>{busy === 'save' ? t('appearance.saving') : t('appearance.save')}</button>
  </div>
</div>

<div class="grid gap-6 lg:grid-cols-[1fr_20rem]">
  <div class="space-y-6">
    <section class="{card} p-5 sm:p-6">
      <h3 class="font-semibold">{t('appearance.logo_title')}</h3>
      <div class="mt-4 flex flex-wrap items-center gap-4">
        <div class="grid h-24 w-40 shrink-0 place-items-center rounded-lg border border-dashed border-zinc-700 bg-zinc-950/60 p-3">
          <Logo appearance={preview} />
        </div>
        <div class="space-y-2">
          <div class="flex flex-wrap gap-2">
            <label class={uploadBtn}>
              {busy === 'logo' ? t('appearance.uploading') : draft.logo_url ? t('appearance.logo_replace') : t('appearance.logo_upload')}
              <input type="file" class="sr-only" accept="image/png,image/jpeg,image/webp,image/gif,image/svg+xml" onchange={(e) => upload('logo', e)} disabled={busy !== ''} />
            </label>
            {#if draft.logo_url}
              <button class={btnDanger} onclick={() => remove('logo')} disabled={busy !== ''}>{t('appearance.remove')}</button>
            {/if}
          </div>
          <p class="text-xs text-zinc-500">{t('appearance.logo_help')}</p>
        </div>
      </div>

      <div class="mt-5 grid gap-4 sm:grid-cols-2">
        <div>
          <label class={label} for="site-name">{t('appearance.site_name')}</label>
          <input id="site-name" class={input} bind:value={draft.site_name} maxlength="40" placeholder={baseDomain} />
          <p class="mt-1.5 text-xs text-zinc-500">{t('appearance.site_name_help')}</p>
        </div>
        <div>
          <label class={label} for="page-title">{t('appearance.page_title')}</label>
          <input id="page-title" class={input} bind:value={draft.page_title} maxlength="60" placeholder={preview.site_name} />
        </div>
      </div>
    </section>

    <section class="{card} p-5 sm:p-6">
      <h3 class="font-semibold">{t('appearance.color_title')}</h3>
      <div class="mt-4">
        <span class={label}>{t('appearance.accent')}</span>
        <div class="flex flex-wrap items-center gap-2">
          {#each accentPresets as c}
            <button
              type="button"
              class="size-8 rounded-full ring-2 ring-offset-2 ring-offset-zinc-900 transition {(draft.accent || defaultAccent).toLowerCase() === c ? 'ring-white' : 'ring-transparent hover:ring-zinc-600'}"
              style="background:{c}"
              aria-label={t('appearance.accent_aria', { color: c })}
              onclick={() => (draft.accent = c === defaultAccent ? '' : c)}
            ></button>
          {/each}
          <span class="mx-1 h-6 w-px bg-zinc-700"></span>
          <input type="color" class={colorInput} value={isHex(draft.accent) ? draft.accent : defaultAccent} oninput={(e) => (draft.accent = e.currentTarget.value)} aria-label={t('appearance.accent_custom')} />
          <input class="{input} !w-28 font-mono" bind:value={draft.accent} placeholder={defaultAccent} maxlength="7" aria-label={t('appearance.accent_code')} />
        </div>
        <p class="mt-1.5 text-xs text-zinc-500">{t('appearance.accent_help')}</p>
        {#if !readable.ok}
          <p class="mt-2 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-200" role="status">
            {#if readable.button < 4.5}{t('appearance.contrast_button', { ratio: readable.button.toFixed(1) })}{/if}
            {#if readable.onPage < 3}{t('appearance.contrast_page', { ratio: readable.onPage.toFixed(1) })}{/if}
            {t('appearance.contrast_fix')}
          </p>
        {/if}
      </div>

      <div class="mt-5">
        <label class={label} for="font">{t('appearance.font')}</label>
        <select id="font" bind:value={draft.font} class="{input} sm:w-64">
          {#each fonts as f}
            <option value={f.id}>{f.name}</option>
          {/each}
        </select>
        <p class="mt-2 text-sm text-zinc-300" style="font-family:{fonts.find((f) => f.id === draft.font)?.family}">
          {t('appearance.font_sample')}
        </p>
      </div>
    </section>

    <section class="{card} p-5 sm:p-6">
      <h3 class="font-semibold">{t('appearance.bg_title')}</h3>
      <div class="mt-4 inline-flex flex-wrap gap-1 rounded-lg border border-zinc-800 bg-zinc-950/60 p-1" role="radiogroup" aria-label={t('appearance.bg_type')}>
        {#each bgTypes as bt}
          <button
            type="button"
            role="radio"
            aria-checked={draft.bg_type === bt.id}
            class="rounded-md px-3 py-1.5 text-sm font-medium transition {draft.bg_type === bt.id ? 'bg-indigo-500 text-white' : 'text-zinc-400 hover:text-zinc-200'}"
            onclick={() => (draft.bg_type = bt.id)}
          >
            {bt.name}
          </button>
        {/each}
      </div>

      {#if draft.bg_type === 'default'}
        <p class="mt-4 text-sm text-zinc-400">{t('appearance.bg_default_help')}</p>
      {:else if draft.bg_type === 'color' || draft.bg_type === 'gradient'}
        <div class="mt-4 flex flex-wrap items-end gap-4">
          <div>
            <span class={label}>{draft.bg_type === 'gradient' ? t('appearance.bg_from') : t('appearance.bg.color')}</span>
            <div class="flex items-center gap-2">
              <input type="color" class={colorInput} bind:value={draft.bg_color} aria-label={t('appearance.bg_color')} />
              <input class="{input} !w-28 font-mono" bind:value={draft.bg_color} maxlength="7" aria-label={t('appearance.bg_color_code')} />
            </div>
          </div>
          {#if draft.bg_type === 'gradient'}
            <div>
              <span class={label}>{t('appearance.bg_to')}</span>
              <div class="flex items-center gap-2">
                <input type="color" class={colorInput} bind:value={draft.bg_color2} aria-label={t('appearance.bg_color2')} />
                <input class="{input} !w-28 font-mono" bind:value={draft.bg_color2} maxlength="7" aria-label={t('appearance.bg_color2_code')} />
              </div>
            </div>
          {/if}
        </div>
        <p class="mt-3 text-xs text-zinc-500">{t('appearance.bg_color_help')}</p>
      {:else}
        <div class="mt-4 flex flex-wrap items-center gap-2">
          <label class={uploadBtn}>
            {busy === 'background' ? t('appearance.uploading') : draft.bg_image_url ? t('appearance.bg_replace') : t('appearance.bg_upload')}
            <input type="file" class="sr-only" accept="image/png,image/jpeg,image/webp" onchange={(e) => upload('background', e)} disabled={busy !== ''} />
          </label>
          {#if draft.bg_image_url}
            <button class={btnDanger} onclick={() => remove('background')} disabled={busy !== ''}>{t('appearance.remove')}</button>
          {/if}
        </div>
        <p class="mt-2 text-xs text-zinc-500">
          {t('appearance.bg_image_help')}
        </p>
        {#if draft.bg_image_url}
          <div class="mt-4">
            <label class={label} for="overlay">{t('appearance.darken', { n: draft.overlay })}</label>
            <input id="overlay" type="range" min="0" max="90" step="5" bind:value={draft.overlay} class="w-full accent-indigo-500 sm:w-80" />
            <p class="mt-1 text-xs text-zinc-500">{t('appearance.darken_help')}</p>
          </div>
        {:else}
          <p class="mt-3 text-sm text-amber-300/90">{t('appearance.bg_no_image')}</p>
        {/if}
      {/if}
    </section>

    <section class="{card} p-5 sm:p-6">
      <h3 class="font-semibold">{t('appearance.lang.title')}</h3>
      <div class="mt-4">
        <label class={label} for="default-lang">{t('appearance.lang.default')}</label>
        <select id="default-lang" bind:value={draft.default_lang} class="{input} sm:w-64">
          {#each offeredNow as l (l.code)}
            <option value={langValue(l.code)}>{l.name}</option>
          {/each}
        </select>
        <p class="mt-1.5 text-xs text-zinc-500">{t('appearance.lang.default_help')}</p>
      </div>

      <div class="mt-5">
        <div class="mb-1.5 flex flex-wrap items-center justify-between gap-2">
          <span class="text-sm font-medium text-zinc-300" id="offered-langs">{t('appearance.lang.offered')}</span>
          <span class="flex items-center gap-2 text-xs text-zinc-500">
            {draft.languages.length ? tn('appearance.lang.count', draft.languages.length, { total: languages.length }) : t('appearance.lang.all')}
            {#if draft.languages.length}
              <button type="button" class="font-medium text-indigo-400 hover:text-indigo-300" onclick={() => (draft.languages = [])}>{t('appearance.lang.offer_all')}</button>
            {/if}
          </span>
        </div>
        <input type="search" class="{input} sm:w-64" bind:value={langFilter} placeholder={t('appearance.lang.filter')} aria-label={t('appearance.lang.filter')} autocomplete="off" />
        <div class="mt-2 grid max-h-64 gap-x-4 gap-y-1 overflow-y-auto rounded-lg border border-zinc-800 bg-zinc-950/40 p-3 sm:grid-cols-2" role="group" aria-labelledby="offered-langs">
          {#each shownLangs as l (l.code)}
            <label class="flex cursor-pointer items-center gap-2 rounded px-1 py-1 text-sm text-zinc-300 hover:bg-zinc-800/40">
              <input type="checkbox" class="size-4 shrink-0 accent-indigo-500" checked={draft.languages.includes(l.code)} onchange={(e) => toggleLang(l.code, e.currentTarget.checked)} />
              <span lang={l.code} dir={l.rtl ? 'rtl' : 'ltr'}>{l.name}</span>
              {#if l.english !== l.name}<span class="truncate text-xs text-zinc-500">{l.english}</span>{/if}
            </label>
          {:else}
            <p class="text-sm text-zinc-500">{t('appearance.lang.no_match')}</p>
          {/each}
        </div>
        <p class="mt-1.5 text-xs text-zinc-500">{t('appearance.lang.offered_help')}</p>
      </div>
    </section>
  </div>

  <!-- Small copy of the sign-in page, so the result is visible without leaving the panel. -->
  <aside class="lg:sticky lg:top-32 lg:self-start">
    <p class="mb-2 text-xs font-semibold uppercase tracking-wider text-zinc-500">{t('appearance.preview')}</p>
    <div class="relative overflow-hidden rounded-xl border border-zinc-800" style="font-family:{fonts.find((f) => f.id === draft.font)?.family}">
      <Backdrop appearance={preview} fixed={false} />
      <div class="relative flex flex-col items-center px-6 py-8">
        <Logo appearance={preview} />
        <p class="mt-2 text-xs font-medium text-zinc-300 [text-shadow:0_1px_6px_rgb(0_0_0/0.7)]">{preview.site_name}</p>
        <div class="mt-5 w-full rounded-xl border border-zinc-800 bg-zinc-900/80 p-4 backdrop-blur-md">
          <p class="text-sm font-semibold">{t('appearance.preview_sign_in')}</p>
          <div class="mt-3 h-7 rounded-md border border-zinc-700/80 bg-zinc-950/60"></div>
          <div class="mt-2 h-7 rounded-md border border-zinc-700/80 bg-zinc-950/60"></div>
          <div class="mt-3 grid h-8 place-items-center rounded-md bg-indigo-500 text-xs font-semibold text-white">{t('appearance.preview_sign_in')}</div>
          <p class="mt-3 border-t border-zinc-800 pt-3 text-center text-[11px] text-zinc-400">
            {t('appearance.preview_no_account')} <span class="font-medium text-indigo-400">{t('appearance.preview_create')}</span>
          </p>
        </div>
      </div>
    </div>
  </aside>
</div>
