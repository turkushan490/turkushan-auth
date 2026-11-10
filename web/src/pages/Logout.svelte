<script>
  import { api } from '../lib/api.js';
  import { app, navigate, refreshSession } from '../lib/state.svelte.js';
  import { t } from '../lib/i18n.svelte.js';

  $effect(() => {
    (async () => {
      const r = await api('/logout', {});
      await refreshSession();
      app.flash = r.ok ? () => t('logout.done') : r.data.error;
      navigate('/login');
    })();
  });
</script>

<div class="grid min-h-screen place-items-center text-sm text-zinc-400">{t('logout.signing_out')}</div>
