import { readWiki, countBy } from '$lib/wiki';

const GTD = process.env.GTDWIKI_PATH || '/data/gtdwiki';
const LLM = process.env.LLMWIKI_PATH || '/data/llmwiki';
const HINDSIGHT_API = process.env.HINDSIGHT_API_URL || 'http://hindsight-api.hindsight.svc.cluster.local:8888';
const HINDSIGHT_UI = process.env.HINDSIGHT_UI_URL || 'http://hindsight-control-plane.hindsight.svc.cluster.local:3000';
const HERMES_UI = process.env.HERMES_UI_URL || 'http://hermes.services.svc.cluster.local:9119';
const OLLAMA_TARGETS = (process.env.OLLAMA_TARGETS || 'umi=http://host.docker.internal:11434,fuu=http://192.168.200.67:11434')
  .split(',')
  .map((entry) => {
    const [id, ...rest] = entry.split('=');
    return { id: id.trim(), url: rest.join('=').trim() };
  })
  .filter((entry) => entry.id && entry.url);
const LLMWIKI_URL = process.env.LLMWIKI_URL || 'http://hairem-hub.services.svc.cluster.local:4567/Home';
const GTDWIKI_URL = process.env.GTDWIKI_URL || 'http://hairem-hub.services.svc.cluster.local:4568/Home';
const LAN = process.env.HAIREM_LAN_BASE || 'http://192.168.199.119';

type Led = 'ok' | 'warn' | 'down' | 'unknown';
type Component = { id: string; label: string; led: Led; detail: string; detailLines: string[] };

async function probe(url: string, ok: (status: number, text: string) => boolean = (s) => s >= 200 && s < 400) {
  try {
    const res = await fetch(url, { signal: AbortSignal.timeout(2500), redirect: 'follow' });
    const text = await res.text().catch(() => '');
    return { ok: ok(res.status, text), status: res.status, text };
  } catch (error) {
    return { ok: false, status: 0, text: error instanceof Error ? error.message : String(error) };
  }
}

async function jsonProbe<T>(url: string): Promise<{ ok: boolean; status: number; data: T | null; error?: string }> {
  try {
    const res = await fetch(url, { signal: AbortSignal.timeout(3500), redirect: 'follow' });
    const text = await res.text();
    return { ok: res.ok, status: res.status, data: text ? JSON.parse(text) as T : null };
  } catch (error) {
    return { ok: false, status: 0, data: null, error: error instanceof Error ? error.message : String(error) };
  }
}

type Bank = { bank_id: string; name?: string; fact_count?: number };
type BanksResponse = { banks?: Bank[]; total?: number };
type HindsightHealth = { status?: string; database?: string; db_pool_waiting?: number; db_pool_in_use?: number; db_pool_idle?: number };
type HindsightVersion = { api_version?: string; features?: Record<string, boolean> };
type BankStats = { bank_id: string; total_nodes?: number; total_documents?: number; pending_operations?: number; failed_operations?: number; pending_consolidation?: number; failed_consolidation?: number; total_observations?: number; operations_by_status?: Record<string, number> };
type LlmStats = { buckets?: Array<{ statuses?: Record<string, number>; total?: number; tokens?: { total?: number } }> };
type LlmRequest = { id?: string; operation?: string; status?: string; started_at?: string; ended_at?: string; provider?: string; model?: string; error?: string };
type LlmRequests = { total?: number; items?: LlmRequest[]; requests?: LlmRequest[]; data?: LlmRequest[] };
type AsyncOperation = { id?: string; operation_id?: string; task_type?: string; operation_type?: string; status?: string; created_at?: string; updated_at?: string; retry_count?: number; progress?: Record<string, unknown>; error_message?: string | null };
type OperationsResponse = { total?: number; operations?: AsyncOperation[] };

type OllamaVersion = { version?: string };
type OllamaTags = { models?: Array<{ name: string; details?: { parameter_size?: string; quantization_level?: string; context_length?: number } }> };
type OllamaPs = { models?: Array<{ name: string; size_vram?: number; expires_at?: string }> };

type OpenAiModels = { data?: Array<{ id: string; owned_by?: string }> };
type ProxyHealth = { status?: string; model?: string; dimensions?: number };
type LlmRuntime = {
  id: string;
  url: string;
  led: Led;
  detail: string;
  version: string | null;
  modelCount: number;
  loadedCount: number;
  models: Array<{ name: string; size?: string; quant?: string; ctx?: number }>;
  loaded: Array<{ name: string; vram?: number }>;
  apiMode: string;
};

function isoHoursAgo(hours: number) {
  return new Date(Date.now() - hours * 60 * 60 * 1000).toISOString();
}

function bucketTotals(stats: LlmStats | null | undefined) {
  const buckets = stats?.buckets || [];
  return {
    total: buckets.reduce((n, b) => n + (b.total || 0), 0),
    errors: buckets.reduce((n, b) => n + (b.statuses?.error || 0), 0),
    success: buckets.reduce((n, b) => n + (b.statuses?.success || 0), 0),
    tokens: buckets.reduce((n, b) => n + (b.tokens?.total || 0), 0)
  };
}

function requestItems(data: LlmRequests | null | undefined) {
  return data?.items || data?.requests || data?.data || [];
}

function ageLabel(iso?: string) {
  if (!iso) return 'n/a';
  const ms = Date.now() - new Date(iso).getTime();
  if (!Number.isFinite(ms) || ms < 0) return iso;
  const minutes = Math.floor(ms / 60000);
  if (minutes < 90) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 48) return `${hours}h`;
  return `${Math.floor(hours / 24)}d`;
}

function opName(op: AsyncOperation) {
  return op.task_type || op.operation_type || 'operation';
}

function opId(op: AsyncOperation) {
  return op.id || op.operation_id || '?';
}

function conciseError(error?: string) {
  if (!error) return 'none';
  return error
    .replace(/\s+/g, ' ')
    .replace(/^Error:\s*/i, '')
    .replace(/^Ollama request failed:\s*/i, '')
    .slice(0, 170);
}

function hindsightLabel(args: { pending: number; failed: number; processing: number; stuck: number; recentErrors: number }) {
  const parts: string[] = [];
  if (args.stuck > 0) parts.push(`${args.stuck} stuck`);
  if (args.failed > 0) parts.push(`${args.failed} failed`);
  if (args.recentErrors > 0) parts.push(`${args.recentErrors} err/h`);
  if (parts.length) return `hindsight ${parts.slice(0, 2).join(' · ')}`;
  if (args.processing > 0) return `hindsight ${args.processing} running`;
  if (args.pending > 0) return `hindsight ${args.pending} pending`;
  return 'hindsight ok';
}

function shortModelName(name: string) {
  return name.replace(':latest', '').replace(':9b-q5-16k', '').replace(/^hf\.co\//, '');
}

async function ollamaRuntime(target: { id: string; url: string }): Promise<LlmRuntime> {
  const [version, tags, ps, openaiModels, health] = await Promise.all([
    jsonProbe<OllamaVersion>(`${target.url}/api/version`),
    jsonProbe<OllamaTags>(`${target.url}/api/tags`),
    jsonProbe<OllamaPs>(`${target.url}/api/ps`),
    jsonProbe<OpenAiModels>(`${target.url}/v1/models`),
    jsonProbe<ProxyHealth>(`${target.url}/health`)
  ]);

  const ollamaModels = tags.data?.models || [];
  const loaded = ps.data?.models || [];
  const v1Models = openaiModels.data?.data || [];
  const isOllama = version.ok || tags.ok || ps.ok;
  const isOpenAi = openaiModels.ok || health.ok;
  const led: Led = isOllama
    ? (version.ok && tags.ok ? 'ok' : 'warn')
    : (isOpenAi ? 'ok' : 'down');
  const apiMode = isOllama && isOpenAi ? 'ollama+/v1' : (isOllama ? 'ollama' : (isOpenAi ? 'openai/proxy' : 'unreachable'));

  if (isOllama) {
    const models = ollamaModels.length
      ? ollamaModels.map((m) => ({ name: m.name, size: m.details?.parameter_size, quant: m.details?.quantization_level, ctx: m.details?.context_length }))
      : v1Models.map((m) => ({ name: m.id, size: m.owned_by || 'openai-api', quant: '', ctx: undefined }));
    return {
      id: target.id,
      url: target.url,
      led,
      detail: version.data?.version ? `v${version.data.version} · ${apiMode} · loaded ${loaded.length}/${models.length} · ${loaded.map((m) => m.name).join(', ') || 'idle'} · ${target.url}` : `http ${version.status || tags.status || openaiModels.status} · ${apiMode} · ${target.url}`,
      version: version.data?.version || null,
      modelCount: models.length,
      loadedCount: loaded.length,
      models: models.slice(0, 8),
      loaded: loaded.map((m) => ({ name: m.name, vram: m.size_vram })),
      apiMode
    };
  }

  const modelName = health.data?.model || v1Models[0]?.id || 'proxy';
  return {
    id: target.id,
    url: target.url,
    led,
    detail: health.ok ? `${modelName} · ${health.data?.dimensions || '?'}d · ${apiMode} · ${target.url}` : health.error || openaiModels.error || `http ${health.status || openaiModels.status} · ${target.url}`,
    version: null,
    modelCount: v1Models.length,
    loadedCount: health.ok || openaiModels.ok ? 1 : 0,
    models: v1Models.map((m) => ({ name: m.id, size: m.owned_by || 'openai-api', quant: '', ctx: undefined })).slice(0, 8),
    loaded: health.ok ? [{ name: modelName, vram: undefined }] : [],
    apiMode
  };
}

async function runtime() {
  const [hHealth, hVersion, banksRes, hUi, llmwikiHttp, gtdwikiHttp, hermesUi, llms] = await Promise.all([
    jsonProbe<HindsightHealth>(`${HINDSIGHT_API}/health`),
    jsonProbe<HindsightVersion>(`${HINDSIGHT_API}/version`),
    jsonProbe<BanksResponse>(`${HINDSIGHT_API}/v1/default/banks`),
    probe(HINDSIGHT_UI),
    probe(LLMWIKI_URL),
    probe(GTDWIKI_URL),
    probe(HERMES_UI, (s, text) => s < 500 && (text.includes('Hermes Agent') || text.includes('Sign in') || text.length > 0)),
    Promise.all(OLLAMA_TARGETS.map(ollamaRuntime))
  ]);

  const banks = banksRes.data?.banks || [];
  const oneHourAgo = isoHoursAgo(1);
  const bankDetails = await Promise.all(banks.map(async (bank) => {
    const bankId = encodeURIComponent(bank.bank_id);
    const [stats, llm7d, llm24h, err1h, lastErr, processingOps] = await Promise.all([
      jsonProbe<BankStats>(`${HINDSIGHT_API}/v1/default/banks/${bankId}/stats`),
      jsonProbe<LlmStats>(`${HINDSIGHT_API}/v1/default/banks/${bankId}/llm-requests/stats?period=7d`),
      jsonProbe<LlmStats>(`${HINDSIGHT_API}/v1/default/banks/${bankId}/llm-requests/stats?period=1d`),
      jsonProbe<LlmRequests>(`${HINDSIGHT_API}/v1/default/banks/${bankId}/llm-requests?status=error&start_date=${encodeURIComponent(oneHourAgo)}&limit=1`),
      jsonProbe<LlmRequests>(`${HINDSIGHT_API}/v1/default/banks/${bankId}/llm-requests?status=error&limit=1`),
      jsonProbe<OperationsResponse>(`${HINDSIGHT_API}/v1/default/banks/${bankId}/operations?status=processing&limit=100`)
    ]);
    const ops = processingOps.data?.operations || [];
    const stuckOps = ops.filter((op) => Date.now() - new Date(op.updated_at || op.created_at || 0).getTime() > 60 * 60 * 1000);
    const lastError = requestItems(lastErr.data)[0];
    const totals7d = bucketTotals(llm7d.data);
    const totals24h = bucketTotals(llm24h.data);
    return {
      ...bank,
      stats: stats.data,
      llm: {
        total: totals7d.total,
        errors: totals7d.errors,
        success: totals7d.success,
        tokens7d: totals7d.tokens,
        errors24h: totals24h.errors,
        success24h: totals24h.success,
        total24h: totals24h.total,
        errors1h: err1h.data?.total ?? requestItems(err1h.data).length,
        lastErrorAt: lastError?.started_at,
        lastErrorAge: ageLabel(lastError?.started_at),
        lastError: conciseError(lastError?.error || `${lastError?.operation || 'llm'} ${lastError?.model || ''}`.trim())
      },
      operations: { processing: ops, stuck: stuckOps }
    };
  }));

  const pending = bankDetails.reduce((n, b) => n + (b.stats?.pending_consolidation || 0), 0);
  const failed = bankDetails.reduce((n, b) => n + (b.stats?.failed_consolidation || 0), 0);
  const processing = bankDetails.reduce((n, b) => n + (b.stats?.operations_by_status?.processing || 0), 0);
  const stuck = bankDetails.reduce((n, b) => n + b.operations.stuck.length, 0);
  const llmErrors = bankDetails.reduce((n, b) => n + b.llm.errors, 0);
  const llmErrors24h = bankDetails.reduce((n, b) => n + b.llm.errors24h, 0);
  const llmErrors1h = bankDetails.reduce((n, b) => n + b.llm.errors1h, 0);
  const lastErrorAges = bankDetails.map((b) => b.llm.lastErrorAt).filter(Boolean).sort().reverse();
  const lastErrorAge = ageLabel(lastErrorAges[0]);

  const bankLines = bankDetails.flatMap((b) => [
    `${b.name || b.bank_id}`,
    `  facts              ${b.stats?.total_nodes || b.fact_count || 0}`,
    `  pending ops        ${b.stats?.pending_consolidation || 0}`,
    `  failed ops         ${b.stats?.failed_consolidation || 0}`,
    `  running ops        ${b.stats?.operations_by_status?.processing || 0}`,
    `  stuck ops          ${b.operations.stuck.length}`,
    `  llm errors         1h ${b.llm.errors1h} · 24h ${b.llm.errors24h} · 7d ${b.llm.errors}`,
    `  last error age     ${b.llm.lastErrorAge}`,
    `  last error summary ${b.llm.lastError || 'none'}`,
    `  stuck details      ${b.operations.stuck.map((op) => `${opName(op)} ${opId(op).slice(0, 8)} age ${ageLabel(op.updated_at || op.created_at)}`).join('; ') || 'none'}`
  ]);
  const hindsightLines = hHealth.data?.database ? [
    `API            ${hVersion.data?.api_version || '?'}`,
    `DB             ${hHealth.data.database}`,
    `Pool           wait ${hHealth.data.db_pool_waiting || 0} · use ${hHealth.data.db_pool_in_use || 0} · idle ${hHealth.data.db_pool_idle || 0}`,
    '',
    ...bankLines
  ] : [hHealth.error || `http ${hHealth.status}`];
  const hindsightLed: Led = !hHealth.ok || hHealth.data?.status !== 'healthy' ? 'down' : (failed > 0 || stuck > 0 || llmErrors1h > 0 ? 'warn' : 'ok');
  const components: Component[] = [
    { id: 'hindsight', label: hindsightLabel({ pending, failed, processing, stuck, recentErrors: llmErrors1h }), led: hindsightLed, detail: hindsightLines.join('\\n'), detailLines: hindsightLines },
    ...llms.map((llm) => {
      const activeLines = llm.loaded.map((m) => `  ${m.name}${m.vram ? ` · vram ${Math.round(m.vram / 1024 / 1024 / 1024)}GiB` : ''}`);
      const activeLabel = llm.loaded.map((m) => shortModelName(m.name)).slice(0, 1).join(', ');
      const lines = [
        `Endpoint       ${llm.url}`,
        `API mode       ${llm.apiMode}`,
        `Version        ${llm.version || 'proxy/health'}`,
        `Active models  ${llm.loadedCount}`,
        `Available      ${llm.modelCount}`,
        '',
        'Active only',
        ...(activeLines.length ? activeLines : ['  idle'])
      ];
      return { id: `llm-${llm.id}`, label: `llm:${llm.id} ${activeLabel || 'idle'}`, led: llm.led, detail: lines.join('\\n'), detailLines: lines };
    }),
    { id: 'llmwiki', label: `llmwiki ${llmwikiHttp.status || ''}`, led: llmwikiHttp.ok ? 'ok' : 'down', detail: llmwikiHttp.ok ? `Gollum 4567\\n${LLMWIKI_URL}` : llmwikiHttp.text.slice(0, 120), detailLines: llmwikiHttp.ok ? ['Gollum         4567', `URL            ${LLMWIKI_URL}`] : [llmwikiHttp.text.slice(0, 120)] },
    { id: 'gtdwiki', label: `gtdwiki ${gtdwikiHttp.status || ''}`, led: gtdwikiHttp.ok ? 'ok' : 'down', detail: gtdwikiHttp.ok ? `Gollum 4568\\n${GTDWIKI_URL}` : gtdwikiHttp.text.slice(0, 120), detailLines: gtdwikiHttp.ok ? ['Gollum         4568', `URL            ${GTDWIKI_URL}`] : [gtdwikiHttp.text.slice(0, 120)] },
    { id: 'hindsight-ui', label: `hindsight-ui ${hUi.status || ''}`, led: hUi.ok ? 'ok' : 'down', detail: hUi.ok ? `Control plane\\n${HINDSIGHT_UI}` : hUi.text.slice(0, 120), detailLines: hUi.ok ? ['Control plane', `URL            ${HINDSIGHT_UI}`] : [hUi.text.slice(0, 120)] },
    { id: 'hermes', label: `hermes ${hermesUi.status || ''}`, led: hermesUi.ok ? 'ok' : 'down', detail: hermesUi.ok ? `Hermes UI/auth 9119\\n${HERMES_UI}` : hermesUi.text.slice(0, 120), detailLines: hermesUi.ok ? ['Hermes UI/auth 9119', `URL            ${HERMES_UI}`] : [hermesUi.text.slice(0, 120)] }
  ];

  return {
    components,
    hindsight: {
      apiVersion: hVersion.data?.api_version || 'n/a',
      banks: bankDetails,
      backlog: { pending, failed, processing, stuck },
      pool: hHealth.data ? { waiting: hHealth.data.db_pool_waiting || 0, inUse: hHealth.data.db_pool_in_use || 0, idle: hHealth.data.db_pool_idle || 0 } : null,
      llm: { total: bankDetails.reduce((n, b) => n + b.llm.total, 0), success: bankDetails.reduce((n, b) => n + b.llm.success, 0), errors: llmErrors, errors24h: llmErrors24h, errors1h: llmErrors1h, lastErrorAge }
    },
    llms
  };
}

export async function load() {
  const [gtdItems, llmItems, rt] = await Promise.all([
    readWiki(GTD, 120).catch((error) => ({ error: String(error), items: [] })),
    readWiki(LLM, 60).catch((error) => ({ error: String(error), items: [] })),
    runtime()
  ]);

  const gtd = Array.isArray(gtdItems) ? gtdItems : gtdItems.items;
  const llm = Array.isArray(llmItems) ? llmItems : llmItems.items;

  return {
    paths: { gtd: GTD, llm: LLM },
    links: { llmwiki: `${LAN}:4567/Home`, gtdwiki: `${LAN}:4568/Home`, hindsight: `${LAN}:8888/`, hermes: `${LAN}:9119/` },
    gtd,
    llm,
    counts: { gtdTotal: gtd.length, llmTotal: llm.length, gtdByStatus: countBy(gtd, 'status'), gtdByOwner: countBy(gtd, 'owner') },
    runtime: rt,
    errors: { gtd: Array.isArray(gtdItems) ? null : gtdItems.error, llm: Array.isArray(llmItems) ? null : llmItems.error }
  };
}
