import { ref } from "vue";
// Invalidate sidebar summaries after a saved turn without reloading the chat route.
export const sessionHistoryVersion = ref(0);
export function invalidateSessionHistory() { sessionHistoryVersion.value++; }
