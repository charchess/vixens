import { readWiki, countBy } from '$lib/wiki';

const GTD = process.env.GTDWIKI_PATH || '/data/gtdwiki';
const LLM = process.env.LLMWIKI_PATH || '/data/llmwiki';
const HINDSIGHT_API = process.env.HINDSIGHT_API_URL || 'http://hindsight-api.hindsight.svc.cluster.local:8888';
const HINDSIGHT_UI = process.env.HINDSIGHT_UI_URL || 'http://hindsight-control-plane.hindsight.svc.cluster.local:3000';
const HERMES_UI = process.env.HERMES_UI_URL || 'http://hermes.services.svc.cluster.local:9119';
const OLLAMA = process.env.OLLAMA_URL || 'http://host.docker.internal:11434';
const LLMWIKI_URL = process.env.LLMWIKI_URL || 'http://hairem-hub.services.svc.cluster.local:4567/Home';
const GTDWIKI_URL = process.env.GTDWIKI_URL || 'http://hairem-hub.services.svc.cluster.local:4568/Home';
const LAN = process.env.HAIREM_LAN_BASE || 'http://192.168.199.119';

type Led = 'ok' | 'warn' | 'down' | 'unknown';
type Component = { id: string; label: string; led: Led; detail: string };

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
type OllamaVersion = { version?: string };
type OllamaTags = { models?: Array<{ name: string; details?: { parameter_size?: string; quantization_level?: string; context_length?: number } }> };
type OllamaPs = { models?: Array<{ name: string; size_vram?: number; expires_at?: string }> };

async function runtime() {
  const [hHealth, hVersion, banksRes, hUi, llmwikiHttp, gtdwikiHttp, hermesUi, ollamaVersion, ollamaTags, ollamaPs] = await Promise.all([
    jsonProbe<HindsightHealth>(`${HINDSIGHT_API}/health`),
    jsonProbe<HindsightVersion>(`${HINDSIGHT_API}/version`),
    jsonProbe<BanksResponse>(`${HINDSIGHT_API}/v1/default/banks`),
    probe(HINDSIGHT_UI),
    probe(LLMWIKI_URL),
    probe(GTDWIKI_URL),
    probe(HERMES_UI, (s, text) => s < 500 && (text.includes('Hermes Agent') || text.includes('Sign in') || text.length > 0)),
    jsonProbe<OllamaVersion>(`${OLLAMA}/api/version`),
    jsonProbe<OllamaTags>(`${OLLAMA}/api/tags`),
    jsonProbe<OllamaPs>(`${OLLAMA}/api/ps`)
  ]);

  const banks = banksRes.data?.banks || [];
  const bankDetails = await Promise.all(banks.map(async (bank) => {
    const [stats, llm] = await Promise.all([
      jsonProbe<BankStats>(`${HINDSIGHT_API}/v1/default/banks/${encodeURIComponent(bank.bank_id)}/stats`),
      jsonProbe<LlmStats>(`${HINDSIGHT_API}/v1/default/banks/${encodeURIComponent(bank.bank_id)}/llm-requests/stats`)
    ]);
    const buckets = llm.data?.buckets || [];
    const llmTotal = buckets.reduce((n, b) => n + (b.total || 0), 0);
    const llmErrors = buckets.reduce((n, b) => n + (b.statuses?.error || 0), 0);
    const llmSuccess = buckets.reduce((n, b) => n + (b.statuses?.success || 0), 0);
    return { ...bank, stats: stats.data, llm: { total: llmTotal, errors: llmErrors, success: llmSuccess } };
  }));

  const pending = bankDetails.reduce((n, b) => n + (b.stats?.pending_consolidation || 0), 0);
  const failed = bankDetails.reduce((n, b) => n + (b.stats?.failed_consolidation || 0), 0);
  const processing = bankDetails.reduce((n, b) => n + (b.stats?.operations_by_status?.processing || 0), 0);
  const llmErrors = bankDetails.reduce((n, b) => n + b.llm.errors, 0);

  const components: Component[] = [
    { id: 'hindsight', label: 'hindsight', led: hHealth.ok && hHealth.data?.status === 'healthy' ? 'ok' : 'down', detail: hHealth.data?.database ? `api · db ${hHealth.data.database}` : hHealth.error || `http ${hHealth.status}` },
    { id: 'hindsight-ui', label: 'hindsight-ui', led: hUi.ok ? 'ok' : 'down', detail: hUi.ok ? 'control-plane' : hUi.text.slice(0, 60) },
    { id: 'llm', label: 'llm', led: ollamaVersion.ok ? (ollamaPs.ok ? 'ok' : 'warn') : 'down', detail: ollamaVersion.data?.version ? `ollama ${ollamaVersion.data.version}` : ollamaVersion.error || `http ${ollamaVersion.status}` },
    { id: 'llmwiki', label: 'llmwiki', led: llmwikiHttp.ok ? 'ok' : 'down', detail: llmwikiHttp.ok ? 'gollum 4567' : llmwikiHttp.text.slice(0, 60) },
    { id: 'gtdwiki', label: 'gtdwiki', led: gtdwikiHttp.ok ? 'ok' : 'down', detail: gtdwikiHttp.ok ? 'gollum 4568' : gtdwikiHttp.text.slice(0, 60) },
    { id: 'hermes', label: 'hermes', led: hermesUi.ok ? 'ok' : 'down', detail: hermesUi.ok ? 'ui/auth 9119' : hermesUi.text.slice(0, 60) }
  ];

  const models = ollamaTags.data?.models || [];
  const loaded = ollamaPs.data?.models || [];

  return {
    components,
    hindsight: {
      apiVersion: hVersion.data?.api_version || 'n/a',
      banks: bankDetails,
      backlog: { pending, failed, processing },
      pool: hHealth.data ? { waiting: hHealth.data.db_pool_waiting || 0, inUse: hHealth.data.db_pool_in_use || 0, idle: hHealth.data.db_pool_idle || 0 } : null,
      llm: { total: bankDetails.reduce((n, b) => n + b.llm.total, 0), success: bankDetails.reduce((n, b) => n + b.llm.success, 0), errors: llmErrors }
    },
    llm: {
      version: ollamaVersion.data?.version || null,
      modelCount: models.length,
      loadedCount: loaded.length,
      models: models.slice(0, 6).map((m) => ({ name: m.name, size: m.details?.parameter_size, quant: m.details?.quantization_level, ctx: m.details?.context_length })),
      loaded: loaded.map((m) => ({ name: m.name, vram: m.size_vram }))
    }
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
