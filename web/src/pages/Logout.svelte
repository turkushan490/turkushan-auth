<script>
  import { api } from '../lib/api.js';
  import { app, navigate, refreshSession } from '../lib/state.svelte.js';

  $effect(() => {
    (async () => {
      const r = await api('/logout', {});
      await refreshSession();
      app.flash = r.ok ? "You're signed out on all sites." : r.data.error;
      navigate('/login');
    })();
  });
</script>

<div class="grid min-h-screen place-items-center bg-zinc-950 text-sm text-zinc-400">Signing out…</div>
