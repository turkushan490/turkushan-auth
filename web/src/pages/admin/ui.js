// Shared class strings for the admin pages, so buttons and cards look the same everywhere.
export const card = 'rounded-xl border border-zinc-800 bg-zinc-900/80';
export const btn =
  'inline-flex items-center justify-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-50';
export const btnPrimary = `${btn} bg-indigo-500 text-white hover:bg-indigo-400`;
export const btnApprove = `${btn} bg-emerald-500/15 text-emerald-300 ring-1 ring-emerald-500/25 hover:bg-emerald-500/25`;
export const btnDanger = `${btn} bg-rose-500/10 text-rose-300 ring-1 ring-rose-500/20 hover:bg-rose-500/20`;
export const btnGhost = `${btn} text-zinc-300 ring-1 ring-zinc-700 hover:bg-zinc-800`;
export const input =
  'block w-full rounded-lg border border-zinc-700/80 bg-zinc-950/60 px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 outline-none transition focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/30';
export const label = 'mb-1.5 block text-sm font-medium text-zinc-300';

export const statusBadge = {
  pending: 'bg-amber-500/15 text-amber-300 ring-amber-500/25',
  approved: 'bg-emerald-500/15 text-emerald-300 ring-emerald-500/25',
  denied: 'bg-rose-500/15 text-rose-300 ring-rose-500/25',
};
export const badge = 'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ring-1';
