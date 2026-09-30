export function timeAgo(unix) {
  if (!unix) return 'never';
  const s = Math.floor(Date.now() / 1000) - unix;
  if (s < 60) return 'just now';
  const m = Math.floor(s / 60);
  if (m < 60) return `${m} min ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h} h ago`;
  const d = Math.floor(h / 24);
  if (d < 30) return `${d} d ago`;
  return new Date(unix * 1000).toLocaleDateString();
}

export function fullDate(unix) {
  return unix ? new Date(unix * 1000).toLocaleString() : '';
}
