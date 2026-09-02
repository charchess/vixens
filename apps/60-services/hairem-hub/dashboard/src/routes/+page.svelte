<script lang="ts">
  let { data } = $props();
  let active = $state('now');

  const cockpit = $derived(data.gtdCockpit);
  const tabs = $derived([
    ['now', 'Now', cockpit.now.length],
    ['inbox', 'Inbox', cockpit.inbox.length],
    ['clarify', 'Clarify', cockpit.inbox.length],
    ['projects', 'Projects', cockpit.projects.filter((p) => p.active).length],
    ['waiting', 'Waiting', cockpit.waiting.length],
    ['someday', 'Someday', cockpit.someday.length],
    ['review', 'Review', cockpit.review.length]
  ]);

  const activeProjects = $derived(cockpit.projects.filter((p) => p.active));
  const noNextProjects = $derived(cockpit.projects.filter((p) => p.noNextAction));

  function contextLabel(item) {
    return item.section?.replace(/^@/, '') || item.context || 'contexte libre';
  }
</script>

<svelte:head>
  <title>hAIrem Dashboard</title>
  <meta name="description" content="Cockpit hAIrem GTD" />
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
              {#if line === ''}<em></em>{:else}<code>{line}</code>{/if}
            {/each}
          </span>
        </span>
      {/each}
    </div>
    <nav><a href={data.links.llmwiki}>llmwiki</a><a href={data.links.gtdwiki}>gtdwiki</a></nav>
  </section>

  <section class="capture panel">
    <div><b>Capture rapide</b><span>lecture seule · capture ≠ engagement · source Katia</span></div>
    <a class="ghost" href={data.links.gtdwiki}>ouvrir GTDWiki</a>
  </section>

  <section class="layout">
    <aside class="panel navpane">
      <h2>GTD</h2>
      {#each tabs as tab}
        <button class:active={active === tab[0]} onclick={() => active = tab[0]}>
          <span>{tab[1]}</span><b>{tab[2]}</b>
        </button>
      {/each}
      <div class="trust" class:trust-warn={cockpit.trust.led === 'warn'} class:trust-down={cockpit.trust.led === 'down'}>
        <small>Confiance système</small><strong>{cockpit.trust.label}</strong>
        <code>inbox {cockpit.trust.inboxOpen}</code><code>P1 {cockpit.trust.p1Open} · next {cockpit.trust.nextOpen}</code><code>projets actifs {cockpit.trust.projectsActive}</code><code>sans next {cockpit.trust.projectsNoNext}</code><code>waiting {cockpit.trust.waiting}</code>
      </div>
    </aside>

    <article class="panel mainpane">
      {#if active === 'now'}
        <header><p>Engage</p><h1>Que faire maintenant ?</h1><span>Actions validées, concrètes, triées P1→P2. Pas tout le backlog.</span></header>
        <div class="filters"><button>5–15 min</button><button>énergie basse</button><button>infra</button><button>maison</button><button>admin</button></div>
        {#if noNextProjects.length}<div class="warning"><b>{noNextProjects.length} projet(s) actif(s) sans prochaine action nette</b><span>À corriger en revue avant de les pousser.</span></div>{/if}
        <div class="cards actioncards">{#each cockpit.now as item}<article class="task"><div><strong>{item.title}</strong><small>{contextLabel(item)} · {item.priority || 'Px'}</small></div>{#if item.project}<p>Projet : {item.project}</p>{/if}{#if item.criterion}<p>Fini quand : {item.criterion}</p>{/if}</article>{/each}</div>
      {:else if active === 'inbox'}
        <header><p>Capture</p><h1>Inbox brute</h1><span>Non clarifié. Non engagé. À traiter sans honte.</span></header>
        <div class="cards">{#each cockpit.inbox as item}<article class="task"><strong>{item.title}</strong><small>{item.section}</small>{#if item.excerpt}<p>{item.excerpt}</p>{/if}</article>{/each}</div>
      {:else if active === 'clarify'}
        <header><p>Clarify</p><h1>Machine à décisions</h1><span>Une capture doit sortir vers trash, référence, someday, projet, next action, waiting-for ou calendrier.</span></header>
        {#if cockpit.inbox[0]}<article class="clarify-card"><small>Premier item inbox</small><h2>{cockpit.inbox[0].title}</h2><div class="decision-grid"><span>Qu’est-ce que c’est ?</span><b>à décider</b><span>Actionnable ?</span><b>oui / non</b><span>Sortie GTD</span><b>trash · référence · someday · projet · next · waiting</b></div></article>{:else}<div class="empty">Inbox claire. Rien à clarifier.</div>{/if}
      {:else if active === 'projects'}
        <header><p>Organize</p><h1>Projets actifs</h1><span>Résultat voulu + prochaine action visible. Sinon le projet casse la confiance.</span></header>
        <div class="cards projectcards">{#each activeProjects as project}<article class="project" class:bad={project.noNextAction}><strong>{project.title}</strong>{#if project.result}<p>{project.result}</p>{/if}<code>Next: {project.nextAction || 'aucune'}</code>{#if project.guardrail}<small>{project.guardrail}</small>{/if}</article>{/each}</div>
      {:else if active === 'waiting'}
        <header><p>Organize</p><h1>Waiting for</h1><span>Attentes externes visibles. Pas des actions actives.</span></header>
        <div class="cards">{#each cockpit.waiting as item}<article class="task"><strong>{item.title}</strong>{#if item.expected}<p>Attendu : {item.expected}</p>{/if}{#if item.actionAfter}<small>Après déclencheur : {item.actionAfter}</small>{/if}</article>{/each}</div>
      {:else if active === 'someday'}
        <header><p>Parking</p><h1>Someday / Maybe</h1><span>Idées conservées, pas engagées.</span></header>
        <div class="cards compactcards">{#each cockpit.someday as item}<article class="task"><strong>{item.title}</strong></article>{/each}</div>
      {:else if active === 'review'}
        <header><p>Reflect</p><h1>Revue de confiance</h1><span>Rituel court : Get clear, Get current, Get creative.</span></header>
        <ol class="review">{#each cockpit.review as step}<li>{step}</li>{/each}</ol>
        <div class="review-grid"><div><b>Get clear</b><span>vider inbox, capturer le diffus</span></div><div><b>Get current</b><span>projets, next, waiting, calendrier</span></div><div><b>Get creative</b><span>relire someday, réactiver ou supprimer</span></div></div>
      {/if}
    </article>

    <aside class="panel detailpane">
      <h2>Doctrine</h2><p>LLMWiki porte la méthode. Katia porte les engagements vivants.</p>
      <code>capture ≠ engagement</code><code>inbox ≠ todo</code><code>projet actif ⇒ next action</code><code>revue ⇒ confiance</code><hr />
      <small>Source GTD</small><code>{cockpit.source}</code><small>Mises à jour</small><code>next {cockpit.updated.next}</code><code>projects {cockpit.updated.projects}</code><code>waiting {cockpit.updated.waiting}</code>
    </aside>
  </section>
</main>

<style>
  :global(body){margin:0;background:#05070b;color:#dff8ff;font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;font-size:14px}:global(a){color:#68e7ff;text-decoration:none}.shell{max-width:1500px;margin:0 auto;padding:14px}.panel{background:rgba(8,15,25,.82);border:1px solid rgba(100,220,255,.16);border-radius:14px;box-shadow:0 12px 40px rgba(0,0,0,.28)}.topbar{height:42px;padding:0 12px;display:flex;align-items:center;gap:18px;position:sticky;top:0;z-index:5;backdrop-filter:blur(10px);overflow:visible}.brand{display:flex;align-items:center;gap:9px;min-width:116px}.brand strong{font-size:13px}.brand small{display:block;color:#8aa6b8;font-size:10px}.mark{width:18px;height:18px;border-radius:50%;background:radial-gradient(circle,#ff2c68 0 35%,#163245 40%);box-shadow:0 0 14px #ff2c68}.ledline{display:flex;gap:8px;align-items:center;flex:1}.ledchip{position:relative;display:inline-flex;gap:6px;align-items:center;padding:4px 8px;border:1px solid rgba(104,231,255,.16);border-radius:999px;background:rgba(255,255,255,.03);font-size:12px;white-space:nowrap}.tooltip{position:absolute;left:0;top:calc(100% + 10px);display:none;min-width:330px;max-width:620px;padding:12px;border:1px solid rgba(104,231,255,.32);border-radius:12px;background:linear-gradient(180deg,rgba(4,10,18,.98),rgba(8,18,30,.98));box-shadow:0 18px 60px rgba(0,0,0,.55);z-index:20}.ledchip:hover .tooltip{display:grid;gap:3px}.tooltip b{color:#fff;margin-bottom:5px}.tooltip code,.detailpane code,.trust code{font-family:ui-monospace,SFMono-Regular,Consolas,monospace;color:#b9f4ff;background:transparent}.tooltip em{height:8px}i{display:inline-block;width:9px;height:9px;border-radius:50%;background:#607080;box-shadow:0 0 7px #607080}.ok{background:#28ff9c;box-shadow:0 0 10px #28ff9c}.warn{background:#ffd447;box-shadow:0 0 10px #ffd447}.down{background:#ff3f6e;box-shadow:0 0 10px #ff3f6e}.topbar nav{display:flex;gap:10px;font-size:12px}.capture{margin-top:12px;padding:12px 14px;display:flex;justify-content:space-between;align-items:center}.capture b{display:block}.capture span,header span,small{color:#8aa6b8}.ghost{border:1px solid rgba(104,231,255,.22);padding:7px 10px;border-radius:10px;background:rgba(104,231,255,.06)}.layout{display:grid;grid-template-columns:210px minmax(0,1fr) 280px;gap:12px;margin-top:12px}.navpane,.mainpane,.detailpane{padding:14px}.navpane h2,.detailpane h2{margin:0 0 10px}.navpane button{width:100%;display:flex;justify-content:space-between;align-items:center;margin:5px 0;padding:10px;border-radius:10px;border:1px solid rgba(104,231,255,.12);background:rgba(255,255,255,.025);color:#dff8ff;text-align:left}.navpane button.active{background:rgba(104,231,255,.13);border-color:rgba(104,231,255,.42)}.navpane button b{color:#68e7ff}.trust{margin-top:14px;padding:11px;border-radius:12px;border:1px solid rgba(40,255,156,.24);display:grid;gap:4px}.trust strong{color:#28ff9c}.trust-warn{border-color:rgba(255,212,71,.35)}.trust-warn strong{color:#ffd447}.trust-down{border-color:rgba(255,63,110,.45)}.trust-down strong{color:#ff6f91}header{margin-bottom:14px}header p{color:#68e7ff;text-transform:uppercase;letter-spacing:.18em;font-size:11px;margin:0 0 5px}h1{font-size:28px;margin:0 0 5px}.filters{display:flex;gap:8px;flex-wrap:wrap;margin-bottom:12px}.filters button{border:1px solid rgba(104,231,255,.15);border-radius:999px;background:rgba(255,255,255,.03);color:#b9f4ff;padding:7px 10px}.warning{display:flex;justify-content:space-between;gap:12px;padding:10px 12px;margin-bottom:12px;border-radius:12px;background:rgba(255,212,71,.09);border:1px solid rgba(255,212,71,.28)}.cards{display:grid;gap:10px}.actioncards{grid-template-columns:repeat(auto-fit,minmax(300px,1fr))}.projectcards{grid-template-columns:repeat(auto-fit,minmax(340px,1fr))}.compactcards{grid-template-columns:repeat(auto-fit,minmax(260px,1fr))}.task,.project,.clarify-card{border:1px solid rgba(104,231,255,.12);border-radius:12px;padding:12px;background:rgba(255,255,255,.03)}.task strong,.project strong{display:block;color:#fff}.task small,.project small{display:block;margin-top:4px}.task p,.project p,.detailpane p{color:#adc6d4;line-height:1.45}.project code{display:block;margin-top:8px;color:#b9f4ff}.project.bad{border-color:rgba(255,63,110,.45)}.clarify-card h2{font-size:24px}.decision-grid{display:grid;grid-template-columns:170px 1fr;gap:8px;padding-top:10px}.decision-grid span{color:#8aa6b8}.review{display:grid;gap:8px}.review li{padding:10px;border-radius:10px;background:rgba(255,255,255,.03)}.review-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:10px;margin-top:14px}.review-grid div{border:1px solid rgba(104,231,255,.12);border-radius:12px;padding:12px}.review-grid span{display:block;color:#8aa6b8;margin-top:4px}.detailpane{display:grid;align-content:start;gap:8px}.detailpane hr{border:0;border-top:1px solid rgba(104,231,255,.14);width:100%}.empty{padding:22px;border:1px dashed rgba(104,231,255,.2);border-radius:12px;color:#8aa6b8}@media(max-width:1100px){.layout{grid-template-columns:1fr}.navpane{display:grid;grid-template-columns:repeat(2,1fr);gap:6px}.navpane h2,.trust{grid-column:1/-1}.detailpane{display:none}.topbar{height:auto;min-height:42px;flex-wrap:wrap;padding:8px 12px}.ledline{order:3;flex-basis:100%;flex-wrap:wrap}.review-grid{grid-template-columns:1fr}}
</style>
