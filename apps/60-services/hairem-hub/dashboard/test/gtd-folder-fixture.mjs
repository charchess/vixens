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
  await writeFile(path.join(root, 'actions', 'call-admin.md'), `---
id: action:call-admin
title: "Call admin"
status: next
project_id: project:admin
is_next: true
contexts: ["admin", "5-15 min"]
tags: ["sensible"]
finish: "mail envoyé"
updated_at: 2026-09-06
source_file: next-actions.md
source_section: "@admin"
---
# Call admin

## Détail extrait
- Critère fini : mail envoyé
`);
  await writeFile(path.join(root, 'actions', 'wait-admin.md'), `---
id: action:wait-admin
title: "Wait admin"
status: waiting_for
project_id: project:waiting-only
is_next: false
contexts: ["En attente"]
waiting_on_ref_id: ref:x
updated_at: 2026-09-06
---
# Wait admin

## Détail extrait
- Attendu : réponse externe
- Action après déclencheur : relancer
`);
  await writeFile(path.join(root, 'actions', 'capture.md'), `---
id: action:capture
title: "Capture raw"
status: inbox
is_next: false
contexts: ["Inbox"]
updated_at: 2026-09-06
---
# Capture raw
`);
  await writeFile(path.join(root, 'actions', 'later.md'), `---
id: action:later
title: "Later maybe"
status: someday
is_next: false
contexts: ["Someday"]
updated_at: 2026-09-06
---
# Later maybe
`);
  await writeFile(path.join(root, 'actions', 'done.md'), `---
id: action:done
title: "Done old"
status: done
is_next: false
updated_at: 2026-09-06
---
# Done old
`);
  await writeFile(path.join(root, 'projects', 'admin.md'), `---
id: project:admin
title: "Admin project"
status: active
tags: ["admin"]
updated_at: 2026-09-06
---
# Admin project

## Notes projet extraites
- Résultat voulu : dossiers rangés.
- Prochaine action : Call admin.
- Garde-fou : ne pas publier.
`);
  await writeFile(path.join(root, 'projects', 'stale.md'), `---
id: project:stale
title: "Stale active"
status: active
updated_at: 2026-09-06
---
# Stale active

## Notes projet extraites
- Résultat voulu : clarifier.
`);
  await writeFile(path.join(root, 'projects', 'waiting-only.md'), `---
id: project:waiting-only
title: "Waiting only"
status: active
updated_at: 2026-09-06
---
# Waiting only

## Notes projet extraites
- Résultat voulu : attendre un retour externe.
`);
  const c = await readGtdCockpit(root);
  const checks = [
    ['nextOpen', c.trust.nextOpen, 1],
    ['inboxOpen', c.trust.inboxOpen, 1],
    ['waiting', c.trust.waiting, 1],
    ['someday', c.trust.someday, 1],
    ['projectsActive', c.trust.projectsActive, 3],
    ['projectsNoNext', c.trust.projectsNoNext, 1],
    ['doneRecent', c.doneRecent.length, 1]
  ];
  const bad = checks.filter(([, got, exp]) => got !== exp);
  if (bad.length) {
    console.error(JSON.stringify({ bad, cockpit: c }, null, 2));
    process.exit(1);
  }
  if (c.projects.find((project) => project.id === 'project:waiting-only')?.noNextAction) {
    throw new Error('A project with explicit external waiting must not enter the no-next-action review queue');
  }
  console.log(JSON.stringify({ ok: true, trust: c.trust, now: c.now.map(x => x.title), waiting: c.waiting.map(x => x.title) }));
} finally {
  await rm(root, { recursive: true, force: true });
}
