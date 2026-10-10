import { useEffect, useMemo, useRef, useState } from "react";
import {
  AssistantRuntimeProvider, AuiIf, ComposerPrimitive, MessagePrimitive,
  ThreadPrimitive, useAui, useAuiState, useLocalRuntime,
  type ChatModelAdapter,
} from "@assistant-ui/react";
import { createFabricThread, listAvailableAgents, listAccessibleTenants, streamFabricTurn } from "./api.mjs";

type Agent = { agentKey: string; label: string };
type Tenant = {
  tenantKey: string; displayName: string;
  capabilities: { chat: boolean; tenantAdmin: boolean };
};

function ChatThread({ agent, toolStatus }: { agent: Agent; toolStatus: string }) {
  const aui = useAui();
  const running = useAuiState((s) => s.thread.isRunning);
  return <ThreadPrimitive.Root className="chat">
    <ThreadPrimitive.Viewport className="messages">
      <AuiIf condition={(s) => s.thread.isEmpty}>
        <div className="empty"><h2>Bonjour !</h2><p>Vous discutez avec {agent.label}. Envoyez un message pour commencer.</p></div>
      </AuiIf>
      <ThreadPrimitive.Messages>
        {({ message }) => <MessagePrimitive.Root className={message.role === "user" ? "message from-user" : "message from-agent"}>
          <div className="message-label">{message.role === "user" ? "Vous" : agent.label}</div>
          <MessagePrimitive.Parts />
        </MessagePrimitive.Root>}
      </ThreadPrimitive.Messages>
      <ThreadPrimitive.ViewportFooter className="composer-footer">
        <div className="tool-status" role="status">{toolStatus}</div>
        <ComposerPrimitive.Root className="composer">
          <ComposerPrimitive.Input placeholder={`Écrire à ${agent.label}…`} rows={2} aria-label="Votre message" />
          <div className="composer-actions">
            <button type="button" disabled={!running} onClick={() => aui.thread().cancelRun()}>Arrêter</button>
            <ComposerPrimitive.Send>Envoyer</ComposerPrimitive.Send>
          </div>
        </ComposerPrimitive.Root>
      </ThreadPrimitive.ViewportFooter>
    </ThreadPrimitive.Viewport>
  </ThreadPrimitive.Root>;
}

/* One mounted runtime per agent: switching agent A -> B -> A does not mix
   their local transcripts. Long-term history will later use Fabric-owned
   persisted thread/message APIs, not browser localStorage. */
function AgentPanel({ tenantKey, agent, active }: { tenantKey: string; agent: Agent; active: boolean }) {
  const thread = useRef<string | null>(null);
  const [toolStatus, setToolStatus] = useState("");
  const adapter = useMemo<ChatModelAdapter>(() => ({
    async *run({ messages, abortSignal }) {
      const last = [...messages].reverse().find((m) => m.role === "user");
      const input = last?.content.filter((p) => p.type === "text").map((p) => p.text).join("\n").trim();
      if (!input) throw new Error("Message vide");
      setToolStatus("");
      const fabricThreadId = thread.current ??
        (thread.current = await createFabricThread(tenantKey, agent.agentKey, { signal: abortSignal }));
      let text = "";
      try {
        for await (const item of streamFabricTurn(tenantKey, fabricThreadId, input, { signal: abortSignal })) {
          if (item.type === "text-delta") {
            text += item.text;
            yield { content: [{ type: "text", text }] };
          } else if (item.type === "tool-started") {
            setToolStatus(`Outil en cours : ${item.tool}`);
          } else if (item.type === "tool-completed") {
            setToolStatus(`Outil terminé : ${item.tool}`);
          }
        }
        setToolStatus("");
        yield { content: [{ type: "text", text }] };
      } catch (error) {
        setToolStatus(error instanceof Error ? error.message : "Conversation indisponible");
        throw error;
      }
    },
  }), [tenantKey, agent.agentKey]);
  const runtime = useLocalRuntime(adapter);
  return <section className="chat-panel" style={{ display: active ? "flex" : "none" }} aria-label={`Conversation avec ${agent.label}`}>
    <AssistantRuntimeProvider runtime={runtime}>
      <ChatThread agent={agent} toolStatus={toolStatus} />
    </AssistantRuntimeProvider>
  </section>;
}

/**
 * TenantSpace remounts on tenant selection, so transient assistant-ui state,
 * Hermes thread IDs and user drafts can NEVER be reused across tenants.
 * The server still authorizes every request independently from this UI.
 */
function TenantSpace({ tenant }: { tenant: Tenant }) {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [active, setActive] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  useEffect(() => {
    const abort = new AbortController();
    if (!tenant.capabilities.chat) {
      setLoading(false);
      return () => abort.abort();
    }
    listAvailableAgents(tenant.tenantKey).then((list) => {
      if (abort.signal.aborted) return;
      setAgents(list);
      setActive(list[0]?.agentKey ?? null);
    }).catch(() => {
      if (!abort.signal.aborted)
        setError("Impossible de charger vos agents. Vérifiez votre connexion et vos droits.");
    }).finally(() => { if (!abort.signal.aborted) setLoading(false); });
    return () => abort.abort();
  }, [tenant.tenantKey, tenant.capabilities.chat]);

  return <div className="tenant-layout">
    <div className="agents-column">
      <strong className="section-heading">Agents de {tenant.displayName}</strong>
      <nav aria-label={`Agents de ${tenant.displayName}`}>
        {agents.map((agent) => <button key={agent.agentKey} type="button"
          className={active === agent.agentKey ? "agent-choice selected" : "agent-choice"}
          aria-current={active === agent.agentKey ? "page" : undefined}
          onClick={() => setActive(agent.agentKey)}>
          <span className="agent-icon">✧</span>{agent.label}
        </button>)}
        {loading && <p className="hint">Chargement des agents…</p>}
        {error && <p className="error" role="alert">{error}</p>}
        {!loading && !error && agents.length === 0 &&
          <p className="hint">Aucun agent autorisé dans cet espace.</p>}
      </nav>
      <div className="sidebar-bottom">
        <span>Paramètres personnels — en préparation</span>
        {tenant.capabilities.tenantAdmin &&
          <span>Administration de {tenant.displayName} — en préparation</span>}
      </div>
    </div>
    <section className="workspace">
      <header>
        <div>
          <strong>{agents.find(a => a.agentKey === active)?.label ?? tenant.displayName}</strong>
          <small>Espace {tenant.displayName} · Conversation privée via Fabric</small>
        </div>
        <span className="privacy">Accès contrôlé par Fabric</span>
      </header>
      {agents.map(agent => <AgentPanel
        key={`${tenant.tenantKey}/${agent.agentKey}`} tenantKey={tenant.tenantKey}
        agent={agent} active={agent.agentKey === active} />)}
      {active === null && <div className="waiting">Sélectionnez un agent autorisé pour commencer.</div>}
    </section>
  </div>;
}

export function App() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [platformAdmin, setPlatformAdmin] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const abort = new AbortController();
    listAccessibleTenants().then(({ tenants: allowed, platformAdmin: isAdmin }) => {
      if (abort.signal.aborted) return;
      setTenants(allowed);
      setPlatformAdmin(isAdmin);
      setSelected(allowed[0]?.tenantKey ?? null);
    }).catch(() => {
      if (!abort.signal.aborted)
        setError("Connexion requise ou service indisponible. L'accès dépend de vos droits Fabric.");
    }).finally(() => { if (!abort.signal.aborted) setLoading(false); });
    return () => abort.abort();
  }, []);

  const tenant = tenants.find(t => t.tenantKey === selected) ?? null;
  return <main className="layout">
    <aside className="sidebar">
      <div className="brand">
        <span className="brand-mark">✦</span>
        <div><strong>TXO Fabric</strong><small>Votre espace de travail</small></div>
      </div>
      <nav aria-label="Espaces autorisés">
        <strong>Mes espaces</strong>
        {tenants.map(t => <button key={t.tenantKey} type="button"
          className={selected === t.tenantKey ? "agent-choice selected" : "agent-choice"}
          aria-current={selected === t.tenantKey ? "page" : undefined}
          onClick={() => setSelected(t.tenantKey)}>
          <span className="agent-icon">◈</span>{t.displayName}
        </button>)}
        {loading && <p className="hint">Chargement de vos espaces…</p>}
        {error && <p className="error" role="alert">{error}</p>}
        {!loading && !error && tenants.length === 0 &&
          <p className="hint">Aucun espace autorisé pour ce compte.</p>}
      </nav>
      <div className="sidebar-bottom">
        <span>webui.truxonline.com</span>
        {platformAdmin && <span>Administration Fabric — en préparation</span>}
      </div>
    </aside>
    {tenant ? <TenantSpace key={tenant.tenantKey} tenant={tenant} /> :
      <section className="workspace">
        <header><div><strong>TXO Fabric</strong><small>Portail global</small></div></header>
        <div className="waiting">
          {loading ? "Identification de vos espaces…" : error || "Aucun espace disponible."}
        </div>
      </section>}
  </main>;
}
