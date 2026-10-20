// Accent colors. Every accent is light enough for dark text on top of it.
export const accents = {
  yellow: '#facc15',
  orange: '#fb923c',
  red: '#f87171',
  pink: '#f472b6',
  violet: '#a78bfa',
  blue: '#60a5fa',
  cyan: '#22d3ee',
  green: '#4ade80',
}

export const loaders = {
  vanilla: { name: 'Vanilla', color: '#a3a3a3' },
  fabric: { name: 'Fabric', color: '#dbb98a' },
  quilt: { name: 'Quilt', color: '#c084fc' },
  forge: { name: 'Forge', color: '#f59e6b' },
  neoforge: { name: 'NeoForge', color: '#f97316' },
}

export function applyAccent(name) {
  const hex = accents[name] ?? accents.yellow
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  const root = document.documentElement.style
  root.setProperty('--accent', hex)
  root.setProperty('--accent-rgb', `${r} ${g} ${b}`)
}
