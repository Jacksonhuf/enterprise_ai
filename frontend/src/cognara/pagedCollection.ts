export interface Page<T> { items: T[]; next_cursor?: string | null }

// A new search invalidates earlier requests; a failed next page retains its cursor.
export class PagedCollection<T> {
  items: T[] = [];
  nextCursor: string | null = null;
  loading = false;
  error = "";
  query = "";
  private generation = 0;
  constructor(
    private fetchPage: (query: string, cursor: string | null) => Promise<Page<T>>,
    private key: (item: T) => string,
  ) {}
  async reset(query = "") {
    const generation = ++this.generation;
    this.query = query;
    this.items = [];
    this.nextCursor = null;
    await this.load(generation, null);
  }
  async next() {
    if (this.loading || this.nextCursor === null) return;
    await this.load(this.generation, this.nextCursor);
  }
  private async load(generation: number, cursor: string | null) {
    this.loading = true;
    this.error = "";
    try {
      const page = await this.fetchPage(this.query, cursor);
      if (generation !== this.generation) return;
      const next = page.next_cursor ?? null;
      if (next !== null && next === cursor) throw new Error("分页游标未前进，请刷新后重试");
      const items = new Map(this.items.map(item => [this.key(item), item]));
      for (const item of page.items) items.set(this.key(item), item);
      this.items = [...items.values()];
      this.nextCursor = next;
    } catch (e) {
      if (generation === this.generation) this.error = (e as Error).message;
    } finally {
      if (generation === this.generation) this.loading = false;
    }
  }
}
