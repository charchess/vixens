<script lang="ts">
  let { data } = $props();
  let page = $state('gtd');
  let active = $state('now');
  let nowFilter = $state('all');
  let selected = $state(null);
  let reviewFocus = $state('all');

  const cockpit = $derived(data.gtdCockpit);
  const activeProjects = $derived(cockpit.projects.filter((p) => p.active));
  const noNextProjects = $derived(cockpit.projects.filter((p) => p.noNextAction));
  const filteredNow = $derived(cockpit.now.filter((item) => matchesNowFilter(item, nowFilter)));
  const infraIssues = $derived(data.runtime.components.filter((c) => c.led !== 'ok'));
  const gtdIssues = $derived([
    ...(noNextProjects.length ? [{ level: 'yellow', title: `${noNextProjects.length} projet(s) actifs sans next action`, detail: 'Review GTD nécessaire', action: 'Lister' }] : []),
    ...(cockpit.inbox.length ? [{ level: 'blue', title: `${cockpit.inbox.length} capture(s) à clarifier`, detail: 'Inbox ≠ engagement', action: 'Clarifier' }] : []),
    ...(cockpit.trust.p1Open ? [{ level: 'red', title: `${cockpit.trust.p1Open} action(s) P1 disponible(s)`, detail: 'Choisir selon énergie/contexte', action: 'Filtrer' }] : [])
  ]);

  const pageTabs = [
    ['gtd', 'GTD'],
    ['review', 'Review Queue'],
    ['productivity', 'Productivity']
  ];

  const tabs = $derived([
    ['now', 'Now', cockpit.now.length],
    ['inbox', 'Inbox', cockpit.inbox.length],
    ['clarify', 'Clarify', cockpit.inbox.length],
    ['projects', 'Projects', activeProjects.length],
    ['waiting', 'Waiting', cockpit.waiting.length],
    ['someday', 'Someday', cockpit.someday.length]
  ]);

  const nowFilters = [
    ['all', 'tout'],
    ['short', '5–15 min'],
    ['low', 'énergie basse'],
    ['infra', 'infra'],
    ['maison', 'maison'],
    ['admin', 'admin'],
    ['sensitive', 'sensible'],
    ['blocked', 'blocked']
  ];

  const productivityCards = $derived([
    ['GTD open', data.counts.gtdTotal, `${cockpit.trust.nextOpen} next · ${cockpit.trust.inboxOpen} inbox`],
    ['Review queue', gtdIssues.length, `${noNextProjects.length} sans next · ${cockpit.inbox.length} inbox`],
    ['Infra signal', infraIssues.length, infraIssues.length ? 'dégradation à part du GTD' : 'runtime sain'],
    ['LLMWiki', data.counts.llmTotal, 'pages indexées']
  ]);

  function textOf(item) {
    return [item.title, item.section, item.context, item.project, item.priority, item.criterion, item.guardrail, item.excerpt, item.statusText].filter(Boolean).join(' ').toLowerCase();
  }

  function matchesNowFilter(item, filter) {
    if (filter === 'all') return true;
    const text = textOf(item);
    if (filter === 'short') return /5\s*[–-]\s*15|15\s*min|quart d.?heure|rapide|court/.test(text);
    if (filter === 'low') return /énergie basse|energie basse|low energy|fatigue|facile|léger|leger/.test(text);
    if (filter === 'sensitive') return /finance|santé|sante|juridique|électricité|electricite|network-prod|achat|publication|sensible/.test(text);
    if (filter === 'blocked') return /blocked|bloqué|bloque|attente|waiting/.test(text);
    return text.includes(filter);
  }

  function contextLabel(item) {
    return item.section?.replace(/^@/, '') || item.context || 'contexte libre';
  }

  function selectItem(item, kind = 'action') {
    selected = { ...item, kind };
  }

  function showMissingNext() {
    page = 'review';
    reviewFocus = 'missing-next';
  }

  function levelIcon(level) {
    return level === 'red' ? '🔴' : level === 'yellow' ? '🟡' : level === 'green' ? '🟢' : level === 'blue' ? '🔵' : '⚪';
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
        <svelte:element this={c.href ? 'a' : 'span'} class="ledchip" href={c.href} target={c.href ? '_blank' : undefined} rel={c.href ? 'noreferrer' : undefined} aria-label={c.aria || c.label}>
          <i class:ok={c.led === 'ok'} class:warn={c.led === 'warn'} class:down={c.led === 'down'}></i>{c.label}
          <span class="tooltip" class:wide={c.tableRows?.length} role="tooltip">
            <b>{c.label}</b>
            {#if c.primaryIssue}<code class="issue">{c.primaryIssue}</code>{/if}
            {#if c.tableRows?.length}
              <table class="banktable"><thead><tr><th>bank</th><th>facts</th><th>pend</th><th>run</th><th>stuck</th><th>fail</th><th>llm 1h/24h</th><th>last</th></tr></thead><tbody>{#each c.tableRows as row}<tr class:rowwarn={row.stuck || row.failed || row.llm1h}><td>{row.bank}</td><td>{row.facts}</td><td>{row.pending}</td><td>{row.running}</td><td>{row.stuck}</td><td>{row.failed}</td><td>{row.llm1h}/{row.llm24h}</td><td>{row.last}</td></tr>{/each}</tbody></table>
            {:else}
              {#each c.detailLines as line}{#if line === ''}<em></em>{:else}<code>{line}</code>{/if}{/each}
            {/if}
          </span>
        </svelte:element>
      {/each}
    </div>
    <nav><a href={data.links.llmwiki} target="_blank" rel="noreferrer">llmwiki</a><a href={data.links.gtdwiki} target="_blank" rel="noreferrer">gtd files</a></nav>
  </section>

  <section class="authority panel">
    <div><b>Source d’autorité : fichiers GTD Katia</b><span>Dashboard = interface/vue. Capture ≠ engagement. Modifications auditées, Katia notifiée si écriture source.</span></div>
    <code>{cockpit.source}</code>
  </section>

  <section class="pagetabs panel">
    {#each pageTabs as tab}<button class:active={page === tab[0]} onclick={() => page = tab[0]}>{tab[1]}</button>{/each}
  </section>

  {#if page === 'gtd'}
    <section class="signalgrid">
      <article class="signal human panel"><h3>Décision humaine / GTD</h3>{#if gtdIssues.length}{#each gtdIssues as issue}<button class="signalrow" onclick={() => issue.action === 'Lister' ? showMissingNext() : active = issue.action === 'Clarifier' ? 'clarify' : 'now'}><span>{levelIcon(issue.level)}</span><b>{issue.title}</b><small>{issue.detail}</small></button>{/each}{:else}<p>🟢 GTD clair. Aucune alerte humaine prioritaire.</p>{/if}</article>
      <article class="signal infra panel"><h3>Runtime infra/personas</h3>{#if infraIssues.length}{#each infraIssues as c}<span class="signalrow passive"><i class:warn={c.led === 'warn'} class:down={c.led === 'down'}></i><b>{c.label}</b><small>surveillance infra, séparée du choix GTD</small></span>{/each}{:else}<p>🟢 Runtime sain. Rien à mélanger avec les next-actions.</p>{/if}</article>
    </section>

    <section class="capture panel"><div><b>Capture rapide</b><span>lecture seule · capturé ≠ engagé · source Katia</span></div><a class="ghost" href={data.links.gtdwiki} target="_blank" rel="noreferrer">ouvrir fichiers GTD</a></section>

    <section class="layout">
      <aside class="panel navpane">
        <h2>GTD</h2>
        {#each tabs as tab}<button class:active={active === tab[0]} onclick={() => active = tab[0]}><span>{tab[1]}</span><b>{tab[2]}</b></button>{/each}
        <button class="reviewlink" onclick={showMissingNext}>🟡 Sans next action <b>{noNextProjects.length}</b></button>
        <div class="trust" class:trust-warn={cockpit.trust.led === 'warn'} class:trust-down={cockpit.trust.led === 'down'}><small>Confiance GTD</small><strong>{cockpit.trust.label}</strong><code>inbox {cockpit.trust.inboxOpen}</code><code>P1 {cockpit.trust.p1Open} · next {cockpit.trust.nextOpen}</code><code>projets actifs {cockpit.trust.projectsActive}</code><code>waiting {cockpit.trust.waiting}</code></div>
      </aside>

      <article class="panel mainpane">
        {#if active === 'now'}
          <header><p>Engage</p><h1>Palette d’actions choisissables</h1><span>Toutes les next-actions actives filtrables. Katia ne force pas une seule action.</span></header>
          <div class="filters">{#each nowFilters as filter}<button class:active={nowFilter === filter[0]} onclick={() => nowFilter = filter[0]}>{filter[1]}</button>{/each}</div>
          {#if noNextProjects.length}<button class="warning clickable" onclick={showMissingNext}><b>{noNextProjects.length} projet(s) actif(s) sans prochaine action nette</b><span>Cliquer pour liste + correction/parquage/fusion/clarification.</span></button>{/if}
          {#if filteredNow.length}<div class="cards actioncards">{#each filteredNow as item}<button type="button" class="task cardbutton" class:sensitive={matchesNowFilter(item, 'sensitive')} onclick={() => selectItem(item, 'action')}><div><strong>{item.title}</strong><small>{contextLabel(item)} · {item.priority || 'Px'}</small></div>{#if item.project}<p>Projet : {item.project}</p>{/if}{#if item.criterion}<p>Fini quand : {item.criterion}</p>{/if}<small>Pourquoi maintenant : priorité/contexte disponibles · source {item.path}</small></button>{/each}</div>{:else}<div class="empty">Aucune action ne correspond à ce filtre.</div>{/if}
        {:else if active === 'inbox'}
          <header><p>Capture</p><h1>Inbox brute</h1><span>Non clarifié. Non engagé. À traiter sans honte.</span></header>
          <div class="cards">{#each cockpit.inbox as item}<button type="button" class="task cardbutton" onclick={() => selectItem(item, 'capture')}><strong>{item.title}</strong><small>{item.section}</small>{#if item.excerpt}<p>{item.excerpt}</p>{/if}</button>{/each}</div>
        {:else if active === 'clarify'}
          <header><p>Clarify</p><h1>Machine à décisions</h1><span>Sortir chaque capture vers trash, référence, someday, projet, next action, waiting-for ou calendrier.</span></header>
          {#if cockpit.inbox[0]}<button type="button" class="clarify-card cardbutton" onclick={() => selectItem(cockpit.inbox[0], 'capture')}><small>Premier item inbox</small><h2>{cockpit.inbox[0].title}</h2><div class="decision-grid"><span>Qu’est-ce que c’est ?</span><b>à décider</b><span>Actionnable ?</span><b>oui / non</b><span>Sortie GTD</span><b>trash · référence · someday · projet · next · waiting</b></div></button>{:else}<div class="empty">Inbox claire. Rien à clarifier.</div>{/if}
        {:else if active === 'projects'}
          <header><p>Organize</p><h1>Projets actifs</h1><span>Résultat voulu + prochaine action visible. Sinon le projet casse la confiance.</span></header>
          <div class="cards projectcards">{#each activeProjects as project}<button type="button" class="project cardbutton" class:bad={project.noNextAction} onclick={() => selectItem(project, 'project')}><strong>{project.title}</strong>{#if project.result}<p>{project.result}</p>{/if}<code>Next: {project.nextAction || 'aucune'}</code>{#if project.guardrail}<small>{project.guardrail}</small>{/if}</button>{/each}</div>
        {:else if active === 'waiting'}
          <header><p>Organize</p><h1>Waiting for</h1><span>Attentes externes visibles. Pas des obligations actives.</span></header>
          <div class="cards">{#each cockpit.waiting as item}<button type="button" class="task cardbutton" onclick={() => selectItem(item, 'waiting_for')}><strong>{item.title}</strong>{#if item.expected}<p>Attendu : {item.expected}</p>{/if}{#if item.actionAfter}<small>Après déclencheur : {item.actionAfter}</small>{/if}</button>{/each}</div>
        {:else if active === 'someday'}
          <header><p>Parking</p><h1>Someday / Maybe</h1><span>Idées conservées, pas engagées.</span></header>
          <div class="cards compactcards">{#each cockpit.someday as item}<button type="button" class="task cardbutton" onclick={() => selectItem(item, 'someday')}><strong>{item.title}</strong></button>{/each}</div>
        {/if}
      </article>

      <aside class="panel detailpane">
        <h2>Détail / Doctrine</h2>
        {#if selected}
          <small>{selected.kind} · {selected.path}</small><strong>{selected.title}</strong>{#if selected.project}<p>Projet : {selected.project}</p>{/if}{#if selected.criterion}<p>Fini quand : {selected.criterion}</p>{/if}{#if selected.guardrail}<p>Garde-fou : {selected.guardrail}</p>{/if}<div class="proposal"><b>Édition sûre</b><button disabled>proposer modification</button><button disabled>masquer aujourd’hui</button><small>Contrat audit/webhook prêt côté API. Écriture réelle à câbler après schéma source.</small></div>
        {:else}
          <p>Cliquer une action/projet pour voir contexte, source, critère fini et actions sûres.</p>
        {/if}
        <hr /><code>capture ≠ engagement</code><code>projet actif ⇒ next action</code><code>waiting ≠ obligation</code><hr /><small>Mises à jour</small><code>next {cockpit.updated.next}</code><code>projects {cockpit.updated.projects}</code><code>waiting {cockpit.updated.waiting}</code>
      </aside>
    </section>
  {:else if page === 'review'}
    <section class="reviewpage panel">
      <header><p>Review Queue</p><h1>Confiance GTD</h1><span>Dette de clarification séparée des alertes infra.</span></header>
      <div class="filters"><button class:active={reviewFocus === 'all'} onclick={() => reviewFocus = 'all'}>tout</button><button class:active={reviewFocus === 'missing-next'} onclick={() => reviewFocus = 'missing-next'}>sans next action</button><button class:active={reviewFocus === 'inbox'} onclick={() => reviewFocus = 'inbox'}>inbox</button></div>
      {#if reviewFocus !== 'inbox'}<h2>Projets actifs sans prochaine action nette</h2><div class="cards projectcards">{#each noNextProjects as project}<article class="project bad"><strong>{project.title}</strong>{#if project.result}<p>{project.result}</p>{/if}<code>Source: {project.path}</code><div class="actions"><button disabled>corriger next</button><button disabled>parquer</button><button disabled>fusionner</button><button disabled>demander clarification Katia</button></div></article>{/each}</div>{/if}
      {#if reviewFocus !== 'missing-next'}<h2>Inbox à clarifier</h2><div class="cards compactcards">{#each cockpit.inbox as item}<article class="task"><strong>{item.title}</strong><small>{item.section} · {item.path}</small></article>{/each}</div>{/if}
      <h2>Rituel</h2><ol class="review">{#each cockpit.review as step}<li>{step}</li>{/each}</ol>
    </section>
  {:else if page === 'productivity'}
    <section class="productivity panel"><header><p>Productivity</p><h1>Parking productivité</h1><span>Page volontairement légère tant que le GTD n’est pas propre. Pas de gamification culpabilisante.</span></header><div class="cards compactcards">{#each productivityCards as card}<article class="task"><strong>{card[0]}</strong><h2>{card[1]}</h2><small>{card[2]}</small></article>{/each}</div><div class="cards compactcards linksgrid"><a class="ghost" href={data.links.gtdwiki} target="_blank" rel="noreferrer">GTD files</a><a class="ghost" href={data.links.llmwiki} target="_blank" rel="noreferrer">LLMWiki</a><a class="ghost" href={data.links.hindsightUi} target="_blank" rel="noreferrer">Hindsight UI</a><a class="ghost" href={data.links.hermes} target="_blank" rel="noreferrer">Hermes</a></div></section>
  {/if}
</main>

<style>
  :global(body){margin:0;background:#05070b;color:#dff8ff;font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;font-size:14px}:global(a){color:#68e7ff;text-decoration:none}.shell{max-width:1500px;margin:0 auto;padding:14px}.panel{background:rgba(8,15,25,.82);border:1px solid rgba(100,220,255,.16);border-radius:14px;box-shadow:0 12px 40px rgba(0,0,0,.28)}.topbar{height:42px;padding:0 12px;display:flex;align-items:center;gap:18px;position:sticky;top:0;z-index:10;backdrop-filter:blur(10px);overflow:visible}.brand{display:flex;align-items:center;gap:9px;min-width:116px}.brand strong{font-size:13px}.brand small{display:block;color:#8aa6b8;font-size:10px}.mark{width:18px;height:18px;border-radius:50%;background:radial-gradient(circle,#ff2c68 0 35%,#163245 40%);box-shadow:0 0 14px #ff2c68}.ledline{display:flex;gap:8px;align-items:center;flex:1}.ledchip{position:relative;display:inline-flex;gap:6px;align-items:center;padding:4px 8px;border:1px solid rgba(104,231,255,.16);border-radius:999px;background:rgba(255,255,255,.03);font-size:12px;white-space:nowrap}.tooltip{position:absolute;left:0;top:calc(100% + 10px);display:none;min-width:330px;max-width:620px;padding:12px;border:1px solid rgba(104,231,255,.32);border-radius:12px;background:linear-gradient(180deg,rgba(4,10,18,.98),rgba(8,18,30,.98));box-shadow:0 18px 60px rgba(0,0,0,.55);z-index:20}.tooltip.wide{min-width:760px;max-width:min(980px,92vw)}.ledchip:hover .tooltip{display:grid;gap:3px}.tooltip b{color:#fff;margin-bottom:5px}.tooltip code,.detailpane code,.trust code{font-family:ui-monospace,SFMono-Regular,Consolas,monospace;color:#b9f4ff;background:transparent}.tooltip .issue{color:#ffd447;border-bottom:1px solid rgba(255,212,71,.25);padding-bottom:6px;margin-bottom:4px}.tooltip em{height:8px}.banktable{border-collapse:collapse;font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:11px;color:#b9f4ff}.banktable th,.banktable td{padding:3px 7px;border-bottom:1px solid rgba(104,231,255,.1);text-align:right}.banktable th:first-child,.banktable td:first-child{text-align:left;color:#fff}.banktable .rowwarn td{color:#ffd447}i{display:inline-block;width:9px;height:9px;border-radius:50%;background:#607080;box-shadow:0 0 7px #607080}.ok{background:#28ff9c;box-shadow:0 0 10px #28ff9c}.warn{background:#ffd447;box-shadow:0 0 10px #ffd447}.down{background:#ff3f6e;box-shadow:0 0 10px #ff3f6e}.topbar nav{display:flex;gap:10px;font-size:12px}.authority{margin-top:12px;padding:10px 14px;display:flex;justify-content:space-between;gap:12px;align-items:center;border-color:rgba(255,212,71,.32);background:linear-gradient(90deg,rgba(255,212,71,.10),rgba(8,15,25,.82))}.authority b{color:#ffd447}.authority span{display:block;color:#b9a96b}.authority code{font-size:11px;color:#b9f4ff}.pagetabs{margin-top:12px;padding:8px;display:flex;gap:8px}.pagetabs button,.filters button,.actions button{border:1px solid rgba(104,231,255,.16);border-radius:999px;background:rgba(255,255,255,.03);color:#b9f4ff;padding:8px 13px}.pagetabs button.active,.filters button.active{background:rgba(104,231,255,.16);border-color:rgba(104,231,255,.45);color:#fff}.signalgrid{display:grid;grid-template-columns:1fr 1fr;gap:12px;margin-top:12px}.signal{padding:12px}.signal h3{margin:0 0 8px}.signalrow{width:100%;display:grid;grid-template-columns:28px 1fr auto;align-items:center;gap:8px;margin:6px 0;padding:9px 10px;border-radius:11px;border:1px solid rgba(104,231,255,.13);background:rgba(255,255,255,.03);color:#dff8ff;text-align:left}.signalrow.passive{display:grid}.capture{margin-top:12px;padding:12px 14px;display:flex;justify-content:space-between;align-items:center}.capture b{display:block}.capture span,header span,small{color:#8aa6b8}.ghost{border:1px solid rgba(104,231,255,.22);padding:7px 10px;border-radius:10px;background:rgba(104,231,255,.06)}.layout{display:grid;grid-template-columns:210px minmax(0,1fr) 320px;gap:12px;margin-top:12px}.navpane,.mainpane,.detailpane,.reviewpage,.productivity{padding:14px}.navpane h2,.detailpane h2{margin:0 0 10px}.navpane button{width:100%;display:flex;justify-content:space-between;align-items:center;margin:5px 0;padding:10px;border-radius:10px;border:1px solid rgba(104,231,255,.12);background:rgba(255,255,255,.025);color:#dff8ff;text-align:left}.navpane button.active{background:rgba(104,231,255,.13);border-color:rgba(104,231,255,.42)}.navpane button b{color:#68e7ff}.reviewlink{border-color:rgba(255,212,71,.28)!important}.trust{margin-top:14px;padding:11px;border-radius:12px;border:1px solid rgba(40,255,156,.24);display:grid;gap:4px}.trust strong{color:#28ff9c}.trust-warn{border-color:rgba(255,212,71,.35)}.trust-warn strong{color:#ffd447}.trust-down{border-color:rgba(255,63,110,.45)}.trust-down strong{color:#ff6f91}header{margin-bottom:14px}header p{color:#68e7ff;text-transform:uppercase;letter-spacing:.18em;font-size:11px;margin:0 0 5px}h1{font-size:28px;margin:0 0 5px}.filters{display:flex;gap:8px;flex-wrap:wrap;margin-bottom:12px}.warning{width:100%;display:flex;justify-content:space-between;gap:12px;padding:10px 12px;margin-bottom:12px;border-radius:12px;background:rgba(255,212,71,.09);border:1px solid rgba(255,212,71,.28);color:#dff8ff}.clickable{cursor:pointer}.cards{display:grid;gap:10px}.actioncards{grid-template-columns:repeat(auto-fit,minmax(300px,1fr))}.projectcards{grid-template-columns:repeat(auto-fit,minmax(340px,1fr))}.compactcards{grid-template-columns:repeat(auto-fit,minmax(260px,1fr))}.task,.project,.clarify-card{border:1px solid rgba(104,231,255,.12);border-radius:12px;padding:12px;background:rgba(255,255,255,.03);cursor:pointer;text-align:left;color:#dff8ff;font:inherit}.cardbutton{display:block;width:100%}.task:hover,.project:hover,.clarify-card:hover{border-color:rgba(104,231,255,.38)}.task.sensitive{border-color:rgba(255,145,71,.5)}.task strong,.project strong{display:block;color:#fff}.task small,.project small{display:block;margin-top:4px}.task p,.project p,.detailpane p{color:#adc6d4;line-height:1.45}.project code{display:block;margin-top:8px;color:#b9f4ff}.project.bad{border-color:rgba(255,212,71,.45)}.clarify-card h2{font-size:24px}.decision-grid{display:grid;grid-template-columns:170px 1fr;gap:8px;padding-top:10px}.decision-grid span{color:#8aa6b8}.review{display:grid;gap:8px}.review li{padding:10px;border-radius:10px;background:rgba(255,255,255,.03)}.reviewpage,.productivity{margin-top:12px}.reviewpage h2{font-size:16px;color:#ffd447}.actions{display:flex;gap:6px;flex-wrap:wrap;margin-top:10px}.proposal{display:grid;gap:7px;margin-top:12px;padding:10px;border-radius:12px;background:rgba(104,231,255,.05)}.proposal button:disabled,.actions button:disabled{opacity:.55;cursor:not-allowed}.detailpane{display:grid;align-content:start;gap:8px}.detailpane hr{border:0;border-top:1px solid rgba(104,231,255,.14);width:100%}.empty{padding:22px;border:1px dashed rgba(104,231,255,.2);border-radius:12px;color:#8aa6b8}.linksgrid{margin-top:12px}@media(max-width:1100px){.signalgrid,.layout{grid-template-columns:1fr}.navpane{display:grid;grid-template-columns:repeat(2,1fr);gap:6px}.navpane h2,.trust{grid-column:1/-1}.topbar{height:auto;min-height:42px;flex-wrap:wrap;padding:8px 12px}.ledline{order:3;flex-basis:100%;flex-wrap:wrap}.authority{display:grid}.detailpane{display:block}}
</style>
