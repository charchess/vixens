import { json, error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import fs from 'node:fs/promises';
import path from 'node:path';
import { spawn } from 'node:child_process';
import matter from 'gray-matter';

const GTD = process.env.GTDWIKI_PATH || '/data/public/GTD';
const AUDIT_DIR = process.env.GTD_AUDIT_DIR || '/tmp/hairem-dashboard-audit';
const allowedRoots = ['actions', 'projects', 'references'];

type Body = {
  path?: string;
  kind?: 'action' | 'project' | 'reference' | 'inbox' | 'waiting';
  fields?: Record<string, unknown>;
  markdown?: string;
  create?: boolean;
};

function arr(v: unknown): string[] {
  if (Array.isArray(v)) return v.map(String).map((s) => s.trim()).filter(Boolean);
  if (v === null || v === undefined || v === '') return [];
  return String(v).split(/[,;]/).map((s) => s.trim()).filter(Boolean);
}
function cleanString(v: unknown): string | undefined {
  if (v === null || v === undefined) return undefined;
  const s = String(v).trim();
  return s ? s : undefined;
}
function relPath(input: string) {
  const normalized = input.replace(/\\/g, '/').replace(/^\/+/, '');
  if (normalized.includes('..')) throw error(400, 'invalid path');
  if (!normalized.endsWith('.md')) throw error(400, 'not a markdown file');
  const root = normalized.split('/')[0];
  if (!allowedRoots.includes(root)) throw error(400, 'root not allowed');
  return normalized;
}
function slug(s: string) {
  return s.toLowerCase().normalize('NFD').replace(/[\u0300-\u036f]/g, '').replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '').slice(0, 80) || 'capture';
}
async function runGit(args: string[], cwd: string) {
  return new Promise<{ code: number; out: string }>((resolve) => {
    const p = spawn('git', args, { cwd, stdio: ['ignore', 'pipe', 'pipe'] });
    let out = '';
    p.stdout.on('data', (d) => out += d);
    p.stderr.on('data', (d) => out += d);
    p.on('close', (code) => resolve({ code: code ?? 0, out }));
    p.on('error', (e) => resolve({ code: 1, out: String(e) }));
  });
}
async function audit(event: Record<string, unknown>) {
  await fs.mkdir(AUDIT_DIR, { recursive: true });
  const stamp = new Date().toISOString().replace(/[:.]/g, '').replace(/[^0-9TZ-]/g, '');
  await fs.writeFile(path.join(AUDIT_DIR, `${stamp}-gtd-edit.json`), JSON.stringify(event, null, 2) + '\n', { mode: 0o660 });
}
function applyFields(data: Record<string, unknown>, fields: Record<string, unknown>) {
  const map: Record<string, string> = {
    title: 'title', status: 'status', priority: 'priority', projectId: 'project_id', project: 'project',
    dueAt: 'due_at', scheduledAt: 'scheduled_at', followupAt: 'followup_at', postponedUntil: 'postponed_until',
    finish: 'finish', altitude: 'altitude', importance: 'importance', result: 'result'
  };
  for (const [from, to] of Object.entries(map)) {
    if (from in fields) {
      const v = cleanString(fields[from]);
      if (v === undefined) delete data[to]; else data[to] = v;
    }
  }
  if ('contexts' in fields) data.contexts = arr(fields.contexts);
  if ('tags' in fields) data.tags = arr(fields.tags);
  if ('people' in fields) data.people = arr(fields.people);
  data.updated_at = new Date().toISOString();
}

export const GET: RequestHandler = async ({ url }) => {
  const rel = relPath(url.searchParams.get('path') || '');
  const abs = path.join(GTD, rel);
  const raw = await fs.readFile(abs, 'utf8');
  const parsed = matter(raw);
  return json({ ok: true, path: rel, frontmatter: parsed.data, markdown: parsed.content });
};

export const POST: RequestHandler = async ({ request }) => {
  const input = await request.json().catch(() => ({})) as Body;
  const fields = input.fields || {};
  let rel: string;
  if (input.create) {
    const title = cleanString(fields.title) || cleanString(input.markdown) || 'Capture rapide';
    rel = `actions/${new Date().toISOString().replace(/[:.]/g, '').slice(0, 15)}-${slug(title)}.md`;
    const abs = path.join(GTD, rel);
    await fs.mkdir(path.dirname(abs), { recursive: true });
    const data: Record<string, unknown> = { id: slug(title), title, status: cleanString(fields.status) || 'inbox', contexts: arr(fields.contexts), tags: arr(fields.tags), updated_at: new Date().toISOString() };
    const content = cleanString(input.markdown) || '';
    await fs.writeFile(abs, matter.stringify(`# ${title}\n\n${content}\n`, data), { mode: 0o660 });
  } else {
    if (!input.path) throw error(400, 'missing path');
    rel = relPath(input.path);
    const abs = path.join(GTD, rel);
    const raw = await fs.readFile(abs, 'utf8');
    const parsed = matter(raw);
    const data = { ...(parsed.data as Record<string, unknown>) };
    applyFields(data, fields);
    const nextBody = typeof input.markdown === 'string' ? input.markdown : parsed.content;
    const tmp = `${abs}.tmp-${process.pid}-${Date.now()}`;
    await fs.writeFile(tmp, matter.stringify(nextBody, data), { mode: 0o660 });
    await fs.rename(tmp, abs);
  }

  const absRoot = GTD;
  const absFile = path.join(GTD, rel);
  await audit({ event: input.create ? 'gtd.item.created' : 'gtd.item.updated', path: rel, kind: input.kind || 'item', fields, timestamp: new Date().toISOString() });
  const gitCheck = await runGit(['rev-parse', '--is-inside-work-tree'], absRoot);
  let git = { available: gitCheck.code === 0, committed: false, output: gitCheck.out.slice(0, 500) };
  if (git.available) {
    await runGit(['add', absFile], absRoot);
    const commit = await runGit(['commit', '-m', input.create ? `GTD capture: ${rel}` : `GTD edit: ${rel}`], absRoot);
    git = { available: true, committed: commit.code === 0, output: commit.out.slice(0, 1000) };
  }
  return json({ ok: true, path: rel, git });
};
