import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import fs from 'node:fs/promises';
import path from 'node:path';

const AUDIT_DIR = process.env.GTD_AUDIT_DIR || '/tmp/hairem-dashboard-audit';
const KATIA_WEBHOOK_URL = process.env.KATIA_WEBHOOK_URL || '';

type GtdEvent = {
  event?: string;
  actor?: string;
  timestamp?: string;
  files?: string[];
  items?: Array<Record<string, unknown>>;
};

function safeEvent(input: GtdEvent) {
  return {
    event: input.event || 'gtd.source.modified',
    actor: input.actor || 'dashboard',
    timestamp: input.timestamp || new Date().toISOString(),
    files: Array.isArray(input.files) ? input.files.slice(0, 20) : [],
    items: Array.isArray(input.items) ? input.items.slice(0, 50) : [],
    needs_katia_review: true
  };
}

export const POST: RequestHandler = async ({ request }) => {
  const body = safeEvent(await request.json().catch(() => ({})));
  await fs.mkdir(AUDIT_DIR, { recursive: true });
  const stamp = body.timestamp.replace(/[:.]/g, '').replace(/[^0-9TZ-]/g, '');
  const file = path.join(AUDIT_DIR, `${stamp}-${body.event}.json`);
  await fs.writeFile(file, `${JSON.stringify(body, null, 2)}\n`, { mode: 0o660 });

  let webhook = { configured: Boolean(KATIA_WEBHOOK_URL), posted: false, status: 0 };
  if (KATIA_WEBHOOK_URL) {
    try {
      const res = await fetch(KATIA_WEBHOOK_URL, {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(body),
        signal: AbortSignal.timeout(3500)
      });
      webhook = { configured: true, posted: res.ok, status: res.status };
    } catch {
      webhook = { configured: true, posted: false, status: 0 };
    }
  }

  return json({ ok: true, audit_file: file, webhook });
};
