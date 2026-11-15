import type { Application } from "./types";
interface Page { items: Application[]; next_cursor?: string | null }
// Do not expose a partial selector if any page fails or the cursor cycles.
export async function loadApplicationOptions(fetchPage: (cursor: string | null) => Promise<Page>): Promise<Application[]> {
  const items = new Map<string, Application>();
  const seen = new Set<string>();
  let cursor: string | null = null;
  for (;;) {
    const page = await fetchPage(cursor);
    for (const item of page.items) items.set(item.app_id, item);
    const next = page.next_cursor ?? null;
    if (next === null) return [...items.values()];
    if (seen.has(next)) throw new Error("应用选项分页未前进，请重试加载");
    seen.add(next);
    cursor = next;
  }
}
