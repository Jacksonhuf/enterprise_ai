export class APIError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
  }
}
export const credential = () =>
  localStorage.getItem("cognara_demo_identity") || "";
export async function api<T = any>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${credential()}`,
      ...init.headers,
    },
  });
  const data = await response.json();
  if (!response.ok)
    throw new APIError(
      response.status,
      data.error?.code || "REQUEST_FAILED",
      data.error?.message || "请求未完成",
    );
  return data;
}
export const post = <T = any>(path: string, data: unknown) =>
  api<T>(path, { method: "POST", body: JSON.stringify(data) });
export const put = <T = any>(path: string, data: unknown) =>
  api<T>(path, { method: "PUT", body: JSON.stringify(data) });
export const prefix = "/api/v1/cognitive";
// SSE parser handles split CRLF/frame boundaries and consumes only the existing message stream.
export async function sendTurn(
  sessionId: string,
  query: string,
  onEvent: (type: string, data: any) => void,
  signal?: AbortSignal,
) {
  const response = await fetch(
    `/api/v1/agent-chat/${encodeURIComponent(sessionId)}`,
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${credential()}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ query }),
      signal,
    },
  );
  if (!response.ok) {
    const e = await response.json();
    throw new APIError(
      response.status,
      e.error?.code,
      e.error?.message || "运行失败",
    );
  }
  if (!response.body) throw new Error("响应没有消息流");
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  try {
    for (;;) {
      const { done, value } = await reader.read();
      buffer += decoder.decode(value, { stream: !done });
      if (buffer.length > 2 * 1024 * 1024)
        throw new Error("消息流超过大小限制");
      buffer = buffer.replace(/\r\n/g, "\n");
      let split: number;
      while ((split = buffer.indexOf("\n\n")) >= 0) {
        const frame = buffer.slice(0, split);
        buffer = buffer.slice(split + 2);
        let type = "message";
        const lines: string[] = [];
        for (const line of frame.split("\n")) {
          if (line.startsWith("event:")) type = line.slice(6).trim();
          if (line.startsWith("data:")) lines.push(line.slice(5).trimStart());
        }
        if (lines.length) onEvent(type, JSON.parse(lines.join("\n")));
      }
      if (done) break;
    }
  } finally {
    await reader.cancel().catch(() => {});
    reader.releaseLock();
  }
}
