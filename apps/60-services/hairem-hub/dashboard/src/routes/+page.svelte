<script lang="ts">
  let { data } = $props();

  const statusOrder = ['next', 'inbox', 'waiting', 'blocked', 'scheduled', 'someday', 'done', 'none'];
  const byStatus = (status: string) => data.gtd.filter((item) => (item.status || 'none') === status);
  const activeStatuses = statusOrder.filter((status) => byStatus(status).length > 0);
</script>

<svelte:head>
  <title>hAIrem Dashboard</title>
  <meta name="description" content="Cockpit hAIrem" />
</svelte:head>

<main class="shell">
  <section class="topbar panel">
    <div class="brand"><span class="mark"></span><div><strong>hAIrem</strong><small>cockpit</small></div></div>
    <div class="ledline">
      {#each data.runtime.components as c}
        <span class="ledchip" aria-label={c.detail}>
          <i class:ok={c.led === 'ok'} class:warn={c.led === 'warn'} class:down={c.led === 'down'}></i>{c.label}
          <span class="tooltip" role="tooltip">
            <b>{c.label}</b>
            {#each c.detailLines as line}
              {#if line === ''}
                <em></em>
              {:else}
                <code>{line}</code>
              {/if}
            {/each}
          </span>
        </span>
      {/each}
    </div>
    <nav><a href={data.links.llmwiki}>llmwiki</a><a href={data.links.gtdwiki}>gtdwiki</a></nav>
  </section>

  <section class="grid metrics">
    <article class="metric panel"><span>hindsight ops</span><strong>{data.runtime.hindsight.backlog.processing}</strong><small>pending {data.runtime.hindsight.backlog.pending} · failed {data.runtime.hindsight.backlog.failed} · stuck {data.runtime.hindsight.backlog.stuck}</small></article>
    <article class="metric panel"><span>llm err 1h</span><strong>{data.runtime.hindsight.llm.errors1h}</strong><small>24h {data.runtime.hindsight.llm.errors24h} · 7d {data.runtime.hindsight.llm.errors} · last {data.runtime.hindsight.llm.lastErrorAge}</small></article>
    <article class="metric panel"><span>llm endpoints</span><strong>{data.runtime.llms.filter((l) => l.led !== 'down').length}/{data.runtime.llms.length}</strong><small>{data.runtime.llms.map((l) => `${l.id}:${l.loadedCount}/${l.modelCount}`).join(' · ')}</small></article>
    <article class="metric panel"><span>wikis</span><strong>{data.counts.llmTotal}/{data.counts.gtdTotal}</strong><small>llmwiki/gtdwiki pages</small></article>
  </section>

  <section class="grid two">
    <article class="panel compact">
      <h2>GTD</h2>
      <div class="lanes">
        {#each activeStatuses as status}
          <div class="lane"><h3>{status}<b>{byStatus(status).length}</b></h3>{#each byStatus(status).slice(0, 5) as item}<article class="card"><strong>{item.title}</strong><code>{item.path}</code></article>{/each}</div>
        {/each}
      </div>
    </article>
    <article class="panel compact">
      <h2>LLMWiki</h2>
      <div class="docs">{#each data.llm.slice(0, 8) as item}<article class="doc"><strong>{item.title}</strong><code>{item.path}</code></article>{/each}</div>
    </article>
  </section>
</main>

<style>
  :global(body){margin:0;background:#05070b;color:#dff8ff;font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;font-size:14px}
  :global(a){color:#68e7ff;text-decoration:none}.shell{max-width:1500px;margin:0 auto;padding:14px}.panel{background:rgba(8,15,25,.82);border:1px solid rgba(100,220,255,.16);border-radius:14px;box-shadow:0 12px 40px rgba(0,0,0,.28)}
  .topbar{height:42px;padding:0 12px;display:flex;align-items:center;gap:18px;position:sticky;top:0;z-index:3;backdrop-filter:blur(10px);overflow:visible}.brand{display:flex;align-items:center;gap:9px;min-width:116px}.brand strong{font-size:15px}.brand small{display:block;color:#7fa5b8;font-size:10px;letter-spacing:.16em;text-transform:uppercase}.mark{width:13px;height:13px;border-radius:50%;background:#ff174d;box-shadow:0 0 18px #ff174d}.ledline{display:flex;gap:8px;flex:1;overflow:visible}.ledchip{white-space:nowrap;color:#a9c4d2;font-size:12px;display:flex;align-items:center;gap:6px;position:relative;height:42px}.ledchip:hover,.ledchip:focus-within{color:#effcff}.tooltip{position:absolute;top:34px;left:0;z-index:20;min-width:310px;max-width:min(560px,calc(100vw - 36px));display:none;white-space:normal;padding:10px 12px;border-radius:12px;background:linear-gradient(180deg,rgba(6,14,23,.98),rgba(3,7,13,.98));border:1px solid rgba(104,231,255,.28);box-shadow:0 18px 48px rgba(0,0,0,.55),0 0 22px rgba(40,255,156,.08);color:#dff8ff}.tooltip b{display:block;margin-bottom:7px;color:#9beafe;text-transform:uppercase;letter-spacing:.08em;font-size:11px}.tooltip code{display:block;font:11px/1.45 ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;color:#cdefff;white-space:pre-wrap}.tooltip em{display:block;height:7px}.ledchip:hover .tooltip,.ledchip:focus-within .tooltip{display:block}nav{display:flex;gap:10px;font-size:12px}
  i{display:inline-block;width:9px;height:9px;border-radius:50%;background:#607080;box-shadow:0 0 7px #607080}.ok{background:#28ff9c;box-shadow:0 0 10px #28ff9c}.warn{background:#ffd447;box-shadow:0 0 10px #ffd447}.down{background:#ff3f6e;box-shadow:0 0 10px #ff3f6e}.grid{display:grid;gap:12px;margin-top:12px}.metrics{grid-template-columns:repeat(4,minmax(0,1fr))}.metric{padding:12px}.metric span,small{color:#8aaec0}.metric strong{display:block;color:#fff;font-size:28px;line-height:1.1;margin:5px 0}.two{grid-template-columns:1.4fr .6fr}.compact{padding:14px}h2{font-size:15px;margin:0 0 12px;text-transform:uppercase;letter-spacing:.08em;color:#9beafe}.docs{display:grid;gap:7px}.card code,.doc code{color:#75d9ff;font-size:11px;word-break:break-word}.lanes{display:grid;grid-template-columns:repeat(auto-fit,minmax(170px,1fr));gap:10px}.lane{padding:10px;border-radius:12px;background:rgba(0,0,0,.24)}.lane h3{margin:0 0 8px;color:#67e8f9;font-size:12px;text-transform:uppercase;letter-spacing:.08em}.lane h3 b{float:right;color:#ff5d85}.card,.doc{padding:9px;margin-bottom:8px;border-radius:10px;background:rgba(16,31,48,.75);border:1px solid rgba(110,231,255,.1)}.card strong,.doc strong{display:block;color:#fff;font-size:13px;margin-bottom:4px}
  @media(max-width:1000px){.metrics,.two{grid-template-columns:1fr}.topbar{height:auto;min-height:42px;flex-wrap:wrap;padding:8px 12px}.ledline{order:3;flex-basis:100%;flex-wrap:wrap}}
</style>
