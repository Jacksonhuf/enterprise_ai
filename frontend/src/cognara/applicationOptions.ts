import type { Application } from "./types";
interface Page<T> { items: T[]; next_cursor?: string | null }
// Do not expose a partial selector if any page fails or the cursor cycles.
export async function loadAllOptions<T>(fetchPage: (cursor: string | null) => Promise<Page<T>>, key: (item: T) => string): Promise<T[]> {
  const items = new Map<string, T>();
  const seen = new Set<string>();
  let cursor: string | null = null;
  for (;;) {
    const page = await fetchPage(cursor);
    for (const item of page.items) items.set(key(item), item);
    const next = page.next_cursor ?? null;
    if (next === null) return [...items.values()];
    if (seen.has(next)) throw new Error("选项分页未前进，请重试加载");
    seen.add(next);
    cursor = next;
  }
}

export const loadApplicationOptions = (fetchPage: (cursor: string | null) => Promise<Page<Application>>) => loadAllOptions(fetchPage, item => item.app_id);
