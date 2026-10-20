<script>
  // Shows the head of the Ely.by skin, or the first letter of the nick.
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
  <div class="avatar letter" style="width:{size}px;height:{size}px;font-size:{size * 0.5}px">
    {account ? account.name[0].toUpperCase() : '?'}
  </div>
{/if}

<style>
  .avatar { border-radius: 6px; flex: none; }
  .skin { image-rendering: pixelated; background-repeat: no-repeat; background-color: var(--raised); }
  .letter { display: grid; place-items: center; background: var(--raised); color: var(--muted); font-weight: 600; }
</style>
