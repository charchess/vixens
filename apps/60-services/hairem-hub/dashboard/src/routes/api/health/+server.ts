import { json } from '@sveltejs/kit';

export function GET() {
  return json({ ok: true, service: 'hairem-dashboard', version: '0.1.0' });
}
