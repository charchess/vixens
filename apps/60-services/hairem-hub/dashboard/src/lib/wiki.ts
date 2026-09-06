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

export type GtdTask = {
  title: string;
  path: string;
  section: string;
  checked: boolean;
  priority?: string;
  project?: string;
  context?: string;
  criterion?: string;
  guardrail?: string;
  statusText?: string;
  excerpt: string;
};

export type GtdProject = {
  title: string;
  path: string;
  result?: string;
  statusText?: string;
  nextAction?: string;
  criterion?: string;
  guardrail?: string;
  personas?: string;
  active: boolean;
  noNextAction: boolean;
};

export type GtdWaiting = {
  title: string;
  path: string;
  expected?: string;
  project?: string;
  state?: string;
  actionAfter?: string;
  excerpt: string;
};

export type GtdCockpit = {
  source: string;
  updated: Record<string, string>;
  inbox: GtdTask[];
  next: GtdTask[];
  doneRecent: GtdTask[];
  projects: GtdProject[];
  waiting: GtdWaiting[];
  someday: GtdTask[];
  review: string[];
  now: GtdTask[];
  trust: {
    inboxOpen: number;
    nextOpen: number;
    p1Open: number;
    projectsActive: number;
    projectsNoNext: number;
    waiting: number;
    someday: number;
    reviewUpdated?: string;
    label: string;
    led: 'ok' | 'warn' | 'down';
  };
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

function cleanValue(value = '') {
  return value.replace(/`/g, '').replace(/\s+/g, ' ').trim();
}

function updatedFrom(raw: string) {
  return raw.match(/^>\s*Dernière mise à jour\s*:\s*(.+)$/m)?.[1]?.trim();
}

function stripPriority(title: string) {
  const m = title.match(/^(P\d+)\s*[—-]\s*(.+)$/);
  return { priority: m?.[1], title: (m?.[2] || title).trim() };
}

function detailsToTask(titleRaw: string, checked: boolean, details: string[], rel: string, section: string): GtdTask {
  const { priority, title } = stripPriority(cleanValue(titleRaw));
  const get = (...labels: string[]) => {
    for (const line of details) {
      const normalized = line.replace(/^\s*-\s*/, '').trim();
      for (const label of labels) {
        const rx = new RegExp(`^${label}\\s*:\\s*(.+)$`, 'i');
        const m = normalized.match(rx);
        if (m) return cleanValue(m[1]);
      }
    }
    return undefined;
  };
  return {
    title,
    path: rel,
    section,
    checked,
    priority,
    project: get('Projet', 'Projet/contexte'),
    context: get('Contexte'),
    criterion: get('Critère fini', 'Critère de reprise'),
    guardrail: get('Garde-fou', 'Contrainte'),
    statusText: get('Statut', 'Résultat'),
    excerpt: details.map((d) => cleanValue(d.replace(/^\s*-\s*/, ''))).filter(Boolean).slice(0, 2).join(' · ').slice(0, 240)
  };
}

function parseChecklist(raw: string, rel: string) {
  const lines = raw.split('\n');
  const tasks: GtdTask[] = [];
  let section = 'GTD';
  let current: { checked: boolean; title: string; details: string[]; section: string } | null = null;
  const flush = () => {
    if (!current) return;
    tasks.push(detailsToTask(current.title, current.checked, current.details, rel, current.section));
    current = null;
  };
  for (const line of lines) {
    const h = line.match(/^##\s+(.+)$/);
    if (h) { flush(); section = cleanValue(h[1]); continue; }
    const item = line.match(/^\s*-\s*\[([ xX])\]\s+(.+)$/);
    if (item) {
      flush();
      current = { checked: item[1].toLowerCase() === 'x', title: item[2], details: [], section };
    } else if (current && /^\s{2,}-\s+/.test(line)) {
      current.details.push(line);
    }
  }
  flush();
  return tasks;
}

function sectionBlocks(raw: string, rel: string) {
  const blocks: Array<{ title: string; lines: string[]; path: string }> = [];
  const lines = raw.split('\n');
  let current: { title: string; lines: string[]; path: string } | null = null;
  for (const line of lines) {
    const h3 = line.match(/^###\s+(.+)$/);
    if (h3) {
      if (current) blocks.push(current);
      current = { title: cleanValue(h3[1]), lines: [], path: rel };
    } else if (current) {
      current.lines.push(line);
    }
  }
  if (current) blocks.push(current);
  return blocks;
}

function field(lines: string[], ...labels: string[]) {
  for (const line of lines) {
    const normalized = line.replace(/^\s*-\s*/, '').trim();
    for (const label of labels) {
      const rx = new RegExp(`^${label}\\s*:\\s*(.+)$`, 'i');
      const m = normalized.match(rx);
      if (m) return cleanValue(m[1]);
    }
  }
  return undefined;
}

function parseProjects(raw: string, rel: string): GtdProject[] {
  return sectionBlocks(raw, rel).map((b) => {
    const statusText = field(b.lines, 'Statut', 'État');
    const nextAction = field(b.lines, 'Prochaine action', 'Prochaine action du projet', 'Prochaine action disponible');
    const status = (statusText || '').toLowerCase();
    const next = (nextAction || '').toLowerCase();
    const active = !/(parking|terminé|termine|aucune|attente externe|en attente)/.test(status) && !/(aucune|si réactivé|si reactive)/.test(next);
    const noNextAction = active && (!nextAction || /aucune|flou|à clarifier/.test(next));
    return {
      title: b.title,
      path: rel,
      result: field(b.lines, 'Résultat voulu'),
      statusText,
      nextAction,
      criterion: field(b.lines, 'Critère fini', 'Critère fini de la prochaine action', 'Critère fini de la prochaine action après déclencheur'),
      guardrail: field(b.lines, 'Garde-fou', 'Déclencheur/garde-fou'),
      personas: field(b.lines, 'Personas utiles', 'Persona utile'),
      active,
      noNextAction
    };
  });
}

function parseWaiting(raw: string, rel: string): GtdWaiting[] {
  return sectionBlocks(raw, rel).map((b) => ({
    title: b.title,
    path: rel,
    expected: field(b.lines, 'Attendu'),
    project: field(b.lines, 'Projet', 'Projet/contexte', 'Bloque'),
    state: field(b.lines, 'État', 'Raison'),
    actionAfter: field(b.lines, 'Action après déclencheur', 'Prochain jalon'),
    excerpt: b.lines.map((l) => cleanValue(l.replace(/^\s*-\s*/, ''))).filter(Boolean).slice(0, 2).join(' · ').slice(0, 240)
  }));
}

function parseReview(raw: string) {
  return parseChecklist(raw, 'review.md').filter((t) => !t.checked).map((t) => t.title);
}

function asString(value: unknown): string | undefined {
  if (value === null || value === undefined) return undefined;
  if (Array.isArray(value)) return value.map((v) => String(v)).join(', ');
  return String(value);
}

function normalizedStatus(value: unknown) {
  return String(value || '').trim().toLowerCase().replace(/[ -]/g, '_');
}

function isDoneStatus(status: string) {
  return ['done', 'closed', 'cancelled', 'canceled', 'trash', 'deleted'].includes(status);
}

function isWaitingStatus(status: string) {
  return ['waiting', 'waiting_for', 'blocked', 'external_waiting', 'en_attente'].includes(status);
}

function isInboxStatus(status: string) {
  return ['inbox', 'capture', 'captured', 'unclarified', 'clarify'].includes(status);
}

function isSomedayStatus(status: string) {
  return ['someday', 'maybe', 'parking', 'parked', 'later', 'backlog'].includes(status);
}

function isActiveProjectStatus(status: string) {
  return ['active', 'current', 'runway', 'next'].includes(status) || !status;
}

function frontmatterArray(value: unknown): string[] {
  if (Array.isArray(value)) return value.map((v) => String(v));
  if (value === null || value === undefined || value === '') return [];
  return [String(value)];
}

function bodyField(body: string, ...labels: string[]) {
  return field(body.split('\n'), ...labels);
}

function detailsExcerptFromBody(body: string) {
  const details = body.split('\n').filter((line) => /^\s*-\s+/.test(line)).map((line) => cleanValue(line.replace(/^\s*-\s*/, '')));
  return details.slice(0, 3).join(' · ').slice(0, 260) || excerpt(body);
}

async function readMarkdown(root: string, rel: string) {
  const raw = await fs.readFile(path.join(root, rel), 'utf8');
  const parsed = matter(raw);
  return { raw, content: parsed.content, data: parsed.data as Record<string, unknown> };
}

function actionFromMarkdown(root: string, rel: string, raw: string, body: string, data: Record<string, unknown>): GtdTask {
  const title = asString(data.title) || titleFromMarkdown(body, path.basename(rel));
  const contexts = frontmatterArray(data.contexts ?? data.context);
  const tags = frontmatterArray(data.tags);
  const status = normalizedStatus(data.status);
  const priority = asString(data.priority) || tags.find((t) => /^p\d$/i.test(t))?.toUpperCase();
  const section = asString(data.source_section) || contexts.join(', ') || status || 'GTD';
  const project = asString(data.project_id ?? data.project);
  const criterion = asString(data.finish) || bodyField(body, 'Critère fini', 'Critère de reprise', 'Fini quand');
  const guardrail = bodyField(body, 'Garde-fou', 'Contrainte', 'GTD');
  const statusText = asString(data.status) || bodyField(body, 'Statut', 'Résultat', 'État');
  return {
    title,
    path: rel,
    section,
    checked: isDoneStatus(status),
    priority,
    project,
    context: contexts.join(', ') || undefined,
    criterion,
    guardrail,
    statusText,
    excerpt: detailsExcerptFromBody(body)
  };
}

function waitingFromAction(task: GtdTask, raw: string, body: string, data: Record<string, unknown>): GtdWaiting {
  return {
    title: task.title,
    path: task.path,
    expected: bodyField(body, 'Attendu') || asString(data.waiting_on_ref_id),
    project: task.project || bodyField(body, 'Projet', 'Projet/contexte', 'Bloque'),
    state: bodyField(body, 'État', 'Raison') || task.statusText,
    actionAfter: bodyField(body, 'Action après déclencheur', 'Prochain jalon'),
    excerpt: task.excerpt
  };
}

function projectFromMarkdown(rel: string, body: string, data: Record<string, unknown>, nextByProject: Map<string, GtdTask[]>): GtdProject {
  const id = asString(data.id);
  const title = asString(data.title) || titleFromMarkdown(body, path.basename(rel));
  const status = normalizedStatus(data.status);
  const statusText = asString(data.status) || bodyField(body, 'Statut', 'État');
  const linkedNext = id ? nextByProject.get(id) || [] : [];
  const extractedNext = bodyField(body, 'Prochaine action', 'Next action du projet', 'Prochaine action du projet', 'Prochaine action disponible');
  const nextAction = linkedNext[0]?.title || extractedNext;
  const active = isActiveProjectStatus(status);
  const nextLower = (nextAction || '').toLowerCase();
  const noNextAction = active && (!nextAction || /aucune|flou|à clarifier|a clarifier|si réactivé|si reactive/.test(nextLower));
  return {
    title,
    path: rel,
    result: bodyField(body, 'Résultat voulu'),
    statusText,
    nextAction,
    criterion: bodyField(body, 'Critère fini', 'Critère fini de la prochaine action', 'Critère fini de la prochaine action après déclencheur'),
    guardrail: bodyField(body, 'Garde-fou', 'Déclencheur/garde-fou'),
    personas: bodyField(body, 'Personas utiles', 'Persona utile'),
    active,
    noNextAction
  };
}

async function readFolderGtdCockpit(root: string): Promise<GtdCockpit | null> {
  const actionFiles = (await walk(path.join(root, 'actions'), root)).filter((rel) => rel.startsWith('actions/') && rel.endsWith('.md'));
  const projectFiles = (await walk(path.join(root, 'projects'), root)).filter((rel) => rel.startsWith('projects/') && rel.endsWith('.md'));
  if (!actionFiles.length && !projectFiles.length) return null;

  const actionReads = await Promise.all(actionFiles.map(async (rel) => ({ rel, ...(await readMarkdown(root, rel)) })));
  const projectReads = await Promise.all(projectFiles.map(async (rel) => ({ rel, ...(await readMarkdown(root, rel)) })));
  const actions = actionReads.map(({ rel, raw, content, data }) => ({ task: actionFromMarkdown(root, rel, raw, content, data), raw, body: content, data }));

  const inbox = actions.filter(({ data }) => isInboxStatus(normalizedStatus(data.status))).map(({ task }) => task);
  const waitingRich = actions.filter(({ data }) => isWaitingStatus(normalizedStatus(data.status))).map(({ task, raw, body, data }) => waitingFromAction(task, raw, body, data));
  const someday = actions.filter(({ data }) => isSomedayStatus(normalizedStatus(data.status))).map(({ task }) => task);
  const doneRecent = actions.filter(({ data }) => isDoneStatus(normalizedStatus(data.status))).map(({ task }) => task).slice(-8).reverse();
  const next = actions
    .filter(({ data }) => {
      const status = normalizedStatus(data.status);
      const isNext = data.is_next === true || String(data.is_next).toLowerCase() === 'true';
      return !isDoneStatus(status) && !isWaitingStatus(status) && !isInboxStatus(status) && !isSomedayStatus(status) && (isNext || ['next', 'active', 'next_action'].includes(status));
    })
    .map(({ task }) => task);

  const nextByProject = new Map<string, GtdTask[]>();
  for (const task of next) {
    if (!task.project) continue;
    const list = nextByProject.get(task.project) || [];
    list.push(task);
    nextByProject.set(task.project, list);
  }
  const projects = projectReads.map(({ rel, content, data }) => projectFromMarkdown(rel, content, data, nextByProject));
  const p1Open = next.filter((t) => t.priority === 'P1').length;
  const projectsActive = projects.filter((p) => p.active).length;
  const projectsNoNext = projects.filter((p) => p.noNextAction).length;
  const now = next.slice().sort((a, b) => (a.priority || 'P9').localeCompare(b.priority || 'P9')).slice(0, 12);
  const label = projectsNoNext > 0 || inbox.length > 15 ? 'à revoir' : (inbox.length > 0 || waitingRich.length > 0 ? 'stable' : 'clair');
  const led: 'ok' | 'warn' | 'down' = projectsNoNext > 0 ? 'down' : (label === 'à revoir' ? 'warn' : 'ok');
  const newest = (items: Array<{ data: Record<string, unknown> }>) => items.map((x) => asString(x.data.updated_at ?? x.data.updated)).filter(Boolean).sort().reverse()[0] || 'n/a';
  return {
    source: root,
    updated: {
      inbox: newest(actions.filter(({ data }) => isInboxStatus(normalizedStatus(data.status)))),
      next: newest(actions.filter(({ task }) => next.includes(task))),
      projects: newest(projectReads),
      waiting: newest(actions.filter(({ data }) => isWaitingStatus(normalizedStatus(data.status)))),
      someday: newest(actions.filter(({ data }) => isSomedayStatus(normalizedStatus(data.status)))),
      review: 'folder-schema'
    },
    inbox,
    next,
    doneRecent,
    projects,
    waiting: waitingRich,
    someday,
    review: projectsNoNext ? [`Clarifier ${projectsNoNext} projet(s) actif(s) sans prochaine action liée.`] : [],
    now,
    trust: { inboxOpen: inbox.length, nextOpen: next.length, p1Open, projectsActive, projectsNoNext, waiting: waitingRich.length, someday: someday.length, reviewUpdated: 'folder-schema', label, led }
  };
}

async function readLegacyGtdCockpit(root: string): Promise<GtdCockpit> {
  const [inboxRaw, nextRaw, projectsRaw, waitingRaw, somedayRaw, reviewRaw] = await Promise.all([
    readTextIf(root, 'inbox.md'),
    readTextIf(root, 'next-actions.md'),
    readTextIf(root, 'projects.md'),
    readTextIf(root, 'waiting-for.md'),
    readTextIf(root, 'someday-maybe.md'),
    readTextIf(root, 'review.md')
  ]);
  const inboxAll = parseChecklist(inboxRaw, 'inbox.md');
  const nextAll = parseChecklist(nextRaw, 'next-actions.md');
  const somedayAll = parseChecklist(somedayRaw, 'someday-maybe.md');
  const projects = parseProjects(projectsRaw, 'projects.md');
  const waiting = parseWaiting(waitingRaw, 'waiting-for.md');
  const inbox = inboxAll.filter((t) => !t.checked);
  const next = nextAll.filter((t) => !t.checked);
  const doneRecent = [...inboxAll, ...nextAll].filter((t) => t.checked).slice(-8).reverse();
  const p1Open = next.filter((t) => t.priority === 'P1').length;
  const projectsActive = projects.filter((p) => p.active).length;
  const projectsNoNext = projects.filter((p) => p.noNextAction).length;
  const now = next.slice().sort((a, b) => (a.priority || 'P9').localeCompare(b.priority || 'P9')).slice(0, 8);
  const label = projectsNoNext > 0 || inbox.length > 15 ? 'à revoir' : (inbox.length > 0 || waiting.length > 0 ? 'stable' : 'clair');
  const led: 'ok' | 'warn' | 'down' = projectsNoNext > 0 ? 'down' : (label === 'à revoir' ? 'warn' : 'ok');
  return {
    source: root,
    updated: {
      inbox: updatedFrom(inboxRaw) || 'n/a',
      next: updatedFrom(nextRaw) || 'n/a',
      projects: updatedFrom(projectsRaw) || 'n/a',
      waiting: updatedFrom(waitingRaw) || 'n/a',
      someday: updatedFrom(somedayRaw) || 'n/a',
      review: updatedFrom(reviewRaw) || 'n/a'
    },
    inbox,
    next,
    doneRecent,
    projects,
    waiting,
    someday: somedayAll.filter((t) => !t.checked),
    review: parseReview(reviewRaw),
    now,
    trust: { inboxOpen: inbox.length, nextOpen: next.length, p1Open, projectsActive, projectsNoNext, waiting: waiting.length, someday: somedayAll.filter((t) => !t.checked).length, reviewUpdated: updatedFrom(reviewRaw), label, led }
  };
}

async function readTextIf(root: string, rel: string) {
  const p = path.join(root, rel);
  try { return await fs.readFile(p, 'utf8'); } catch { return ''; }
}

export async function readGtdCockpit(root: string): Promise<GtdCockpit> {
  return (await readFolderGtdCockpit(root)) || readLegacyGtdCockpit(root);
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
