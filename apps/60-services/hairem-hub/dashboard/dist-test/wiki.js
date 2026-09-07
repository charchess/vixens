import fs from 'node:fs/promises';
import path from 'node:path';
import matter from 'gray-matter';
async function exists(p) { try {
    await fs.access(p);
    return true;
}
catch {
    return false;
} }
async function walk(dir, root = dir) { if (!(await exists(dir)))
    return []; const entries = await fs.readdir(dir, { withFileTypes: true }); const out = []; for (const e of entries) {
    if (e.name === '.git' || e.name === 'archive')
        continue;
    const full = path.join(dir, e.name);
    if (e.isDirectory())
        out.push(...await walk(full, root));
    else if (e.isFile() && e.name.endsWith('.md'))
        out.push(path.relative(root, full));
} return out.sort(); }
function arr(v) { if (Array.isArray(v))
    return v.map(String).filter(Boolean); if (v === null || v === undefined || v === '')
    return []; return String(v).split(/[,;]/).map(s => s.trim()).filter(Boolean); }
function val(v) { if (v === null || v === undefined)
    return undefined; const s = String(v).trim(); return s && s !== 'null' ? s : undefined; }
function nullable(v) { return val(v) ?? null; }
function titleFromMarkdown(body, fallback) { return body.match(/^#\s+(.+)$/m)?.[1]?.trim() || fallback.replace(/\.md$/, '').replaceAll('-', ' '); }
function clean(s = '') { return s.replace(/`/g, '').replace(/\s+/g, ' ').trim(); }
function excerpt(body) { return body.replace(/^#\s+.+$/m, '').split('\n').map(clean).filter(Boolean).filter(x => !x.startsWith('| Champ')).slice(0, 4).join(' · ').slice(0, 260); }
function lineField(body, ...labels) { for (const label of labels) {
    const rx = new RegExp(`(?:^|\\n)\\s*(?:[-*]\\s*)?${label}\\s*:\\s*(.+)`, 'i');
    const m = body.match(rx);
    if (m)
        return clean(m[1]);
} return undefined; }
// People are canonical frontmatter only. Linguistic inference produced false contacts
// (e.g. "une" from "contacter une première vague") and must not silently mutate UI metadata.
function inferPeople(_task) { return []; }
function parseFrontmatterTask(rel, raw) {
    const parsed = matter(raw);
    const d = parsed.data;
    const title = String(d.title || titleFromMarkdown(parsed.content, path.basename(rel)));
    const status = val(d.status) || 'inbox';
    const task = { id: val(d.id), title, path: rel, section: val(d.source_section) || path.dirname(rel), checked: status === 'done', status, isNext: Boolean(d.is_next) || status === 'next', priority: val(d.priority), project: val(d.project) || val(d.project_title), projectId: nullable(d.project_id), contexts: arr(d.contexts).length ? arr(d.contexts) : arr(d.context), tags: arr(d.tags), people: [], finish: val(d.finish), criterion: val(d.finish) || lineField(parsed.content, 'Critère fini', 'Fini quand'), guardrail: lineField(parsed.content, 'Garde-fou', 'Contrainte'), statusText: lineField(parsed.content, 'Statut', 'Résultat'), excerpt: excerpt(parsed.content), dueAt: nullable(d.due_at), scheduledAt: nullable(d.scheduled_at), followupAt: nullable(d.followup_at), postponedUntil: nullable(d.postponed_until), updatedAt: nullable(d.updated_at), waitingOnRefId: nullable(d.waiting_on_ref_id) };
    task.people = [...new Set([...arr(d.people), ...inferPeople(task)])];
    return task;
}
function parseReference(rel, raw) { const p = matter(raw); const d = p.data; return { id: val(d.id), title: String(d.title || titleFromMarkdown(p.content, path.basename(rel))), path: rel, projectId: nullable(d.project_id), actionId: nullable(d.action_id), tags: arr(d.tags), updatedAt: nullable(d.updated_at), excerpt: excerpt(p.content) }; }
function parseProject(rel, raw, actions, refs) { const p = matter(raw); const d = p.data; const id = val(d.id); const title = String(d.title || titleFromMarkdown(p.content, path.basename(rel))); const status = val(d.status) || 'active'; const linked = actions.filter(a => a.projectId === id || a.project === title); const next = linked.filter(a => !a.checked && (a.isNext || a.status === 'next')); const waiting = linked.filter(a => ['waiting', 'waiting_for', 'blocked', 'scheduled'].includes(a.status || '')); const referenceCount = refs.filter(r => r.projectId === id).length; const body = p.content; const extractedNext = lineField(body, 'Prochaine action', 'Prochaine action disponible', 'Next action'); const active = !['done', 'someday', 'maybe', 'parking', 'parked', 'archive', 'archived', 'cancelled'].includes(status.toLowerCase()); const nextAction = next[0]?.title || extractedNext; const guardrail = lineField(body, 'Garde-fou', 'Déclencheur/garde-fou'); const noNextAction = active && !nextAction && waiting.length === 0; const noGuidance = noNextAction && !guardrail; return { id, title, path: rel, status, altitude: val(d.altitude), dueAt: nullable(d.due_at), importance: nullable(d.importance), tags: arr(d.tags), result: lineField(body, 'Résultat voulu', 'Outcome'), nextAction, criterion: lineField(body, 'Critère fini', 'Critère fini de la prochaine action'), guardrail, personas: lineField(body, 'Personas utiles', 'Persona utile'), active, noNextAction, noGuidance, linkedActions: linked.length, nextActions: next.length, waiting: waiting.length, references: referenceCount, updatedAt: nullable(d.updated_at), excerpt: excerpt(body) }; }
function updatedMax(items) { return items.map(i => i.updatedAt).filter(Boolean).sort().reverse()[0] || 'n/a'; }
function sortActions(a, b) { const pa = (a.priority || 'P9').localeCompare(b.priority || 'P9'); if (pa)
    return pa; return (b.updatedAt || '').localeCompare(a.updatedAt || ''); }
function stripPriority(title) { const m = title.match(/^(P\d+)\s*[—-]\s*(.+)$/); return { priority: m?.[1], title: (m?.[2] || title).trim() }; }
function detailsToTask(titleRaw, checked, details, rel, section) { const sp = stripPriority(clean(titleRaw)); const get = (...labels) => { for (const line of details) {
    const n = line.replace(/^\s*-\s*/, '').trim();
    for (const label of labels) {
        const m = n.match(new RegExp(`^${label}\\s*:\\s*(.+)$`, 'i'));
        if (m)
            return clean(m[1]);
    }
} return undefined; }; const t = { title: sp.title, path: rel, section, checked, status: checked ? 'done' : 'next', isNext: !checked, priority: sp.priority, project: get('Projet', 'Projet/contexte'), projectId: null, contexts: [section], tags: [], people: [], criterion: get('Critère fini', 'Critère de reprise'), guardrail: get('Garde-fou', 'Contrainte'), statusText: get('Statut', 'Résultat'), excerpt: details.map(d => clean(d.replace(/^\s*-\s*/, ''))).filter(Boolean).slice(0, 2).join(' · ').slice(0, 240) }; t.people = inferPeople(t); return t; }
function parseChecklist(raw, rel) { const lines = raw.split('\n'); const tasks = []; let section = 'GTD'; let current = null; const flush = () => { if (current) {
    tasks.push(detailsToTask(current.title, current.checked, current.details, rel, current.section));
    current = null;
} }; for (const line of lines) {
    const h = line.match(/^##\s+(.+)$/);
    if (h) {
        flush();
        section = clean(h[1]);
        continue;
    }
    const item = line.match(/^\s*-\s*\[([ xX])\]\s+(.+)$/);
    if (item) {
        flush();
        current = { checked: item[1].toLowerCase() === 'x', title: item[2], details: [], section };
    }
    else if (current && /^\s{2,}-\s+/.test(line))
        current.details.push(line);
} flush(); return tasks; }
function parseFlatProjects(raw, rel) { const blocks = raw.split(/\n(?=###\s+)/).filter(b => b.trim().startsWith('###')); return blocks.map(b => { const title = clean(b.match(/^###\s+(.+)$/m)?.[1] || 'Projet'); const next = lineField(b, 'Prochaine action', 'Prochaine action du projet'); const status = lineField(b, 'Statut', 'État') || 'active'; const active = !/parking|terminé|termine|someday/.test(status.toLowerCase()); return { title, path: rel, status, tags: [], result: lineField(b, 'Résultat voulu'), nextAction: next, criterion: lineField(b, 'Critère fini'), guardrail: lineField(b, 'Garde-fou'), personas: lineField(b, 'Personas utiles'), active, noNextAction: active && !next, noGuidance: active && !next && !lineField(b, 'Garde-fou'), linkedActions: 0, nextActions: next ? 1 : 0, waiting: 0, references: 0, excerpt: excerpt(b) }; }); }
async function readFolderCockpit(root) {
    const actionFiles = await walk(path.join(root, 'actions'), path.join(root, 'actions'));
    const refFiles = await walk(path.join(root, 'references'), path.join(root, 'references'));
    const actions = await Promise.all(actionFiles.map(async (rel) => parseFrontmatterTask(`actions/${rel}`, await fs.readFile(path.join(root, 'actions', rel), 'utf8'))));
    const refs = await Promise.all(refFiles.map(async (rel) => parseReference(`references/${rel}`, await fs.readFile(path.join(root, 'references', rel), 'utf8'))));
    const projectFiles = await walk(path.join(root, 'projects'), path.join(root, 'projects'));
    const projects = await Promise.all(projectFiles.map(async (rel) => parseProject(`projects/${rel}`, await fs.readFile(path.join(root, 'projects', rel), 'utf8'), actions, refs)));
    const inbox = actions.filter(a => !a.checked && a.status === 'inbox');
    const next = actions.filter(a => !a.checked && (a.isNext || a.status === 'next')).sort(sortActions);
    const waiting = actions.filter(a => !a.checked && ['waiting', 'waiting_for', 'blocked', 'scheduled'].includes(a.status || '')).map(a => ({ ...a, expected: a.finish || a.criterion, actionAfter: a.followupAt || a.guardrail }));
    const someday = actions.filter(a => !a.checked && ['someday', 'maybe', 'parked'].includes(a.status || ''));
    const doneRecent = actions.filter(a => a.checked).sort((a, b) => (b.updatedAt || '').localeCompare(a.updatedAt || '')).slice(0, 8);
    const p1Open = next.filter(t => t.priority === 'P1').length;
    const projectsActive = projects.filter(p => p.active).length;
    const projectsNoNext = projects.filter(p => p.noNextAction).length;
    const projectsNoGuidance = projects.filter(p => p.noGuidance).length;
    const contextOptions = [...new Set(actions.flatMap(a => a.contexts))].filter(Boolean).sort();
    const tagOptions = [...new Set(actions.flatMap(a => a.tags))].filter(Boolean).sort();
    const peopleOptions = [...new Set(actions.flatMap(a => a.people))].filter(Boolean).sort();
    const label = projectsNoNext > 0 || inbox.length > 15 ? 'à revoir' : (inbox.length || waiting.length ? 'stable' : 'clair');
    const led = projectsNoNext > 0 ? 'warn' : 'ok';
    return { source: root, schema: 'folder', updated: { actions: updatedMax(actions), projects: updatedMax(projects), references: updatedMax(refs), next: updatedMax(next), inbox: updatedMax(inbox), waiting: updatedMax(waiting) }, inbox, next, doneRecent, projects, waiting, someday, references: refs, review: [...(projectsNoNext ? [`${projectsNoNext} projet(s) actif(s) sans prochaine action ni attente externe`] : []), ...(projectsNoGuidance ? [`${projectsNoGuidance} projet(s) sans action, attente ni déclencheur de reprise`] : []), ...(inbox.length ? [`${inbox.length} capture(s) inbox à clarifier avec Katia`] : []), ...(next.filter(a => !a.contexts.length && !a.tags.length).length ? [`${next.filter(a => !a.contexts.length && !a.tags.length).length} next action(s) sans contexte/tag`] : [])], now: next, contextOptions, tagOptions, peopleOptions, trust: { inboxOpen: inbox.length, nextOpen: next.length, p1Open, projectsActive, projectsNoNext, waiting: waiting.length, someday: someday.length, references: refs.length, label, led } };
}
async function readFlatCockpit(root) { const read = async (rel) => { try {
    return await fs.readFile(path.join(root, rel), 'utf8');
}
catch {
    return '';
} }; const [inboxRaw, nextRaw, projectsRaw, waitingRaw, somedayRaw, reviewRaw] = await Promise.all(['inbox.md', 'next-actions.md', 'projects.md', 'waiting-for.md', 'someday-maybe.md', 'review.md'].map(read)); const inbox = parseChecklist(inboxRaw, 'inbox.md').filter(t => !t.checked); const next = parseChecklist(nextRaw, 'next-actions.md').filter(t => !t.checked); const projects = parseFlatProjects(projectsRaw, 'projects.md'); const waiting = parseChecklist(waitingRaw, 'waiting-for.md').filter(t => !t.checked); const someday = parseChecklist(somedayRaw, 'someday-maybe.md').filter(t => !t.checked); const projectsNoNext = projects.filter(p => p.noNextAction).length; return { source: root, schema: 'flat', updated: { next: 'flat', projects: 'flat' }, inbox, next, doneRecent: [], projects, waiting, someday, references: [], review: parseChecklist(reviewRaw, 'review.md').filter(t => !t.checked).map(t => t.title), now: next.sort(sortActions), contextOptions: [...new Set(next.flatMap(a => a.contexts))].sort(), tagOptions: [], peopleOptions: [...new Set(next.flatMap(a => a.people))].sort(), trust: { inboxOpen: inbox.length, nextOpen: next.length, p1Open: next.filter(t => t.priority === 'P1').length, projectsActive: projects.filter(p => p.active).length, projectsNoNext, waiting: waiting.length, someday: someday.length, references: 0, label: projectsNoNext ? 'à revoir' : 'stable', led: projectsNoNext ? 'warn' : 'ok' } }; }
export async function readGtdCockpit(root) { if (await exists(path.join(root, 'actions')))
    return readFolderCockpit(root); if (await exists(path.join(root, 'next-actions.md')))
    return readFlatCockpit(root); return { source: root, schema: 'empty', updated: {}, inbox: [], next: [], doneRecent: [], projects: [], waiting: [], someday: [], references: [], review: ['Aucune source GTD trouvée'], now: [], contextOptions: [], tagOptions: [], peopleOptions: [], trust: { inboxOpen: 0, nextOpen: 0, p1Open: 0, projectsActive: 0, projectsNoNext: 0, waiting: 0, someday: 0, references: 0, label: 'vide', led: 'down' } }; }
export async function readWiki(root, limit = 80) { const files = (await walk(root)).slice(0, limit); const items = []; for (const rel of files) {
    const raw = await fs.readFile(path.join(root, rel), 'utf8');
    const p = matter(raw);
    const d = p.data;
    items.push({ title: String(d.title || titleFromMarkdown(p.content, path.basename(rel))), path: rel, status: val(d.status), owner: val(d.owner), project: val(d.project) || val(d.project_id), context: arr(d.contexts)[0] || val(d.context), priority: val(d.priority), updated: val(d.updated_at) || val(d.updated), excerpt: excerpt(p.content) });
} return items; }
export function countBy(items, key) { return items.reduce((acc, item) => { const value = item[key] ? String(item[key]) : 'none'; acc[value] = (acc[value] || 0) + 1; return acc; }, {}); }
