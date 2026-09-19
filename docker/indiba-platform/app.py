import asyncio
import json
import os
from contextlib import asynccontextmanager
from typing import Any

import asyncpg
import httpx
import uvicorn
from fastapi import FastAPI, Header, HTTPException, Request, Response
from fastapi.responses import StreamingResponse
from kubernetes import client, config, watch

ROLE = os.getenv("INDIBA_ROLE", "control-plane")
DATABASE_URL = os.getenv("DATABASE_URL", "")
OPENFGA_URL = os.getenv("OPENFGA_URL", "http://openfga.indiba-system.svc.cluster.local:8080")
HINDSIGHT_URL = os.getenv("HINDSIGHT_URL", "http://hindsight-indiba-api.tenant-indiba.svc.cluster.local:8888")
CONTROL_PLANE_URL = os.getenv("CONTROL_PLANE_URL", "http://control-plane.indiba-system.svc.cluster.local:8081")
OPENROUTER_BASE_URL = os.getenv("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1")
OPENROUTER_API_KEY = os.getenv("OPENROUTER_API_KEY", "")
GATEWAY_TOKEN = os.getenv("GATEWAY_TOKEN", "")
TENANT_REF = os.getenv("TENANT_REF", "TEN00001")
TENANT_ID = os.getenv("TENANT_ID", "00000000-0000-7000-8000-000000000001")
HERMES_IMAGE = os.getenv("HERMES_IMAGE", "nousresearch/hermes-agent:v2026.9.7")
POC_ALLOW_ANY_AUTHENTICATED = os.getenv("POC_ALLOW_ANY_AUTHENTICATED", "true").lower() == "true"

pool: asyncpg.Pool | None = None
openfga_store_id: str | None = None
openfga_model_id: str | None = None


async def db() -> asyncpg.Pool:
    global pool
    if pool is None:
        if not DATABASE_URL:
            raise RuntimeError("DATABASE_URL is required")
        pool = await asyncpg.create_pool(DATABASE_URL, min_size=1, max_size=5)
    return pool


async def init_schema() -> None:
    p = await db()
    async with p.acquire() as conn:
        await conn.execute(
            """
            create table if not exists platform_setting (
              key text primary key,
              value text not null
            );
            create table if not exists agent_identity (
              agent_ref text primary key,
              tenant_ref text not null,
              principal_sub text,
              alias text not null,
              runtime_namespace text not null default 'tenant-indiba',
              runtime_name text not null,
              memory_bank_ref text not null,
              memory_bank_id text not null,
              enabled boolean not null default true,
              created_at timestamptz not null default now()
            );
            create table if not exists usage_event (
              id bigserial primary key,
              occurred_at timestamptz not null default now(),
              tenant_ref text not null,
              principal_sub text,
              agent_ref text,
              service text not null,
              operation text not null,
              provider text,
              model text,
              prompt_tokens bigint,
              completion_tokens bigint,
              estimated_cost_usd numeric(12,6),
              request_id text
            );
            """
        )
        await conn.execute(
            """
            insert into agent_identity (
              agent_ref, tenant_ref, principal_sub, alias, runtime_name,
              memory_bank_ref, memory_bank_id
            ) values ($1,$2,$3,$4,$5,$6,$7)
            on conflict (agent_ref) do nothing
            """,
            os.getenv("POC_AGENT_REF", "TEN00001-USR00001-TPL00001-INS001"),
            TENANT_REF,
            os.getenv("POC_PRINCIPAL_SUB") or None,
            os.getenv("POC_AGENT_ALIAS", "Nadia Sud-Ouest"),
            os.getenv("POC_RUNTIME_NAME", "nadia-bertrand"),
            os.getenv("POC_BANK_REF", "MB-TEN00001-USR00001-TPL00001-INS001"),
            os.getenv("POC_BANK_ID", "nadia-bertrand"),
        )


async def ensure_openfga() -> None:
    global openfga_store_id, openfga_model_id
    p = await db()
    async with p.acquire() as conn:
        row = await conn.fetchrow("select value from platform_setting where key='openfga_store_id'")
        if row:
            openfga_store_id = row["value"]
        row = await conn.fetchrow("select value from platform_setting where key='openfga_model_id'")
        if row:
            openfga_model_id = row["value"]
    async with httpx.AsyncClient(timeout=10) as h:
        if not openfga_store_id:
            r = await h.post(f"{OPENFGA_URL}/stores", json={"name": "indiba-phase0"})
            r.raise_for_status()
            openfga_store_id = r.json()["id"]
            async with p.acquire() as conn:
                await conn.execute(
                    "insert into platform_setting(key,value) values('openfga_store_id',$1) on conflict(key) do update set value=excluded.value",
                    openfga_store_id,
                )
        if not openfga_model_id:
            model = {
                "schema_version": "1.1",
                "type_definitions": [
                    {"type": "user"},
                    {
                        "type": "agent",
                        "relations": {
                            "owner": {"this": {}},
                            "viewer": {"this": {}},
                            "can_use": {"union": {"child": [{"computedUserset": {"relation": "owner"}}, {"computedUserset": {"relation": "viewer"}}]}},
                        },
                        "metadata": {
                            "relations": {
                                "owner": {"directly_related_user_types": [{"type": "user"}]},
                                "viewer": {"directly_related_user_types": [{"type": "user"}]},
                            }
                        },
                    },
                ],
            }
            r = await h.post(f"{OPENFGA_URL}/stores/{openfga_store_id}/authorization-models", json=model)
            r.raise_for_status()
            openfga_model_id = r.json()["authorization_model_id"]
            async with p.acquire() as conn:
                await conn.execute(
                    "insert into platform_setting(key,value) values('openfga_model_id',$1) on conflict(key) do update set value=excluded.value",
                    openfga_model_id,
                )
        async with p.acquire() as conn:
            rows = await conn.fetch("select agent_ref, principal_sub from agent_identity where principal_sub is not null")
        if rows:
            tuples = [
                {"user": f"user:{r['principal_sub']}", "relation": "owner", "object": f"agent:{r['agent_ref']}"}
                for r in rows
            ]
            await h.post(
                f"{OPENFGA_URL}/stores/{openfga_store_id}/write",
                json={"writes": {"tuple_keys": tuples}, "authorization_model_id": openfga_model_id},
            )


async def can_use_agent(principal_sub: str, agent_ref: str) -> bool:
    if POC_ALLOW_ANY_AUTHENTICATED and principal_sub:
        return True
    if not principal_sub or not openfga_store_id or not openfga_model_id:
        return False
    async with httpx.AsyncClient(timeout=5) as h:
        r = await h.post(
            f"{OPENFGA_URL}/stores/{openfga_store_id}/check",
            json={
                "tuple_key": {"user": f"user:{principal_sub}", "relation": "can_use", "object": f"agent:{agent_ref}"},
                "authorization_model_id": openfga_model_id,
            },
        )
        return r.is_success and bool(r.json().get("allowed"))


async def operator_loop() -> None:
    config.load_incluster_config()
    api = client.CustomObjectsApi()
    apps = client.AppsV1Api()
    core = client.CoreV1Api()
    w = watch.Watch()
    while True:
        try:
            for event in w.stream(
                api.list_cluster_custom_object,
                group="agents.indiba.io",
                version="v1alpha1",
                plural="hermesruntimes",
                timeout_seconds=60,
            ):
                obj = event["object"]
                ns = obj["metadata"]["namespace"]
                spec = obj.get("spec", {})
                if spec.get("tenantRef") != TENANT_REF:
                    continue
                name = obj["metadata"]["name"]
                labels = {
                    "app.kubernetes.io/name": name,
                    "app.kubernetes.io/part-of": "indiba-platform",
                    "indiba.io/component": "hermes-runtime",
                    "indiba.io/tenant-ref": spec.get("tenantRef", ""),
                    "indiba.io/agent-ref": spec.get("agentRef", "")[:63],
                    "indiba.io/llm-client": "true",
                }
                pvc_name = f"{name}-home"
                try:
                    core.read_namespaced_persistent_volume_claim(pvc_name, ns)
                except client.exceptions.ApiException as exc:
                    if exc.status == 404:
                        core.create_namespaced_persistent_volume_claim(
                            ns,
                            client.V1PersistentVolumeClaim(
                                metadata=client.V1ObjectMeta(name=pvc_name, labels=labels),
                                spec=client.V1PersistentVolumeClaimSpec(
                                    access_modes=["ReadWriteOnce"],
                                    resources=client.V1VolumeResourceRequirements(requests={"storage": "5Gi"}),
                                ),
                            ),
                        )
                dep = client.V1Deployment(
                    metadata=client.V1ObjectMeta(name=name, namespace=ns, labels=labels),
                    spec=client.V1DeploymentSpec(
                        replicas=1,
                        selector=client.V1LabelSelector(match_labels={"app.kubernetes.io/name": name}),
                        template=client.V1PodTemplateSpec(
                            metadata=client.V1ObjectMeta(labels=labels),
                            spec=client.V1PodSpec(
                                containers=[
                                    client.V1Container(
                                        name="hermes",
                                        image=HERMES_IMAGE,
                                        args=["gateway", "run"],
                                        ports=[client.V1ContainerPort(name="gateway", container_port=8642)],
                                        env=[
                                            client.V1EnvVar(name="HERMES_HOME", value="/opt/data"),
                                            client.V1EnvVar(name="OPENAI_BASE_URL", value="http://llm-gateway.indiba-system.svc.cluster.local:8080/v1"),
                                            client.V1EnvVar(name="OPENAI_API_KEY", value_from=client.V1EnvVarSource(secret_key_ref=client.V1SecretKeySelector(name="indiba-runtime-capability", key="gateway-token"))),
                                            client.V1EnvVar(name="HINDSIGHT_API_URL", value="http://memory-gateway.tenant-indiba.svc.cluster.local:8877"),
                                            client.V1EnvVar(name="HINDSIGHT_BANK_ID", value=spec.get("memory", {}).get("bankRef", name)),
                                        ],
                                        volume_mounts=[client.V1VolumeMount(name="home", mount_path="/opt/data")],
                                        resources=client.V1ResourceRequirements(
                                            requests={"cpu": "100m", "memory": "512Mi"},
                                            limits={"cpu": "2", "memory": "4Gi"},
                                        ),
                                    )
                                ],
                                volumes=[client.V1Volume(name="home", persistent_volume_claim=client.V1PersistentVolumeClaimVolumeSource(claim_name=pvc_name))],
                            ),
                        ),
                    ),
                )
                try:
                    apps.read_namespaced_deployment(name, ns)
                    apps.patch_namespaced_deployment(name, ns, dep)
                except client.exceptions.ApiException as exc:
                    if exc.status == 404:
                        apps.create_namespaced_deployment(ns, dep)
                service = client.V1Service(
                    metadata=client.V1ObjectMeta(name=name, namespace=ns, labels=labels),
                    spec=client.V1ServiceSpec(
                        selector={"app.kubernetes.io/name": name},
                        ports=[client.V1ServicePort(name="gateway", port=8642, target_port="gateway")],
                    ),
                )
                try:
                    core.read_namespaced_service(name, ns)
                    core.patch_namespaced_service(name, ns, service)
                except client.exceptions.ApiException as exc:
                    if exc.status == 404:
                        core.create_namespaced_service(ns, service)
                body = {
                    "status": {
                        "observedGeneration": obj["metadata"].get("generation", 1),
                        "phase": "Ready",
                        "runtimeId": name,
                        "endpoint": f"http://{name}.{ns}.svc.cluster.local:8642",
                        "configurationRevision": spec.get("configurationRevision"),
                    }
                }
                api.patch_namespaced_custom_object_status(
                    group="agents.indiba.io",
                    version="v1alpha1",
                    namespace=ns,
                    plural="hermesruntimes",
                    name=name,
                    body=body,
                )
        except Exception as exc:
            print(f"operator loop error: {exc}", flush=True)
            await asyncio.sleep(3)


@asynccontextmanager
async def lifespan(app: FastAPI):
    if ROLE == "control-plane":
        await init_schema()
        for _ in range(30):
            try:
                await ensure_openfga()
                break
            except Exception as exc:
                print(f"waiting for OpenFGA: {exc}", flush=True)
                await asyncio.sleep(2)
    if ROLE == "operator":
        task = asyncio.create_task(operator_loop())
    else:
        task = None
    yield
    if task:
        task.cancel()
    global pool
    if pool:
        await pool.close()
        pool = None


app = FastAPI(title=f"INDIBA {ROLE}", version="0.1.0", lifespan=lifespan)


@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok", "role": ROLE}


@app.get("/v1/agents")
async def list_agents(x_indiba_principal_sub: str = Header(default="")) -> dict[str, Any]:
    if ROLE != "control-plane":
        raise HTTPException(404)
    p = await db()
    async with p.acquire() as conn:
        rows = await conn.fetch("select * from agent_identity where tenant_ref=$1 and enabled=true order by alias", TENANT_REF)
    items = []
    for row in rows:
        if await can_use_agent(x_indiba_principal_sub, row["agent_ref"]):
            items.append(
                {
                    "agentRef": row["agent_ref"],
                    "name": row["alias"],
                    "status": "alwaysOn",
                    "memoryBankRef": row["memory_bank_ref"],
                    "memoryBankId": row["memory_bank_id"],
                }
            )
    return {"items": items}


@app.post("/v1/agents/{agent_ref}/chat")
async def chat(agent_ref: str, request: Request, x_indiba_principal_sub: str = Header(default="")) -> Response:
    if ROLE != "control-plane":
        raise HTTPException(404)
    if not await can_use_agent(x_indiba_principal_sub, agent_ref):
        raise HTTPException(403, "agent access denied")
    p = await db()
    async with p.acquire() as conn:
        row = await conn.fetchrow("select runtime_namespace,runtime_name from agent_identity where agent_ref=$1", agent_ref)
    if not row:
        raise HTTPException(404, "agent not found")
    payload = await request.json()
    url = f"http://{row['runtime_name']}.{row['runtime_namespace']}.svc.cluster.local:8642/v1/chat/completions"
    h = httpx.AsyncClient(timeout=None)
    upstream = await h.send(h.build_request("POST", url, json=payload), stream=True)
    if upstream.status_code >= 400:
        body = await upstream.aread()
        await h.aclose()
        return Response(body, status_code=upstream.status_code, media_type=upstream.headers.get("content-type"))

    async def stream():
        try:
            async for chunk in upstream.aiter_raw():
                yield chunk
        finally:
            await upstream.aclose()
            await h.aclose()

    return StreamingResponse(stream(), media_type=upstream.headers.get("content-type", "application/json"))


@app.post("/internal/memory/authorize")
async def authorize_memory(request: Request) -> dict[str, bool]:
    if ROLE != "control-plane":
        raise HTTPException(404)
    data = await request.json()
    agent_ref = str(data.get("agentRef", ""))
    bank_id = str(data.get("bankId", ""))
    operation = str(data.get("operation", ""))
    if operation not in {"retain", "recall", "reflect", "mcp", "read", "write"}:
        return {"allowed": False}
    p = await db()
    async with p.acquire() as conn:
        allowed = await conn.fetchval(
            "select exists(select 1 from agent_identity where agent_ref=$1 and memory_bank_id=$2 and enabled=true)",
            agent_ref,
            bank_id,
        )
    return {"allowed": bool(allowed)}


def require_gateway_token(auth: str | None) -> None:
    if not GATEWAY_TOKEN:
        raise HTTPException(503, "gateway token is not configured")
    if auth != f"Bearer {GATEWAY_TOKEN}":
        raise HTTPException(401, "invalid workload capability")


@app.api_route("/v1/{path:path}", methods=["GET", "POST", "PUT", "PATCH", "DELETE"])
async def llm_proxy(path: str, request: Request, authorization: str | None = Header(default=None)) -> Response:
    if ROLE != "llm-gateway":
        raise HTTPException(404)
    require_gateway_token(authorization)
    if not OPENROUTER_API_KEY:
        raise HTTPException(503, "OpenRouter credential unavailable")
    body = await request.body()
    headers = {
        "authorization": f"Bearer {OPENROUTER_API_KEY}",
        "content-type": request.headers.get("content-type", "application/json"),
        "x-title": "INDIBA Phase0",
    }
    h = httpx.AsyncClient(timeout=None)
    upstream = await h.send(h.build_request(request.method, f"{OPENROUTER_BASE_URL}/{path}", content=body, headers=headers), stream=True)

    async def stream():
        try:
            async for chunk in upstream.aiter_raw():
                yield chunk
        finally:
            await upstream.aclose()
            await h.aclose()

    return StreamingResponse(stream(), status_code=upstream.status_code, media_type=upstream.headers.get("content-type"))


def extract_bank(path: str) -> tuple[str, str]:
    parts = [p for p in path.split("/") if p]
    if "banks" in parts:
        i = parts.index("banks")
        if len(parts) > i + 1:
            op = parts[i + 2] if len(parts) > i + 2 else "read"
            return parts[i + 1], op
    if parts and parts[0] == "mcp" and len(parts) > 1:
        return parts[1], "mcp"
    return "", ""


@app.api_route("/{path:path}", methods=["GET", "POST", "PUT", "PATCH", "DELETE"])
async def memory_proxy(path: str, request: Request, x_indiba_agent_ref: str = Header(default="")) -> Response:
    if ROLE != "memory-gateway":
        raise HTTPException(404)
    bank_id, op = extract_bank(path)
    if not x_indiba_agent_ref or not bank_id:
        raise HTTPException(400, "agent and bank are required")
    async with httpx.AsyncClient(timeout=15) as h:
        authz = await h.post(
            f"{CONTROL_PLANE_URL}/internal/memory/authorize",
            json={"agentRef": x_indiba_agent_ref, "bankId": bank_id, "operation": op},
        )
        if not authz.is_success or not authz.json().get("allowed"):
            raise HTTPException(403, "memory bank access denied")
    body = await request.body()
    headers = {k: v for k, v in request.headers.items() if k.lower() not in {"host", "x-indiba-agent-ref", "content-length"}}
    async with httpx.AsyncClient(timeout=None) as h:
        upstream = await h.request(request.method, f"{HINDSIGHT_URL}/{path}", content=body, headers=headers, params=request.query_params)
    return Response(upstream.content, status_code=upstream.status_code, media_type=upstream.headers.get("content-type"))


if __name__ == "__main__":
    ports = {"control-plane": 8081, "operator": 8082, "llm-gateway": 8080, "memory-gateway": 8877}
    uvicorn.run(app, host="0.0.0.0", port=int(os.getenv("PORT", ports.get(ROLE, 8080))))
