import { useEffect, useMemo, useRef, useState } from "react";
import {
  AssistantRuntimeProvider, AuiIf, ComposerPrimitive, MessagePrimitive,
  ThreadPrimitive, useAui, useAuiState, useLocalRuntime,
  type ChatModelAdapter,
} from "@assistant-ui/react";
import { createFabricThread, listAvailableAgents, streamFabricTurn } from "./api.mjs";

type Agent = { agentKey: string; label: string };

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
function AgentPanel({ agent, active }: { agent: Agent; active: boolean }) {
  const thread = useRef<string | null>(null);
  const [toolStatus, setToolStatus] = useState("");
  const adapter = useMemo<ChatModelAdapter>(() => ({
    async *run({ messages, abortSignal }) {
      const last = [...messages].reverse().find((m) => m.role === "user");
      const input = last?.content.filter((p) => p.type === "text").map((p) => p.text).join("\n").trim();
      if (!input) throw new Error("Message vide");
      setToolStatus("");
      if (!thread.current) thread.current = await createFabricThread(agent.agentKey, { signal: abortSignal });
      let text = "";
      try {
        for await (const item of streamFabricTurn(thread.current, input, { signal: abortSignal })) {
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
  }), [agent.agentKey]);
  const runtime = useLocalRuntime(adapter);
  return <section className="chat-panel" style={{ display: active ? "flex" : "none" }} aria-label={`Conversation avec ${agent.label}`}>
    <AssistantRuntimeProvider runtime={runtime}>
      <ChatThread agent={agent} toolStatus={toolStatus} />
    </AssistantRuntimeProvider>
  </section>;
}

export function App() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [active, setActive] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  useEffect(() => {
    const abort = new AbortController();
    // No hardcoded or browser-selected tenant. Server returns authorized agents.
    listAvailableAgents().then((list) => {
      if (abort.signal.aborted) return;
      setAgents(list);
      setActive(list[0]?.agentKey ?? null);
    }).catch(() => {
      if (!abort.signal.aborted) setError("Impossible de charger vos agents. Vérifiez votre connexion et vos droits.");
    }).finally(() => { if (!abort.signal.aborted) setLoading(false); });
    return () => abort.abort();
  }, []);
  return <main className="layout">
    <aside className="sidebar">
      <div className="brand"><span className="brand-mark">✦</span><div><strong>TXO Fabric</strong><small>Mon espace</small></div></div>
      <nav aria-label="Navigation principale"><strong>Mes agents</strong>
        {agents.map((agent) => <button key={agent.agentKey} type="button"
          className={active === agent.agentKey ? "agent-choice selected" : "agent-choice"}
          aria-current={active === agent.agentKey ? "page" : undefined}
          onClick={() => setActive(agent.agentKey)}>
          <span className="agent-icon">✧</span>{agent.label}
        </button>)}
        {loading && <p className="hint">Chargement des agents…</p>}
        {error && <p className="error" role="alert">{error}</p>}
        {!loading && !error && agents.length === 0 && <p className="hint">Aucun agent autorisé.</p>}
      </nav>
      <div className="sidebar-bottom"><span>Paramètres — bientôt</span><span>Administration — selon vos droits</span></div>
    </aside>
    <section className="workspace">
      <header><div><strong>{agents.find((a) => a.agentKey === active)?.label ?? "Mes agents"}</strong><small>Conversation privée via Fabric</small></div><span className="privacy">Connexion sécurisée</span></header>
      {agents.map((agent) => <AgentPanel key={agent.agentKey} agent={agent} active={agent.agentKey === active} />)}
      {active === null && <div className="waiting">Sélectionnez un agent pour commencer.</div>}
    </section>
  </main>;
}
