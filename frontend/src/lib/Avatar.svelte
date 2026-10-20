<script>
  // Head of the Ely.by skin, or the nick's initial.
  let { account, size = 28 } = $props()
  let failed = $state(false)
  let showSkin = $derived(account && (account.kind === 'elyby' || account.elySkin) && !failed)
  let url = $derived(account ? `https://skinsystem.ely.by/skins/${encodeURIComponent(account.name)}.png` : '')
</script>

{#if showSkin}
  <div
    class="avatar skin"
    style="width:{size}px;height:{size}px;background-image:url({url});background-size:{size * 8}px auto;background-position:-{size}px -{size}px"
  >
    <img src={url} alt="" onerror={() => (failed = true)} hidden />
  </div>
{:else}
  <div class="avatar letter" style="width:{size}px;height:{size}px;font-size:{size * 0.45}px">
    {account ? account.name[0].toUpperCase() : '?'}
  </div>
{/if}

<style>
  .avatar { flex: none; border-radius: 4px; }
  .skin { image-rendering: pixelated; background-repeat: no-repeat; background-color: var(--selected); }
  .letter { display: grid; place-items: center; background: var(--selected); color: var(--muted); font-weight: 600; }
</style>
