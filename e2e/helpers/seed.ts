/**
 * API seed helpers for e2e specs. Seeding goes through the JSON admin API
 * (no CSRF on /api/admin/login — the UI form login is a separate path) so
 * specs never depend on admin UI state. Waffle slugs get a random 8-char
 * suffix server-side, so every create is collision-free across reruns.
 */

export interface CreateWaffleInput {
  readonly title: string;
  readonly total_spots: number;
  readonly spot_price: number;
}

export interface SeededWaffle {
  readonly id: string;
  readonly slug: string;
  readonly title: string;
  readonly total_spots: number;
  readonly spot_price: number;
  readonly status: string;
}

async function postJson(
  url: string,
  body: unknown,
  headers: Record<string, string>,
): Promise<unknown> {
  let response: Response;
  try {
    response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...headers },
      body: JSON.stringify(body),
    });
  } catch (cause) {
    const detail = cause instanceof Error ? cause.message : String(cause);
    throw new Error(`POST ${url} failed: ${detail}`);
  }
  if (!response.ok) {
    const text = await response.text();
    throw new Error(`POST ${url} returned ${response.status}: ${text}`);
  }
  return response.json();
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function requireString(
  record: Record<string, unknown>,
  field: string,
  url: string,
): string {
  const value = record[field];
  if (typeof value !== 'string' || value === '') {
    throw new Error(`POST ${url} response missing string field "${field}"`);
  }
  return value;
}

function requireNumber(
  record: Record<string, unknown>,
  field: string,
  url: string,
): number {
  const value = record[field];
  if (typeof value !== 'number') {
    throw new Error(`POST ${url} response missing numeric field "${field}"`);
  }
  return value;
}

export async function loginAPI(baseURL: string): Promise<string> {
  const url = `${baseURL}/api/admin/login`;
  const body = await postJson(url, { username: 'admin', password: 'syrup' }, {});
  if (!isRecord(body)) {
    throw new Error(`POST ${url} response is not a JSON object`);
  }
  return requireString(body, 'token', url);
}

export async function createWaffleAPI(
  baseURL: string,
  token: string,
  input: CreateWaffleInput,
): Promise<SeededWaffle> {
  const url = `${baseURL}/api/admin/waffles`;
  const body = await postJson(url, input, { Authorization: `Bearer ${token}` });
  if (!isRecord(body)) {
    throw new Error(`POST ${url} response is not a JSON object`);
  }
  return {
    id: requireString(body, 'id', url),
    slug: requireString(body, 'slug', url),
    title: requireString(body, 'title', url),
    total_spots: requireNumber(body, 'total_spots', url),
    spot_price: requireNumber(body, 'spot_price', url),
    status: requireString(body, 'status', url),
  };
}
