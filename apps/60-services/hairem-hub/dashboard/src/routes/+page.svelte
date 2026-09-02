<script lang="ts">
  let { data } = $props();

  const statusOrder = ['next', 'inbox', 'waiting', 'blocked', 'scheduled', 'someday', 'done', 'none'];
  const byStatus = (status: string) => data.gtd.filter((item) => (item.status || 'none') === status);
  const activeStatuses = statusOrder.filter((status) => byStatus(status).length > 0);
</script>

<svelte:head>
  <title>hAIrem Dashboard</title>
  <meta name="description" content="Cockpit hAIrem pour GTDWiki, LLMWiki et runtime agents" />
</svelte:head>

<main class="shell">
  <section class="hero panel">
    <div>
      <p class="eyebrow">hAIrem cockpit · SvelteKit v0.1</p>
      <h1>Dashboard opérationnel</h1>
      <p class="lead">Vue directe sur le GTDWiki canonique et le LLMWiki documentaire. Pas de base parallèle. Pas de duplication de vérité.</p>
    </div>
    <div class="pulse-card">
      <span class="heart"></span>
      <strong>STATUS</strong>
      <p>Dashboard dynamique actif</p>
      <small>{data.generatedAt}</small>
    </div>
  </section>

  <section class="grid metrics">
    <article class="panel metric"><span>GTD items</span><strong>{data.counts.gtdTotal}</strong><small>{data.paths.gtd}</small></article>
    <article class="panel metric"><span>LLMWiki pages</span><strong>{data.counts.llmTotal}</strong><small>{data.paths.llm}</small></article>
    <article class="panel metric"><span>Owner Katia</span><strong>{data.counts.gtdByOwner.katia || 0}</strong><small>mutations GTD owner-mediated</small></article>
    <article class="panel metric"><span>Health</span><strong>OK</strong><small>/api/health</small></article>
  </section>

  <section class="grid two">
    <article class="panel">
      <div class="section-title">
        <h2>GTDWiki</h2>
        <a href={data.links.gtdwiki}>ouvrir Gollum</a>
      </div>
      {#if data.errors.gtd}<p class="error">{data.errors.gtd}</p>{/if}
      <div class="kanban">
        {#each activeStatuses as status}
          <div class="lane">
            <h3>{status} <span>{byStatus(status).length}</span></h3>
            {#each byStatus(status).slice(0, 8) as item}
              <article class="card">
                <strong>{item.title}</strong>
                <code>{item.path}</code>
                {#if item.excerpt}<p>{item.excerpt}</p>{/if}
                <footer>
                  {#if item.owner}<span>{item.owner}</span>{/if}
                  {#if item.priority}<span>{item.priority}</span>{/if}
                  {#if item.context}<span>{item.context}</span>{/if}
                </footer>
              </article>
            {/each}
          </div>
        {/each}
      </div>
    </article>

    <article class="panel">
      <div class="section-title">
        <h2>LLMWiki</h2>
        <a href={data.links.llmwiki}>ouvrir Gollum</a>
      </div>
      {#if data.errors.llm}<p class="error">{data.errors.llm}</p>{/if}
      <div class="doc-list">
        {#each data.llm.slice(0, 12) as item}
          <article class="doc">
            <strong>{item.title}</strong>
            <code>{item.path}</code>
            {#if item.excerpt}<p>{item.excerpt}</p>{/if}
          </article>
        {/each}
      </div>
    </article>
  </section>

  <section class="panel roadmap">
    <h2>Prochaine couche</h2>
    <ul>
      <li>Endpoints serveur pour health Kubernetes, Hindsight, Hermes et Ollama.</li>
      <li>Commandes allowlistées vers Katia via webhook, avec audit JSONL.</li>
      <li>Édition GTD verrouillée/atomique, ou mutation Katia-first selon mode choisi.</li>
    </ul>
  </section>
</main>

<style>
  :global(body) {
    margin: 0;
    background: radial-gradient(circle at top left, #12324a 0, transparent 28rem), #05070d;
    color: #e6f7ff;
    font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  }
  :global(a) { color: #6ee7ff; text-decoration: none; }
  .shell { max-width: 1440px; margin: 0 auto; padding: 32px; }
  .panel { background: rgba(9, 18, 31, .78); border: 1px solid rgba(110, 231, 255, .18); border-radius: 24px; box-shadow: 0 24px 80px rgba(0,0,0,.35), inset 0 1px 0 rgba(255,255,255,.04); }
  .hero { display: flex; justify-content: space-between; gap: 24px; padding: 32px; align-items: center; }
  .eyebrow { color: #67e8f9; text-transform: uppercase; letter-spacing: .16em; font-size: .78rem; }
  h1 { font-size: clamp(2.4rem, 5vw, 5.5rem); line-height: .9; margin: 0; }
  h2 { margin: 0; }
  .lead { color: #a7c7d9; max-width: 760px; font-size: 1.15rem; }
  .pulse-card { min-width: 260px; padding: 24px; border-radius: 20px; background: linear-gradient(135deg, rgba(14,165,233,.2), rgba(248, 24, 88, .12)); }
  .heart { display: inline-block; width: 18px; height: 18px; background: #ff174d; border-radius: 50%; box-shadow: 0 0 24px #ff174d; animation: beat 1.1s infinite; }
  @keyframes beat { 50% { transform: scale(1.28); } }
  .grid { display: grid; gap: 18px; margin-top: 18px; }
  .metrics { grid-template-columns: repeat(4, minmax(0, 1fr)); }
  .two { grid-template-columns: minmax(0, 1.35fr) minmax(360px, .65fr); align-items: start; }
  .metric { padding: 20px; }
  .metric span, small { color: #8cb4c9; }
  .metric strong { display: block; margin: 10px 0; font-size: 2.2rem; color: #fff; }
  .section-title { display: flex; justify-content: space-between; gap: 16px; align-items: center; padding: 22px 24px 0; }
  .kanban { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 14px; padding: 24px; }
  .lane { min-height: 220px; padding: 14px; border-radius: 18px; background: rgba(1, 8, 15, .55); border: 1px solid rgba(255,255,255,.06); }
  .lane h3 { margin: 0 0 12px; color: #67e8f9; text-transform: uppercase; font-size: .86rem; letter-spacing: .08em; }
  .lane h3 span { float: right; color: #ff5d85; }
  .card, .doc { padding: 14px; margin-bottom: 12px; border-radius: 14px; background: rgba(15, 27, 44, .9); border: 1px solid rgba(110,231,255,.12); }
  .card strong, .doc strong { display: block; color: #fff; }
  code { display: inline-block; margin: 8px 0; color: #7dd3fc; font-size: .78rem; word-break: break-word; }
  p { color: #b8ceda; }
  footer { display: flex; flex-wrap: wrap; gap: 6px; }
  footer span { padding: 3px 8px; border-radius: 999px; background: rgba(103,232,249,.1); color: #9beafe; font-size: .72rem; }
  .doc-list { padding: 24px; }
  .roadmap { margin-top: 18px; padding: 24px; }
  .error { color: #ff8aa8; }
  @media (max-width: 900px) { .hero, .two { grid-template-columns: 1fr; display: grid; } .metrics { grid-template-columns: repeat(2, 1fr); } }
</style>
