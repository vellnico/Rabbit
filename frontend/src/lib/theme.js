// Accent colors, named after Minecraft dyes. Yellow is the lemon of the
// design tokens; the others replace the --accent-* tokens at runtime.
export const accents = {
  yellow: '#F3E046',
  orange: '#f9801d',
  red: '#d6453c',
  pink: '#f38baa',
  violet: '#8932b8',
  blue: '#3ab3da',
  cyan: '#169c9c',
  green: '#80c71f',
}

// Loader names and the colour of their dot in badges.
export const loaders = {
  vanilla: { name: 'Vanilla', color: 'var(--loader-vanilla)' },
  fabric: { name: 'Fabric', color: 'var(--loader-fabric)' },
  quilt: { name: 'Quilt', color: 'var(--loader-quilt)' },
  forge: { name: 'Forge', color: 'var(--loader-forge)' },
  neoforge: { name: 'NeoForge', color: 'var(--loader-neoforge)' },
}

function luminance(r, g, b) {
  const f = (c) => {
    c /= 255
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
  }
  return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b)
}

const tokens = ['200', '300', '400', '500', '600', 'a08', 'a14', 'a24', 'glow', 'glow-hover']

export function applyAccent(name) {
  const root = document.documentElement.style
  for (const k of tokens) root.removeProperty('--accent-' + k)
  root.removeProperty('--text-on-accent')
  const hex = accents[name]
  if (!hex || name === 'yellow') return

  const mix = (other, pct) => `color-mix(in srgb, ${hex} ${pct}%, ${other})`
  const alpha = (pct) => mix('transparent', pct)
  root.setProperty('--accent-200', mix('#ffffff', 45))
  root.setProperty('--accent-300', mix('#ffffff', 75))
  root.setProperty('--accent-400', hex)
  root.setProperty('--accent-500', mix('#000000', 88))
  root.setProperty('--accent-600', mix('#000000', 70))
  root.setProperty('--accent-a08', alpha(8))
  root.setProperty('--accent-a14', alpha(14))
  root.setProperty('--accent-a24', alpha(24))
  root.setProperty('--accent-glow', `0 0 0 1px ${alpha(35)}, 0 8px 32px -8px ${alpha(35)}`)
  root.setProperty('--accent-glow-hover', `0 0 0 1px ${alpha(50)}, 0 8px 36px -8px ${alpha(50)}`)

  const L = luminance(parseInt(hex.slice(1, 3), 16), parseInt(hex.slice(3, 5), 16), parseInt(hex.slice(5, 7), 16))
  // Dark text unless the accent is too dark for it.
  if (L < 0.18) root.setProperty('--text-on-accent', 'var(--text-primary)')
}
