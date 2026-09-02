import fs from 'node:fs/promises';
import path from 'node:path';
import matter from 'gray-matter';

export type WikiItem = {
  title: string;
  path: string;
  status?: string;
  owner?: string;
  project?: string;
  context?: string;
  priority?: string;
  updated?: string;
  excerpt: string;
};

async function exists(p: string) {
  try { await fs.access(p); return true; } catch { return false; }
}

async function walk(dir: string, root = dir): Promise<string[]> {
  if (!(await exists(dir))) return [];
  const entries = await fs.readdir(dir, { withFileTypes: true });
  const out: string[] = [];
  for (const entry of entries) {
    if (entry.name === '.git') continue;
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) out.push(...await walk(full, root));
    if (entry.isFile() && entry.name.endsWith('.md')) out.push(path.relative(root, full));
  }
  return out.sort();
}

function titleFromMarkdown(body: string, fallback: string) {
  const h1 = body.match(/^#\s+(.+)$/m)?.[1]?.trim();
  return h1 || fallback.replace(/\.md$/, '').replaceAll('-', ' ');
}

function excerpt(body: string) {
  return body
    .replace(/^---[\s\S]*?---\s*/m, '')
    .replace(/^#\s+.+$/m, '')
    .split('\n')
    .map((x) => x.trim())
    .filter(Boolean)
    .slice(0, 3)
    .join(' · ')
    .slice(0, 220);
}

export async function readWiki(root: string, limit = 80): Promise<WikiItem[]> {
  const files = (await walk(root)).slice(0, limit);
  const items: WikiItem[] = [];
  for (const rel of files) {
    const full = path.join(root, rel);
    const raw = await fs.readFile(full, 'utf8');
    const parsed = matter(raw);
    const data = parsed.data as Record<string, unknown>;
    items.push({
      title: String(data.title || titleFromMarkdown(parsed.content, path.basename(rel))),
      path: rel,
      status: data.status ? String(data.status) : undefined,
      owner: data.owner ? String(data.owner) : undefined,
      project: data.project ? String(data.project) : undefined,
      context: data.context ? String(data.context) : undefined,
      priority: data.priority ? String(data.priority) : undefined,
      updated: data.updated ? String(data.updated) : undefined,
      excerpt: excerpt(parsed.content)
    });
  }
  return items;
}

export function countBy(items: WikiItem[], key: keyof WikiItem) {
  return items.reduce<Record<string, number>>((acc, item) => {
    const value = item[key] ? String(item[key]) : 'none';
    acc[value] = (acc[value] || 0) + 1;
    return acc;
  }, {});
}
