import { readFile as readPrivateFile } from "node:fs/promises";
import { resolve, sep } from "node:path";

const MAX_ASSET_BYTES = 4 * 1024 * 1024;
const ASSET_NAME = /^assets\/[A-Za-z0-9][A-Za-z0-9._-]{0,255}$/;
const SOURCE_MAP = /\.map$/i;
const TYPES = Object.freeze({
  ".css": "text/css; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".mjs": "text/javascript; charset=utf-8",
  ".svg": "image/svg+xml",
  ".woff": "font/woff",
  ".woff2": "font/woff2",
  ".png": "image/png",
  ".webp": "image/webp",
  ".ico": "image/x-icon",
});
const fail = () => { throw new Error("Declared browser artifacts unavailable"); };

function contentType(name) {
  if (name === "index.html") return "text/html; charset=utf-8";
  const extension = name.slice(name.lastIndexOf(".")).toLowerCase();
  return TYPES[extension] || null;
}

function declaredNames(manifest) {
  if (!manifest || typeof manifest !== "object" || Array.isArray(manifest)) fail();
  const names = new Set(["index.html"]);
  for (const entry of Object.values(manifest)) {
    if (!entry || typeof entry !== "object" || Array.isArray(entry)) fail();
    for (const name of [entry.file, ...(entry.css || []), ...(entry.assets || [])]) {
      if (typeof name !== "string" || !ASSET_NAME.test(name) || SOURCE_MAP.test(name) ||
          !contentType(name)) fail();
      names.add(name);
    }
  }
  return names;
}

/**
 * Immutable Vite build boundary. The manifest is an allow-list: source files,
 * arbitrary paths, manifest JSON and source maps cannot become HTTP content.
 */
export async function createDeclaredBrowserArtifacts({
  assetDir = "/app/client", manifestPath = assetDir + "/asset-manifest.json",
  readFile = readPrivateFile,
} = {}) {
  if (typeof assetDir !== "string" || typeof manifestPath !== "string" ||
      typeof readFile !== "function") fail();
  let manifest;
  try { manifest = JSON.parse(await readFile(manifestPath, "utf8")); }
  catch { fail(); }
  const names = declaredNames(manifest);
  const root = resolve(assetDir) + sep;
  return Object.freeze({
    async get(pathname) {
      const name = pathname === "/" ? "index.html" : pathname.slice(1);
      if (typeof pathname !== "string" || !pathname.startsWith("/") ||
          pathname.includes("\\") || pathname.includes("%") || SOURCE_MAP.test(name) ||
          !names.has(name)) return null;
      const target = resolve(assetDir, name);
      if (!target.startsWith(root) || !contentType(name)) return null;
      try {
        const body = await readFile(target);
        if (!Buffer.isBuffer(body) || body.length > MAX_ASSET_BYTES) fail();
        return Object.freeze({ body, contentType: contentType(name) });
      } catch { return null; }
    },
  });
}
