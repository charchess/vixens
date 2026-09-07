import { json } from '@sveltejs/kit';

export function GET() {
  return json({ ok: true, service: 'hairem-dashboard', version: process.env.DASHBOARD_VERSION || '0.2.3' });
}
