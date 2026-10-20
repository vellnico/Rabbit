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
  <div class="avatar letter" style="width:{size}px;height:{size}px;font-size:{size * 0.46}px">
    {account ? account.name[0].toUpperCase() : '?'}
  </div>
{/if}

<style>
  .avatar { border-radius: 8px; flex: none; box-shadow: 0 0 0 1px rgb(255 255 255 / .06); }
  .skin { image-rendering: pixelated; background-repeat: no-repeat; background-color: var(--raised); }
  .letter {
    display: grid; place-items: center; font-weight: 700;
    background: linear-gradient(135deg, rgb(var(--accent-rgb) / .35), rgb(var(--accent-rgb) / .08));
    color: var(--accent);
  }
</style>
