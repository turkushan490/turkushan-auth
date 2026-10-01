// The look set in Admin panel → Appearance, applied to the whole app.

const fallback = 'ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, sans-serif';

// ids match allowedFonts in internal/server/appearance.go; the files are bundled (src/fonts.css).
export const fonts = [
  { id: '', name: 'System default', family: fallback },
  { id: 'inter', name: 'Inter', family: `"Inter Variable", ${fallback}` },
  { id: 'montserrat', name: 'Montserrat', family: `"Montserrat Variable", ${fallback}` },
  { id: 'space-grotesk', name: 'Space Grotesk', family: `"Space Grotesk Variable", ${fallback}` },
  { id: 'nunito', name: 'Nunito', family: `"Nunito Variable", ${fallback}` },
  { id: 'outfit', name: 'Outfit', family: `"Outfit Variable", ${fallback}` },
  { id: 'mono', name: 'JetBrains Mono', family: '"JetBrains Mono Variable", ui-monospace, monospace' },
];

export const defaultAccent = '#6366f1';
export const accentPresets = ['#6366f1', '#3b82f6', '#06b6d4', '#10b981', '#f59e0b', '#f43f5e', '#a855f7'];

export const isHex = (c) => /^#[0-9a-f]{6}$/i.test(c || '');

// Black or white, whichever reads better on the color.
function textOn(hex) {
  const [r, g, b] = [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255);
  const lum = 0.2126 * r + 0.7152 * g + 0.0722 * b;
  return lum > 0.6 ? '#18181b' : '#ffffff';
}

// The UI uses Tailwind's indigo scale as its accent; overriding those variables recolors everything.
export function applyAppearance(a) {
  const root = document.documentElement.style;
  if (isHex(a?.accent)) {
    root.setProperty('--color-indigo-500', a.accent);
    root.setProperty('--color-indigo-400', `color-mix(in oklab, ${a.accent} 82%, white)`);
    root.setProperty('--color-indigo-300', `color-mix(in oklab, ${a.accent} 60%, white)`);
    root.setProperty('--color-indigo-200', `color-mix(in oklab, ${a.accent} 35%, white)`);
    root.setProperty('--accent-fg', textOn(a.accent));
  } else {
    for (const v of ['--color-indigo-500', '--color-indigo-400', '--color-indigo-300', '--color-indigo-200', '--accent-fg']) {
      root.removeProperty(v);
    }
  }

  const font = fonts.find((f) => f.id === (a?.font || ''));
  if (font?.id) root.setProperty('--font-sans', font.family);
  else root.removeProperty('--font-sans');

  if (a?.page_title) document.title = a.page_title;

  const icon = document.querySelector('link[rel="icon"]');
  if (icon) {
    if (a?.logo_url) {
      icon.removeAttribute('type');
      icon.href = a.logo_url;
    } else if (!icon.href.endsWith('/favicon.svg')) {
      icon.type = 'image/svg+xml';
      icon.href = '/favicon.svg';
    }
  }
}

// CSS for the page background; '' means the built-in dark look.
export function backgroundStyle(a) {
  switch (a?.bg_type) {
    case 'color':
      return isHex(a.bg_color) ? `background:${a.bg_color}` : '';
    case 'gradient':
      return isHex(a.bg_color) && isHex(a.bg_color2) ? `background:linear-gradient(160deg, ${a.bg_color}, ${a.bg_color2})` : '';
    case 'image':
      return a.bg_image_url ? `background:#09090b url("${a.bg_image_url}") center / cover no-repeat` : '';
    default:
      return '';
  }
}
