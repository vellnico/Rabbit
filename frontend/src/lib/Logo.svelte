<script>
  // The Rabbit mark: a 12×11 pixel sprite.
  let { size = 24 } = $props()

  const sprite = [
    '..OO....OO..',
    '.OWWO..OWWO.',
    '.OWPO..OPWO.',
    '.OWPO..OPWO.',
    '.OWWOOOOWWO.',
    'OWWWWWWWWWWO',
    'OWWKWWWWKWWO',
    'OWWWWPPWWWWO',
    'OSWWWWWWWWSO',
    '.OSSWWWWSSO.',
    '..OOOOOOOO..',
  ]
  const fill = { O: '#2b2b2b', W: '#f4f1ea', P: '#e79aa6', K: '#2b2b2b', S: '#cfc8bb' }
  const pixels = sprite.flatMap((row, y) => [...row].map((c, x) => ({ x, y, c })).filter((p) => fill[p.c]))
</script>

<svg viewBox="0 0 12 11" width={size} height={(size * 11) / 12} shape-rendering="crispEdges" aria-hidden="true">
  {#each pixels as p (p.x + ':' + p.y)}<rect x={p.x} y={p.y} width="1" height="1" fill={fill[p.c]} />{/each}
</svg>
