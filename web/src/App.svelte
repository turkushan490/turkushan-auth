<script>
  // Milestone 1 placeholder: shows that the portal and its database are up.
  // Login, register and the admin panel replace this in later milestones.
  let health = $state('checking');

  $effect(() => {
    fetch('/healthz')
      .then((r) => (health = r.ok ? 'ok' : 'down'))
      .catch(() => (health = 'down'));
  });
</script>

<main class="grid min-h-screen place-items-center bg-zinc-950 p-4 text-zinc-100">
  <div class="w-full max-w-sm rounded-2xl border border-zinc-800 bg-zinc-900/70 p-8 shadow-2xl shadow-black/50">
    <div class="flex items-center gap-3">
      <div class="grid size-10 place-items-center rounded-xl bg-indigo-500/15 text-indigo-400">
        <svg viewBox="0 0 24 24" class="size-5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <rect x="4" y="11" width="16" height="10" rx="2" />
          <path d="M8 11V7a4 4 0 0 1 8 0v4" />
        </svg>
      </div>
      <div>
        <h1 class="text-lg font-semibold tracking-tight">turkushan.com</h1>
        <p class="text-sm text-zinc-400">Login portal</p>
      </div>
    </div>

    <div class="mt-6 flex items-center justify-between rounded-lg border border-zinc-800 bg-zinc-950/60 px-4 py-3 text-sm">
      <span class="text-zinc-400">Status</span>
      <span class="flex items-center gap-2 font-medium">
        <span
          class="size-2 rounded-full"
          class:bg-emerald-400={health === 'ok'}
          class:bg-rose-400={health === 'down'}
          class:bg-zinc-500={health === 'checking'}
        ></span>
        {health === 'ok' ? 'Running' : health === 'down' ? 'Unavailable' : 'Checking…'}
      </span>
    </div>
  </div>
</main>
