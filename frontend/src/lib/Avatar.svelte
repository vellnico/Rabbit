<script>
  // Head of the Ely.by skin in an inventory slot, or the nick's initial.
  let { account, size = 32 } = $props()
  let failed = $state(false)
  let showSkin = $derived(account && (account.kind === 'elyby' || account.elySkin) && !failed)
  let url = $derived(account ? `https://skinsystem.ely.by/skins/${encodeURIComponent(account.name)}.png` : '')
  let inner = $derived(size - 4)
</script>

<div class="slot frame" style="width:{size}px;height:{size}px">
  {#if showSkin}
    <div
      class="px head"
      style="width:{inner}px;height:{inner}px;background-image:url({url});background-size:{inner * 8}px auto;background-position:-{inner}px -{inner}px"
    >
      <img src={url} alt="" onerror={() => (failed = true)} hidden />
    </div>
  {:else}
    <span class="letter" style="font-size:{size * 0.5}px">{account ? account.name[0].toUpperCase() : '?'}</span>
  {/if}
</div>

<style>
  .frame { flex: none; display: grid; place-items: center; }
  .head { background-repeat: no-repeat; }
  .letter { font-family: var(--pixel); color: #fff; text-shadow: 2px 2px 0 #3f3f3f; line-height: 1; }
</style>
