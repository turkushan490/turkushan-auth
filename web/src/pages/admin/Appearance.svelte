<script>
  import { onDestroy, untrack } from 'svelte';
  import { api, apiUpload } from '../../lib/api.js';
  import { app, refreshSession, toast } from '../../lib/state.svelte.js';
  import { fonts, accentPresets, defaultAccent, isHex } from '../../lib/appearance.js';
  import Backdrop from '../../components/Backdrop.svelte';
  import Logo from '../../components/Logo.svelte';
  import { card, btnGhost, btnPrimary, btnDanger, input, label } from './ui.js';

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

  const bgTypes = [
    { id: 'default', name: 'Default' },
    { id: 'color', name: 'Color' },
    { id: 'gradient', name: 'Gradient' },
    { id: 'image', name: 'Image' },
  ];

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
      },
      'PUT',
    );
    busy = '';
    if (!r.ok) return toast(r.data.error, 'error');
    await refreshSession();
    toast('Appearance saved. Everyone sees the new look.');
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
    toast(kind === 'logo' ? 'Logo uploaded.' : 'Background uploaded.');
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
    toast(kind === 'logo' ? 'Logo removed.' : 'Background image removed.');
  }

  function resetAll() {
    if (!confirm('Go back to the default look? Uploaded images stay until you remove them.')) return;
    Object.assign(draft, { site_name: '', page_title: '', accent: '', font: '', bg_type: 'default', overlay: 50 });
    save();
  }

  const colorInput = 'h-9 w-12 shrink-0 cursor-pointer rounded-lg border border-zinc-700 bg-zinc-950 p-1';
  const uploadBtn = `${btnGhost} cursor-pointer`;
</script>

<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
  <div>
    <h2 class="text-lg font-semibold tracking-tight">Appearance</h2>
    <p class="text-sm text-zinc-400">Changes show right away as a preview. They go live for everyone when you save.</p>
  </div>
  <div class="flex gap-2">
    <button class={btnGhost} onclick={resetAll} disabled={busy !== ''}>Reset</button>
    <button class={btnPrimary} onclick={save} disabled={busy !== ''}>{busy === 'save' ? 'Saving…' : 'Save'}</button>
  </div>
</div>

<div class="grid gap-6 lg:grid-cols-[1fr_20rem]">
  <div class="space-y-6">
    <section class="{card} p-5 sm:p-6">
      <h3 class="font-semibold">Logo and name</h3>
      <div class="mt-4 flex flex-wrap items-center gap-4">
        <div class="grid h-24 w-40 shrink-0 place-items-center rounded-lg border border-dashed border-zinc-700 bg-zinc-950/60 p-3">
          <Logo appearance={preview} />
        </div>
        <div class="space-y-2">
          <div class="flex flex-wrap gap-2">
            <label class={uploadBtn}>
              {busy === 'logo' ? 'Uploading…' : draft.logo_url ? 'Replace logo' : 'Upload logo'}
              <input type="file" class="sr-only" accept="image/png,image/jpeg,image/webp,image/gif,image/svg+xml" onchange={(e) => upload('logo', e)} disabled={busy !== ''} />
            </label>
            {#if draft.logo_url}
              <button class={btnDanger} onclick={() => remove('logo')} disabled={busy !== ''}>Remove</button>
            {/if}
          </div>
          <p class="text-xs text-zinc-500">PNG, JPG, WebP, GIF or SVG, max 1 MB. A transparent PNG or SVG looks best. Also used as the browser tab icon.</p>
        </div>
      </div>

      <div class="mt-5 grid gap-4 sm:grid-cols-2">
        <div>
          <label class={label} for="site-name">Site name</label>
          <input id="site-name" class={input} bind:value={draft.site_name} maxlength="40" placeholder={baseDomain} />
          <p class="mt-1.5 text-xs text-zinc-500">Shown under the logo.</p>
        </div>
        <div>
          <label class={label} for="page-title">Browser tab title</label>
          <input id="page-title" class={input} bind:value={draft.page_title} maxlength="60" placeholder={preview.site_name} />
        </div>
      </div>
    </section>

    <section class="{card} p-5 sm:p-6">
      <h3 class="font-semibold">Color and font</h3>
      <div class="mt-4">
        <span class={label}>Accent color</span>
        <div class="flex flex-wrap items-center gap-2">
          {#each accentPresets as c}
            <button
              type="button"
              class="size-8 rounded-full ring-2 ring-offset-2 ring-offset-zinc-900 transition {(draft.accent || defaultAccent).toLowerCase() === c ? 'ring-white' : 'ring-transparent hover:ring-zinc-600'}"
              style="background:{c}"
              aria-label="Accent {c}"
              onclick={() => (draft.accent = c === defaultAccent ? '' : c)}
            ></button>
          {/each}
          <span class="mx-1 h-6 w-px bg-zinc-700"></span>
          <input type="color" class={colorInput} value={isHex(draft.accent) ? draft.accent : defaultAccent} oninput={(e) => (draft.accent = e.currentTarget.value)} aria-label="Custom accent color" />
          <input class="{input} !w-28 font-mono" bind:value={draft.accent} placeholder={defaultAccent} maxlength="7" aria-label="Accent color code" />
        </div>
        <p class="mt-1.5 text-xs text-zinc-500">Used for buttons, links and highlights.</p>
      </div>

      <div class="mt-5">
        <label class={label} for="font">Font</label>
        <select id="font" bind:value={draft.font} class="{input} sm:w-64">
          {#each fonts as f}
            <option value={f.id}>{f.name}</option>
          {/each}
        </select>
        <p class="mt-2 text-sm text-zinc-300" style="font-family:{fonts.find((f) => f.id === draft.font)?.family}">
          The quick brown fox jumps over the lazy dog. 0123456789
        </p>
      </div>
    </section>

    <section class="{card} p-5 sm:p-6">
      <h3 class="font-semibold">Background</h3>
      <div class="mt-4 inline-flex flex-wrap gap-1 rounded-lg border border-zinc-800 bg-zinc-950/60 p-1" role="radiogroup" aria-label="Background type">
        {#each bgTypes as t}
          <button
            type="button"
            role="radio"
            aria-checked={draft.bg_type === t.id}
            class="rounded-md px-3 py-1.5 text-sm font-medium transition {draft.bg_type === t.id ? 'bg-indigo-500 text-white' : 'text-zinc-400 hover:text-zinc-200'}"
            onclick={() => (draft.bg_type = t.id)}
          >
            {t.name}
          </button>
        {/each}
      </div>

      {#if draft.bg_type === 'default'}
        <p class="mt-4 text-sm text-zinc-400">The built-in dark background with a soft glow in your accent color.</p>
      {:else if draft.bg_type === 'color' || draft.bg_type === 'gradient'}
        <div class="mt-4 flex flex-wrap items-end gap-4">
          <div>
            <span class={label}>{draft.bg_type === 'gradient' ? 'From' : 'Color'}</span>
            <div class="flex items-center gap-2">
              <input type="color" class={colorInput} bind:value={draft.bg_color} aria-label="Background color" />
              <input class="{input} !w-28 font-mono" bind:value={draft.bg_color} maxlength="7" aria-label="Background color code" />
            </div>
          </div>
          {#if draft.bg_type === 'gradient'}
            <div>
              <span class={label}>To</span>
              <div class="flex items-center gap-2">
                <input type="color" class={colorInput} bind:value={draft.bg_color2} aria-label="Second background color" />
                <input class="{input} !w-28 font-mono" bind:value={draft.bg_color2} maxlength="7" aria-label="Second background color code" />
              </div>
            </div>
          {/if}
        </div>
        <p class="mt-3 text-xs text-zinc-500">Dark colors work best: the text on the pages is light.</p>
      {:else}
        <div class="mt-4 flex flex-wrap items-center gap-2">
          <label class={uploadBtn}>
            {busy === 'background' ? 'Uploading…' : draft.bg_image_url ? 'Replace image' : 'Upload image'}
            <input type="file" class="sr-only" accept="image/png,image/jpeg,image/webp" onchange={(e) => upload('background', e)} disabled={busy !== ''} />
          </label>
          {#if draft.bg_image_url}
            <button class={btnDanger} onclick={() => remove('background')} disabled={busy !== ''}>Remove</button>
          {/if}
        </div>
        <p class="mt-2 text-xs text-zinc-500">
          PNG, JPG or WebP, max 8 MB. A wide image of about 1920×1080 works well; on phones it's cropped around the center to fill the screen.
        </p>
        {#if draft.bg_image_url}
          <div class="mt-4">
            <label class={label} for="overlay">Darken image: {draft.overlay}%</label>
            <input id="overlay" type="range" min="0" max="90" step="5" bind:value={draft.overlay} class="w-full accent-indigo-500 sm:w-80" />
            <p class="mt-1 text-xs text-zinc-500">Darker makes the text easier to read.</p>
          </div>
        {:else}
          <p class="mt-3 text-sm text-amber-300/90">No image yet. Upload one, or the default background is used.</p>
        {/if}
      {/if}
    </section>
  </div>

  <!-- Small copy of the sign-in page, so the result is visible without leaving the panel. -->
  <aside class="lg:sticky lg:top-32 lg:self-start">
    <p class="mb-2 text-xs font-semibold uppercase tracking-wider text-zinc-500">Sign-in page preview</p>
    <div class="relative overflow-hidden rounded-xl border border-zinc-800" style="font-family:{fonts.find((f) => f.id === draft.font)?.family}">
      <Backdrop appearance={preview} fixed={false} />
      <div class="relative flex flex-col items-center px-6 py-8">
        <Logo appearance={preview} />
        <p class="mt-2 text-xs font-medium text-zinc-300 [text-shadow:0_1px_6px_rgb(0_0_0/0.7)]">{preview.site_name}</p>
        <div class="mt-5 w-full rounded-xl border border-zinc-800 bg-zinc-900/80 p-4 backdrop-blur-md">
          <p class="text-sm font-semibold">Sign in</p>
          <div class="mt-3 h-7 rounded-md border border-zinc-700/80 bg-zinc-950/60"></div>
          <div class="mt-2 h-7 rounded-md border border-zinc-700/80 bg-zinc-950/60"></div>
          <div class="mt-3 grid h-8 place-items-center rounded-md bg-indigo-500 text-xs font-semibold text-white">Sign in</div>
          <p class="mt-3 border-t border-zinc-800 pt-3 text-center text-[11px] text-zinc-400">
            No account yet? <span class="font-medium text-indigo-400">Create one</span>
          </p>
        </div>
      </div>
    </div>
  </aside>
</div>
