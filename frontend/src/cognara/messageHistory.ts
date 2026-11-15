import { APIError, formatAPIError } from "./api";
import type { Message } from "./types";
export interface MessagePage { success: boolean; data: Message[]; has_more: boolean; next_before_message_id: string | null }
export class MessageHistory {
  items: Message[] = [];
  cursor: string | null = null;
  loading = false;
  error = "";
  private generation = 0;
  constructor(private fetchPage: (cursor: string | null) => Promise<MessagePage>) {}
  async refresh() {
    const generation = ++this.generation;
    this.items = []; this.cursor = null;
    await this.load(generation, null);
  }
  async older() {
    if (this.loading || this.cursor === null) return;
    await this.load(this.generation, this.cursor);
  }
  private async load(generation: number, cursor: string | null) {
    this.loading = true; this.error = "";
    try {
      const page = await this.fetchPage(cursor);
      if (generation !== this.generation) return;
      if (page.has_more && (!page.next_before_message_id || page.next_before_message_id === cursor)) throw new Error("消息游标未前进，请刷新记录");
      const ids = new Set(this.items.map(m => m.id));
      this.items = [...page.data.filter(m => !ids.has(m.id)), ...this.items];
      this.cursor = page.has_more ? page.next_before_message_id : null;
    } catch (e) {
      if (generation !== this.generation) return;
      this.error = formatAPIError(e);
      if (e instanceof APIError && [401, 403, 404].includes(e.status)) { this.items = []; this.cursor = null; }
    } finally { if (generation === this.generation) this.loading = false; }
  }
}
