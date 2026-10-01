<script>
  import { backgroundStyle } from '../lib/appearance.js';

  // Fills the screen behind every page, or (fixed = false) the box it's placed in for a preview.
  let { appearance, fixed = true } = $props();

  const style = $derived(backgroundStyle(appearance));
  const overlay = $derived(appearance?.bg_type === 'image' && style ? Math.min(90, Math.max(0, appearance.overlay || 0)) / 100 : 0);
</script>

<div class="{fixed ? 'fixed -z-10' : 'absolute'} inset-0 overflow-hidden bg-zinc-950" aria-hidden="true">
  {#if style}
    <div class="absolute inset-0" {style}></div>
    {#if overlay > 0}
      <div class="absolute inset-0 bg-black" style="opacity:{overlay}"></div>
    {/if}
  {:else}
    <div class="absolute inset-x-0 top-0 h-96 bg-gradient-to-b from-indigo-500/10 via-indigo-500/5 to-transparent"></div>
  {/if}
</div>
