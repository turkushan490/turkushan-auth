<script>
  let { kind = 'error', children } = $props();

  const styles = {
    error: 'border-rose-500/30 bg-rose-500/10 text-rose-200',
    success: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-200',
    info: 'border-indigo-500/30 bg-indigo-500/10 text-indigo-200',
  };
</script>

<div role={kind === 'error' ? 'alert' : 'status'} class="rounded-lg border px-3.5 py-2.5 text-sm {styles[kind]}">
  {@render children()}
</div>
