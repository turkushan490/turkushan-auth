<script>
  import { tick } from 'svelte';

  let { open = $bindable(false), title, children } = $props();

  let dialog = $state();
  const titleId = 'm-' + Math.random().toString(36).slice(2, 9);
  const focusable = 'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

  // On open: remember what had focus and move into the dialog. On close: go back there.
  $effect(() => {
    if (!open) return;
    const before = document.activeElement;
    tick().then(() => {
      const first = dialog?.querySelector('[data-autofocus]') || dialog?.querySelector(focusable);
      (first || dialog)?.focus();
    });
    return () => {
      if (before instanceof HTMLElement && document.contains(before)) before.focus();
    };
  });

  function onkeydown(e) {
    if (!open) return;
    if (e.key === 'Escape') {
      open = false;
      return;
    }
    // Keep Tab inside the dialog.
    if (e.key === 'Tab' && dialog) {
      const items = [...dialog.querySelectorAll(focusable)];
      if (items.length === 0) {
        e.preventDefault();
        return;
      }
      const first = items[0];
      const last = items[items.length - 1];
      if (e.shiftKey && (document.activeElement === first || !dialog.contains(document.activeElement))) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && (document.activeElement === last || !dialog.contains(document.activeElement))) {
        e.preventDefault();
        first.focus();
      }
    }
  }
</script>

<svelte:window {onkeydown} />

{#if open}
  <div class="fixed inset-0 z-50 grid place-items-center p-4">
    <!-- Clicking outside closes; keyboard users have Esc. -->
    <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" aria-hidden="true" onclick={() => (open = false)}></div>
    <div
      bind:this={dialog}
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      tabindex="-1"
      class="relative max-h-[90vh] w-full max-w-md overflow-y-auto rounded-2xl border border-zinc-800 bg-zinc-900 p-6 shadow-2xl shadow-black/60 outline-none"
    >
      <h2 id={titleId} class="text-lg font-semibold tracking-tight">{title}</h2>
      <div class="mt-4">
        {@render children()}
      </div>
    </div>
  </div>
{/if}
