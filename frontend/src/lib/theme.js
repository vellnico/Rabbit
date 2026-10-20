// Accent colors, named after Minecraft dyes. The text color on top of the
// accent (black or white) is picked from its luminance.
export const accents = {
  yellow: '#fed83d',
  orange: '#f9801d',
  red: '#d6453c',
  pink: '#f38baa',
  violet: '#8932b8',
  blue: '#3ab3da',
  cyan: '#169c9c',
  green: '#80c71f',
}

export const loaders = {
  vanilla: { name: 'Vanilla', color: '#5d8c3a' },
  fabric: { name: 'Fabric', color: '#b29467' },
  quilt: { name: 'Quilt', color: '#9b59b6' },
  forge: { name: 'Forge', color: '#c0662b' },
  neoforge: { name: 'NeoForge', color: '#d9541e' },
}

function luminance(r, g, b) {
  const f = (c) => {
    c /= 255
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
  }
  return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b)
}

export function applyAccent(name) {
  const hex = accents[name] ?? accents.yellow
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  const L = luminance(r, g, b)
  const root = document.documentElement.style
  root.setProperty('--accent', hex)
  // Contrast against near-black text vs white text.
  root.setProperty('--accent-ink', (L + 0.05) / 0.05 > 1.05 / (L + 0.05) ? '#1b1b1b' : '#ffffff')
}
