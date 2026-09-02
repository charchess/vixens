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
        <span class="ledchip" title={c.detail}><i class:ok={c.led === 'ok'} class:warn={c.led === 'warn'} class:down={c.led === 'down'}></i>{c.label}</span>
      {/each}
    </div>
    <nav><a href={data.links.llmwiki}>llmwiki</a><a href={data.links.gtdwiki}>gtdwiki</a></nav>
  </section>

  <section class="grid metrics">
    <article class="metric panel"><span>hindsight pending</span><strong>{data.runtime.hindsight.backlog.pending}</strong><small>failed {data.runtime.hindsight.backlog.failed} · processing {data.runtime.hindsight.backlog.processing}</small></article>
    <article class="metric panel"><span>llm req 7j</span><strong>{data.runtime.hindsight.llm.errors}</strong><small>{data.runtime.hindsight.llm.success} ok · {data.runtime.hindsight.llm.total} total</small></article>
    <article class="metric panel"><span>ollama</span><strong>{data.runtime.llm.loadedCount}/{data.runtime.llm.modelCount}</strong><small>{data.runtime.llm.version || 'down'}</small></article>
    <article class="metric panel"><span>wikis</span><strong>{data.counts.llmTotal}/{data.counts.gtdTotal}</strong><small>llmwiki/gtdwiki pages</small></article>
  </section>

  <section class="grid main">
    <article class="panel statuspanel">
      <h2>États composants</h2>
      <div class="components">
        {#each data.runtime.components as c}
          <div class="row"><span><i class:ok={c.led === 'ok'} class:warn={c.led === 'warn'} class:down={c.led === 'down'}></i>{c.label}</span><code>{c.detail}</code></div>
        {/each}
      </div>
    </article>

    <article class="panel statuspanel">
      <h2>Hindsight mémoire</h2>
      <div class="banks">
        {#each data.runtime.hindsight.banks as b}
          <div class="bank">
            <header><strong>{b.name || b.bank_id}</strong><code>{b.bank_id}</code></header>
            <div class="bankgrid">
              <span><b>{b.stats?.total_nodes || b.fact_count || 0}</b> facts</span>
              <span><b>{b.stats?.pending_consolidation || 0}</b> pending</span>
              <span><b>{b.stats?.failed_consolidation || 0}</b> failed</span>
              <span><b>{b.llm.errors}</b> llm err</span>
            </div>
          </div>
        {/each}
      </div>
    </article>

    <article class="panel statuspanel">
      <h2>LLM</h2>
      <div class="models">
        {#if data.runtime.llm.loaded.length === 0}<p class="muted">Aucun modèle chargé.</p>{/if}
        {#each data.runtime.llm.loaded as m}<div class="row hot"><span><i class="ok"></i>{m.name}</span><code>loaded</code></div>{/each}
        {#each data.runtime.llm.models as m}<div class="row"><span>{m.name}</span><code>{m.size || ''} {m.quant || ''}</code></div>{/each}
      </div>
    </article>
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
  .topbar{height:42px;padding:0 12px;display:flex;align-items:center;gap:18px;position:sticky;top:0;z-index:3;backdrop-filter:blur(10px)}.brand{display:flex;align-items:center;gap:9px;min-width:116px}.brand strong{font-size:15px}.brand small{display:block;color:#7fa5b8;font-size:10px;letter-spacing:.16em;text-transform:uppercase}.mark{width:13px;height:13px;border-radius:50%;background:#ff174d;box-shadow:0 0 18px #ff174d}.ledline{display:flex;gap:8px;flex:1;overflow:hidden}.ledchip{white-space:nowrap;color:#a9c4d2;font-size:12px;display:flex;align-items:center;gap:6px}nav{display:flex;gap:10px;font-size:12px}
  i{display:inline-block;width:9px;height:9px;border-radius:50%;background:#607080;box-shadow:0 0 7px #607080}.ok{background:#28ff9c;box-shadow:0 0 10px #28ff9c}.warn{background:#ffd447;box-shadow:0 0 10px #ffd447}.down{background:#ff3f6e;box-shadow:0 0 10px #ff3f6e}.grid{display:grid;gap:12px;margin-top:12px}.metrics{grid-template-columns:repeat(4,minmax(0,1fr))}.metric{padding:12px}.metric span,.muted,small{color:#8aaec0}.metric strong{display:block;color:#fff;font-size:28px;line-height:1.1;margin:5px 0}.main{grid-template-columns:1fr 1.2fr 1fr}.two{grid-template-columns:1.4fr .6fr}.statuspanel,.compact{padding:14px}h2{font-size:15px;margin:0 0 12px;text-transform:uppercase;letter-spacing:.08em;color:#9beafe}.components,.models,.docs{display:grid;gap:7px}.row{display:flex;justify-content:space-between;gap:12px;align-items:center;padding:8px 10px;background:rgba(0,0,0,.22);border-radius:10px}.row span{display:flex;gap:8px;align-items:center}.row code,.card code,.doc code,.bank code{color:#75d9ff;font-size:11px;word-break:break-word}.hot{border:1px solid rgba(40,255,156,.22)}.banks{display:grid;gap:9px}.bank{padding:10px;background:rgba(0,0,0,.22);border-radius:12px}.bank header{display:flex;justify-content:space-between;gap:10px;margin-bottom:8px}.bankgrid{display:grid;grid-template-columns:repeat(4,1fr);gap:8px}.bankgrid span{color:#9fb9c8}.bankgrid b{color:#fff}.lanes{display:grid;grid-template-columns:repeat(auto-fit,minmax(170px,1fr));gap:10px}.lane{padding:10px;border-radius:12px;background:rgba(0,0,0,.24)}.lane h3{margin:0 0 8px;color:#67e8f9;font-size:12px;text-transform:uppercase;letter-spacing:.08em}.lane h3 b{float:right;color:#ff5d85}.card,.doc{padding:9px;margin-bottom:8px;border-radius:10px;background:rgba(16,31,48,.75);border:1px solid rgba(110,231,255,.1)}.card strong,.doc strong{display:block;color:#fff;font-size:13px;margin-bottom:4px}
  @media(max-width:1000px){.metrics,.main,.two{grid-template-columns:1fr}.topbar{height:auto;min-height:42px;flex-wrap:wrap;padding:8px 12px}.ledline{order:3;flex-basis:100%;flex-wrap:wrap}}
</style>
