
<script lang="ts">
  let { data } = $props();
  let page = $state('actions');
  let selected = $state(null);
  let query = $state('');
  let filter = $state('all');
  let projectQuery = $state('');
  let referenceQuery = $state('');
  let cart = $state([]);
  let activeProject = $state(null);

  const cockpit = $derived(data.gtdCockpit);
  const allProjects = $derived(cockpit.projects);
  const noNextProjects = $derived(cockpit.projects.filter((p) => p.noNextAction));
  const filterChips = $derived(['all','@courses','@téléphone','@ordinateur','@maison','@admin','@infra','5-15min','30min+','énergie basse','énergie haute','sensible','blocked', ...cockpit.peopleOptions.slice(0,10).map((p)=>`avec:${p}`), ...cockpit.contextOptions.slice(0,12), ...cockpit.tagOptions.slice(0,14)].filter((v,i,a)=>a.indexOf(v)===i));
  const filteredActions = $derived(cockpit.next.filter((a) => matches(a, filter, query)));
  const filteredProjects = $derived(allProjects.filter((p) => projectMatches(p, projectQuery)));
  const projectActions = $derived(activeProject ? cockpit.next.filter((a) => a.projectId === activeProject.id || a.project === activeProject.title) : []);
  const filteredReferences = $derived(cockpit.references.filter((r) => referenceMatches(r, referenceQuery)).slice(0,80));
  const refsForSelection = $derived(selected ? cockpit.references.filter((r) => r.projectId === selected.id || r.projectId === selected.projectId || r.actionId === selected.id).slice(0,12) : []);

  function hay(item){ return [item.title,item.status,item.priority,item.project,item.projectId,item.finish,item.criterion,item.guardrail,item.result,item.excerpt,...(item.contexts||[]),...(item.tags||[]),...(item.people||[])].filter(Boolean).join(' ').toLowerCase(); }
  function matches(item, f, q){ const text=hay(item); const qq=q.trim().toLowerCase(); if(qq && !text.includes(qq)) return false; if(f==='all') return true; if(f==='5-15min') return /5\s*[–-]\s*15|15\s*min|rapide|court|quart/.test(text); if(f==='30min+') return /30\s*min|45\s*min|1\s*h|long|profond|focus/.test(text); if(f==='énergie basse') return /énergie basse|energie basse|facile|léger|leger|fatigue/.test(text); if(f==='énergie haute') return /énergie haute|energie haute|complexe|profond|focus|concentration/.test(text); if(f==='sensible') return /finance|santé|sante|juridique|électricité|electricite|achat|publication|sensible/.test(text); if(f==='blocked') return /blocked|bloqué|bloque|waiting|attente/.test(text); if(f.startsWith('avec:')) return (item.people||[]).map((p)=>p.toLowerCase()).includes(f.slice(5).toLowerCase()) || text.includes(f.slice(5).toLowerCase()); return text.includes(f.replace(/^@/,'').toLowerCase()); }
  function projectMatches(project, q){ const qq=q.trim().toLowerCase(); if(!qq) return true; return hay(project).includes(qq); }
  function referenceMatches(ref, q){ const qq=q.trim().toLowerCase(); if(!qq) return true; return [ref.title,ref.excerpt,ref.projectId,ref.actionId,...(ref.tags||[])].filter(Boolean).join(' ').toLowerCase().includes(qq); }
  function meta(item){ return [item.priority, ...(item.people||[]).map((p)=>`avec ${p}`), item.dueAt?`due ${item.dueAt}`:null, item.followupAt?`relance ${item.followupAt}`:null, item.scheduledAt?`planifié ${item.scheduledAt}`:null].filter(Boolean).slice(0,5); }
  function addCart(item){ if(!cart.some((x)=>x.path===item.path)) cart=[...cart,item]; }
  function removeCart(item){ cart=cart.filter((x)=>x.path!==item.path); }
  function openDetail(item, kind='action'){ selected={...item,kind}; }
  function closeDetail(){ selected=null; }
  function openProject(project){ activeProject=project; openDetail(project,'project'); }
</script>

<svelte:head><title>hAIrem GTD cockpit</title></svelte:head>
<main class="shell">
  <section class="topbar panel">
    <div class="brand"><span class="heart"></span><b>hAIrem</b><small>GTD</small></div>
    <div class="leds">
      {#each data.runtime.components as c}
        <svelte:element this={c.href?'a':'span'} class="led" href={c.href} target={c.href?'_blank':undefined} rel="noreferrer" aria-label={c.aria || c.detail || c.label}>
          <i class:ok={c.led==='ok'} class:warn={c.led==='warn'} class:down={c.led==='down'}></i>{c.label}
          <span class="tip">
            <b>{c.label}</b>
            {#if c.primaryIssue}<code class="issue">{c.primaryIssue}</code>{/if}
            {#if c.tableRows?.length}
              <table><thead><tr><th>bank</th><th>facts</th><th>pend</th><th>run</th><th>stuck</th><th>fail</th><th>llm</th><th>last</th></tr></thead><tbody>{#each c.tableRows as r}<tr class:bad={r.stuck||r.failed||r.llm1h}><td>{r.bank}</td><td>{r.facts}</td><td>{r.pending}</td><td>{r.running}</td><td>{r.stuck}</td><td>{r.failed}</td><td>{r.llm1h}/{r.llm24h}</td><td>{r.last}</td></tr>{/each}</tbody></table>
            {:else}
              {#each c.detailLines as l}<code>{l}</code>{/each}
            {/if}
          </span>
        </svelte:element>
      {/each}
    </div>
    <nav><a href={data.links.gtdwiki} target="_blank" rel="noreferrer">GTD files</a><a href={data.links.llmwiki} target="_blank" rel="noreferrer">LLMWiki</a></nav>
  </section>

  <section class="authority"><b>Katia GTD</b><span>Dashboard = vue + sélection. Capture ≠ engagement.</span><code>{cockpit.schema}</code></section>

  <section class="tabs panel">
    {#each [['projects','Projets',allProjects.length],['actions','Actions',cockpit.next.length],['productivity','Productivité',cart.length],['review','Review',cockpit.review.length + noNextProjects.length]] as t}
      <button class:active={page===t[0]} onclick={()=>page=t[0]}>{t[1]} <b>{t[2]}</b></button>
    {/each}
  </section>

  {#if page==='actions'}
    <section class="work actions-view">
      <aside class="panel side"><h2>Filtres</h2><input bind:value={query} placeholder="personne, projet, contexte…" />{#each filterChips as chip}<button class:active={filter===chip} onclick={()=>filter=chip}>{chip}</button>{/each}</aside>
      <article class="panel main"><header><p>Choisir maintenant</p><h1>Actions</h1><span>Uniquement les next actions. {filteredActions.length}/{cockpit.next.length} visibles · panier {cart.length}. Filtrage par contexte, personne, énergie, temps, projet.</span></header><div class="cards">{#each filteredActions as a}<article class="card action"><button class="cardopen" onclick={()=>openDetail(a,'action')}><div class="row"><strong>{a.title}</strong><small>{a.priority||'Px'} · {(a.contexts||[]).join(', ')||'sans contexte'}</small></div><div class="chips meta">{#each meta(a) as m}<em>{m}</em>{/each}</div><div class="chips">{#each [...(a.contexts||[]),...(a.tags||[])] as tag}<em>{tag}</em>{/each}</div>{#if a.project || a.projectId}<p>Projet : {a.project || a.projectId}</p>{/if}{#if a.finish}<p>Fini quand : {a.finish}</p>{/if}</button><footer><span>{a.updatedAt||'n/a'}</span><span class="footactions"><button type="button" onclick={()=>openDetail(a,'action')}>éditer/détail</button><button type="button" class="addcart" onclick={()=>addCart(a)}>+ panier</button></span></footer></article>{/each}</div></article>
    </section>
  {/if}

  {#if page==='projects'}
    <section class="panel page"><header><p>Vision projets</p><h1>Projets</h1><span>Tous les projets, avec deadline, statut, tags, compteurs et next action liée.</span></header><input class="wide-search" bind:value={projectQuery} placeholder="chercher projet, domaine, tag, deadline…"/><div class="cards projects">{#each filteredProjects as p}<button class="card project" class:warncard={p.noNextAction} onclick={()=>openProject(p)}><strong>{p.title}</strong><div class="chips">{#each p.tags as tag}<em>{tag}</em>{/each}</div><p>{p.result||p.excerpt}</p><code>Next: {p.nextAction||'aucune next action liée'}</code><footer><span>{p.status} · due {p.dueAt||'n/a'}</span><span>{p.linkedActions} actions · {p.references} refs</span></footer></button>{/each}</div></section>
  {/if}

  {#if page==='productivity'}
    <section class="panel page"><header><p>Activation</p><h1>Productivité</h1><span>Panier de travail choisi consciemment + capture inbox visible. Aucune mutation réelle automatique.</span></header><div class="split"><section><h2>Panier</h2>{#if cart.length}<div class="cards compact">{#each cart as a}<article class="card action"><button class="cardopen" onclick={()=>openDetail(a,'action')}><strong>{a.title}</strong><p>{a.project||a.projectId||'hors projet'}</p></button><footer><span>{(a.contexts||[]).join(', ')}</span><button onclick={()=>removeCart(a)}>retirer</button></footer></article>{/each}</div>{:else}<div class="empty">Panier vide. Ajoute des next actions depuis l’onglet Actions.</div>{/if}</section><section><h2>Inbox rapide</h2><div class="capture"><input placeholder="Capture rapide — câblage écriture à venir" disabled/><button disabled>capturer</button></div><div class="cards compact">{#each cockpit.inbox.slice(0,12) as a}<button class="card" onclick={()=>openDetail(a,'inbox')}><strong>{a.title}</strong><p>{a.excerpt}</p><small>{a.path}</small></button>{/each}</div></section></div><section class="references-inline"><h2>Références</h2><input bind:value={referenceQuery} placeholder="chercher référence, projet, personne…"/><div class="cards compact">{#each filteredReferences as r}<button class="card" onclick={()=>openDetail(r,'reference')}><strong>{r.title}</strong><p>{r.excerpt}</p><small>{r.projectId||r.actionId||'global'} · {r.path}</small></button>{/each}</div></section></section>
  {/if}

  {#if page==='review'}
    <section class="panel page"><header><p>Pré-revue Katia</p><h1>Review</h1><span>Anomalies GTD à clarifier, sans bruit infra.</span></header><div class="alerts">{#each cockpit.review as r}<article>⚠️ {r}</article>{/each}</div><h2>Projets actifs sans next action</h2><div class="cards projects">{#each noNextProjects as p}<article class="card warncard"><strong>{p.title}</strong><p>{p.result||p.excerpt}</p><div class="actions"><button disabled>corriger next</button><button disabled>parquer</button><button disabled>demander Katia</button></div></article>{/each}</div><h2>Waiting / blocked</h2><div class="cards compact">{#each cockpit.waiting.slice(0,24) as a}<button class="card" onclick={()=>openDetail(a,'waiting')}><strong>{a.title}</strong><p>{a.finish||a.excerpt}</p><small>{a.followupAt||a.scheduledAt||'sans date'} · {a.project||a.projectId}</small></button>{/each}</div></section>
  {/if}
</main>

{#if selected}
  <div class="modal-backdrop" role="presentation" onclick={closeDetail}>
    <div class="modal panel" role="dialog" aria-modal="true" aria-labelledby="detail-title" tabindex="-1" onclick={(event)=>event.stopPropagation()} onkeydown={(event)=>{ if(event.key==='Escape') closeDetail(); }}>
      <header class="modal-head"><div><p>{selected.kind||'item'}</p><h1 id="detail-title">{selected.title}</h1></div><button class="close" onclick={closeDetail} aria-label="Fermer">×</button></header>
      <div class="modal-grid">
        <article>
          <small>{selected.path}</small>
          {#if selected.project||selected.projectId}<p><b>Projet :</b> {selected.project||selected.projectId}</p>{/if}
          {#if selected.result}<p><b>Résultat voulu :</b> {selected.result}</p>{/if}
          {#if selected.finish||selected.criterion}<p><b>Fini quand :</b> {selected.finish||selected.criterion}</p>{/if}
          {#if selected.guardrail}<p><b>Garde-fou :</b> {selected.guardrail}</p>{/if}
          {#if selected.excerpt}<p class="excerpt">{selected.excerpt}</p>{/if}
          <div class="chips">{#each [...(selected.contexts||[]),...(selected.tags||[]),...(selected.people||[])] as tag}<em>{tag}</em>{/each}</div>
          <div class="proposal"><button disabled>éditer</button><button disabled>postpone</button><button disabled>done</button><small>Mutation réelle désactivée : audit/webhook Katia requis.</small></div>
        </article>
        <aside>
          {#if selected.kind==='project'}
            <h2>Actions liées</h2>{#each projectActions as a}<button class="mini" onclick={()=>openDetail(a,'action')}>{a.title}</button>{/each}
          {/if}
          <h2>Références liées</h2>{#if refsForSelection.length}{#each refsForSelection as r}<code>{r.title}</code>{/each}{:else}<p class="muted">Aucune référence liée détectée.</p>{/if}
        </aside>
      </div>
    </div>
  </div>
{/if}

<style>
:global(body){margin:0;background:#05070b;color:#dff8ff;font-family:Inter,ui-sans-serif,system-ui,sans-serif}.shell{max-width:1540px;margin:0 auto;padding:12px}.panel{background:rgba(8,15,25,.86);border:1px solid rgba(100,220,255,.16);border-radius:14px;box-shadow:0 14px 44px rgba(0,0,0,.30)}a{color:#68e7ff;text-decoration:none}.topbar{position:sticky;top:0;z-index:20;display:flex;gap:12px;align-items:center;padding:7px 10px;backdrop-filter:blur(10px)}.brand{display:flex;align-items:center;gap:8px;min-width:128px}.brand small{color:#8aa6b8}.heart{width:16px;height:16px;border-radius:50%;background:radial-gradient(circle,#ff2c68 0 35%,#163245 40%);box-shadow:0 0 14px #ff2c68}.leds{display:flex;gap:6px;flex-wrap:wrap;flex:1}.led{position:relative;display:inline-flex;align-items:center;gap:5px;padding:3px 7px;border:1px solid rgba(104,231,255,.16);border-radius:999px;background:rgba(255,255,255,.03);font-size:11px;line-height:1.2}.tip{display:none;position:absolute;top:calc(100% + 8px);left:0;min-width:360px;max-width:900px;padding:12px;border-radius:12px;border:1px solid rgba(104,231,255,.3);background:#06101b;z-index:40}.led:hover .tip{display:grid;gap:4px}.tip table{border-collapse:collapse;font:11px ui-monospace,monospace}.tip td,.tip th{padding:3px 6px;border-bottom:1px solid rgba(104,231,255,.12)}.bad td,.issue{color:#ffd447}i{display:inline-block;width:8px;height:8px;border-radius:50%;background:#607080}.ok{background:#28ff9c;box-shadow:0 0 9px #28ff9c}.warn{background:#ffd447;box-shadow:0 0 9px #ffd447}.down{background:#ff3f6e;box-shadow:0 0 9px #ff3f6e}nav{display:flex;gap:10px;font-size:12px}.authority{max-width:1540px;margin:10px auto 0;padding:0 4px;display:flex;gap:10px;align-items:center;color:#8aa6b8;font-size:12px}.authority b{color:#ffd447}.authority code{margin-left:auto}.tabs{margin-top:10px;padding:7px;display:flex;gap:8px;flex-wrap:wrap}.tabs button,.side button,.card footer button,.actions button{border:1px solid rgba(104,231,255,.16);border-radius:999px;background:rgba(255,255,255,.03);color:#dff8ff;padding:8px 12px;cursor:pointer}.tabs button.active,.side button.active{background:rgba(104,231,255,.16);border-color:rgba(104,231,255,.45)}.work{display:grid;grid-template-columns:230px minmax(0,1fr);gap:12px;margin-top:12px}.side,.main,.page{padding:14px}.side{display:grid;align-content:start;gap:7px}.side h2{margin:0 0 8px}input{width:100%;box-sizing:border-box;border:1px solid rgba(104,231,255,.18);background:rgba(255,255,255,.04);color:#dff8ff;border-radius:10px;padding:10px}.wide-search{margin-bottom:12px}header{margin-bottom:14px}header p{margin:0 0 5px;color:#68e7ff;text-transform:uppercase;letter-spacing:.18em;font-size:11px}h1{margin:0 0 4px;font-size:30px}h2{margin:14px 0 10px}.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(315px,1fr));gap:10px}.cards.projects{grid-template-columns:repeat(auto-fit,minmax(360px,1fr))}.cards.compact{grid-template-columns:repeat(auto-fit,minmax(280px,1fr))}.card{border:1px solid rgba(104,231,255,.12);border-radius:13px;background:rgba(255,255,255,.03);color:#dff8ff;padding:12px;text-align:left;font:inherit}.card:hover{border-color:rgba(104,231,255,.38)}button.card{cursor:pointer}.cardopen{display:block;width:100%;padding:0;border:0;background:transparent;color:inherit;text-align:left;font:inherit;cursor:pointer}.card strong{display:block;color:#fff}.card p,.modal p{color:#adc6d4;line-height:1.45}.row{display:flex;justify-content:space-between;gap:10px}.chips{display:flex;gap:5px;flex-wrap:wrap;margin:8px 0}.chips em{font-style:normal;color:#b9f4ff;border:1px solid rgba(104,231,255,.16);background:rgba(104,231,255,.06);border-radius:999px;padding:3px 7px;font-size:11px}.chips.meta em{color:#ffd447;border-color:rgba(255,212,71,.2);background:rgba(255,212,71,.06)}.references-inline{margin-top:16px}.card footer{display:flex;justify-content:space-between;gap:10px;align-items:center;margin-top:10px;color:#8aa6b8;font-size:12px}.footactions{display:flex;gap:6px}.card footer button{padding:5px 8px}.card footer .addcart{border:0;color:#071016;background:#68e7ff}.warncard{border-color:rgba(255,212,71,.45);background:rgba(255,212,71,.055)}code{display:block;font:12px ui-monospace,monospace;color:#b9f4ff;margin:3px 0;overflow-wrap:anywhere}.page{margin-top:12px}.split{display:grid;grid-template-columns:1fr 1fr;gap:14px}.capture{display:flex;gap:8px;margin-bottom:12px}.capture button,.proposal button{border:1px solid rgba(104,231,255,.18);background:rgba(255,255,255,.04);color:#8aa6b8;border-radius:10px;padding:9px}.empty,.alerts article{padding:14px;border:1px dashed rgba(104,231,255,.2);border-radius:12px;color:#8aa6b8}.alerts{display:grid;gap:8px}.actions{display:flex;gap:8px;flex-wrap:wrap}.modal-backdrop{position:fixed;inset:0;z-index:100;display:grid;place-items:center;background:rgba(0,0,0,.66);backdrop-filter:blur(5px);padding:4vh 4vw}.modal{width:min(80vw,1220px);height:min(80vh,860px);overflow:auto;padding:18px}.modal-head{display:flex;justify-content:space-between;gap:16px;align-items:flex-start;border-bottom:1px solid rgba(104,231,255,.12);padding-bottom:10px}.modal-head p{margin:0 0 5px;color:#68e7ff;text-transform:uppercase;letter-spacing:.18em;font-size:11px}.close{width:40px;height:40px;border-radius:50%;border:1px solid rgba(104,231,255,.2);background:rgba(255,255,255,.04);color:#dff8ff;font-size:26px;cursor:pointer}.modal-grid{display:grid;grid-template-columns:minmax(0,1fr) 340px;gap:18px;margin-top:14px}.excerpt{white-space:pre-wrap}.proposal{display:flex;gap:8px;align-items:center;flex-wrap:wrap;margin-top:14px;padding:10px;border-radius:12px;background:rgba(104,231,255,.05)}.mini{display:block;width:100%;margin:5px 0;padding:8px;border-radius:9px;border:1px solid rgba(104,231,255,.14);background:rgba(255,255,255,.03);color:#dff8ff;text-align:left;cursor:pointer}.muted,.card small,header span{color:#8aa6b8}@media(max-width:1100px){.work,.split,.modal-grid{grid-template-columns:1fr}.authority{flex-wrap:wrap}.authority code{margin-left:0}.topbar{align-items:flex-start;flex-wrap:wrap}.modal{width:92vw;height:86vh}} 
</style>
