import { createPrivateBFFHandler, createPrivateBFFServer, resolveRuntimeGate } from "./private-bff-server.mjs";

export async function startPrivateBFF(env = process.env) {
  // Strict feature gate checked BEFORE loading any database credential,
  // HTTP client or OIDC session adapter.
  const enabled = resolveRuntimeGate(env);
  const port = Number(env.PORT || "3000");
  if (!Number.isSafeInteger(port) || port < 1024 || port > 65535) {
    throw new Error("Invalid private BFF port");
  }
  const handler = createPrivateBFFHandler({ enabled });
  const server = createPrivateBFFServer({ handle: handler });
  await new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(port, "0.0.0.0", resolve);
  });
  const close = () => { server.close(); };
  process.once("SIGTERM", close);
  process.once("SIGINT", close);
  return server;
}

// Imported by Node's actual runtime entrypoint, not during unit tests.
if (process.argv[1] && import.meta.url === new URL("file://" + process.argv[1]).href) {
  startPrivateBFF().catch(() => {
    // Do not log environment, token, key, UUID, cookie or DB URL.
    process.stderr.write("Private Fabric BFF startup denied\n");
    process.exitCode = 1;
  });
}
