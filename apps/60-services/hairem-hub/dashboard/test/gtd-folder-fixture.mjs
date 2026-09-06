import { mkdtemp, mkdir, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { readGtdCockpit } from '../dist-test/wiki.js';

const root = await mkdtemp(path.join(tmpdir(), 'gtd-folder-'));
try {
  await mkdir(path.join(root, 'actions'), { recursive: true });
  await mkdir(path.join(root, 'projects'), { recursive: true });
  await mkdir(path.join(root, 'references'), { recursive: true });
  await mkdir(path.join(root, 'archive'), { recursive: true });
  await writeFile(path.join(root, 'actions', 'call-admin.md'), `---\nid: action:call-admin\ntitle: "Call admin"\nstatus: next\nproject_id: project:admin\nis_next: true\ncontexts: ["admin", "5-15 min"]\ntags: ["sensible"]\nfinish: "mail envoyé"\nupdated_at: 2026-09-06\nsource_file: next-actions.md\nsource_section: "@admin"\n---\n# Call admin\n\n## Détail extrait\n- Critère fini : mail envoyé\n`);
  await writeFile(path.join(root, 'actions', 'wait-admin.md'), `---\nid: action:wait-admin\ntitle: "Wait admin"\nstatus: waiting_for\nproject_id: project:admin\nis_next: false\ncontexts: ["En attente"]\nwaiting_on_ref_id: ref:x\nupdated_at: 2026-09-06\n---\n# Wait admin\n\n## Détail extrait\n- Attendu : réponse externe\n- Action après déclencheur : relancer\n`);
  await writeFile(path.join(root, 'actions', 'capture.md'), `---\nid: action:capture\ntitle: "Capture raw"\nstatus: inbox\nis_next: false\ncontexts: ["Inbox"]\nupdated_at: 2026-09-06\n---\n# Capture raw\n`);
  await writeFile(path.join(root, 'actions', 'later.md'), `---\nid: action:later\ntitle: "Later maybe"\nstatus: someday\nis_next: false\ncontexts: ["Someday"]\nupdated_at: 2026-09-06\n---\n# Later maybe\n`);
  await writeFile(path.join(root, 'actions', 'done.md'), `---\nid: action:done\ntitle: "Done old"\nstatus: done\nis_next: false\nupdated_at: 2026-09-06\n---\n# Done old\n`);
  await writeFile(path.join(root, 'projects', 'admin.md'), `---\nid: project:admin\ntitle: "Admin project"\nstatus: active\ntags: ["admin"]\nupdated_at: 2026-09-06\n---\n# Admin project\n\n## Notes projet extraites\n- Résultat voulu : dossiers rangés.\n- Prochaine action : Call admin.\n- Garde-fou : ne pas publier.\n`);
  await writeFile(path.join(root, 'projects', 'stale.md'), `---\nid: project:stale\ntitle: "Stale active"\nstatus: active\nupdated_at: 2026-09-06\n---\n# Stale active\n\n## Notes projet extraites\n- Résultat voulu : clarifier.\n`);
  const c = await readGtdCockpit(root);
  const checks = [
    ['nextOpen', c.trust.nextOpen, 1],
    ['inboxOpen', c.trust.inboxOpen, 1],
    ['waiting', c.trust.waiting, 1],
    ['someday', c.trust.someday, 1],
    ['projectsActive', c.trust.projectsActive, 2],
    ['projectsNoNext', c.trust.projectsNoNext, 1],
    ['doneRecent', c.doneRecent.length, 1]
  ];
  const bad = checks.filter(([, got, exp]) => got !== exp);
  if (bad.length) {
    console.error(JSON.stringify({ bad, cockpit: c }, null, 2));
    process.exit(1);
  }
  console.log(JSON.stringify({ ok: true, trust: c.trust, now: c.now.map(x => x.title), waiting: c.waiting.map(x => x.title) }));
} finally {
  await rm(root, { recursive: true, force: true });
}
