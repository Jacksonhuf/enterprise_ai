import { test } from "node:test";
import assert from "node:assert/strict";
import { api, APIError, sendTurn } from "./api";
Object.defineProperty(globalThis, "localStorage", {
  configurable: true,
  value: { getItem: () => "demo-user" },
});

test("SSE handles every UTF-8 byte and CRLF frame boundary", async (t) => {
  const text =
    'event: answer\r\ndata: {"content":"原始缺口276.6"}\r\n\r\nevent: done\r\ndata: {"done":true}\r\n\r\n';
  const bytes = new TextEncoder().encode(text);
  const frames: any[] = [];
  t.mock.method(globalThis, "fetch", async (_url: any, init: any) => {
    assert.equal(init.headers.Authorization, "Bearer demo-user");
    assert.deepEqual(JSON.parse(init.body), { query: "业务问题" });
    return new Response(
      new ReadableStream({
        start(c) {
          for (const b of bytes) c.enqueue(Uint8Array.of(b));
          c.close();
        },
      }),
    );
  });
  await sendTurn("session/one", "业务问题", (event, data) =>
    frames.push({ event, data }),
  );
  assert.deepEqual(frames, [
    { event: "answer", data: { content: "原始缺口276.6" } },
    { event: "done", data: { done: true } },
  ]);
});
test("Authorization errors retain their status and contract code", async (t) => {
  t.mock.method(
    globalThis,
    "fetch",
    async () =>
      new Response(
        JSON.stringify({
          error: { code: "AUTHORIZATION_DENIED", message: "无权访问" },
        }),
        { status: 403 },
      ),
  );
  await assert.rejects(
    () => api("/api/v1/cognitive/models"),
    (e: any) =>
      e instanceof APIError &&
      e.status === 403 &&
      e.code === "AUTHORIZATION_DENIED",
  );
  await assert.rejects(
    () => sendTurn("s", "query", () => {}),
    (e: any) => e instanceof APIError && e.status === 403,
  );
});
test("Malformed SSE is rejected and the underlying stream is canceled", async (t) => {
  let canceled = false;
  t.mock.method(
    globalThis,
    "fetch",
    async () =>
      new Response(
        new ReadableStream({
          start(c) {
            c.enqueue(
              new TextEncoder().encode("event: answer\ndata: invalid-json\n\n"),
            );
          },
          cancel() {
            canceled = true;
          },
        }),
      ),
  );
  await assert.rejects(() => sendTurn("s", "query", () => {}), SyntaxError);
  assert.equal(canceled, true);
});

test("Request tracing preserves safe IDs and rejects unsafe response identifiers", async (t) => {
  const { formatAPIError } = await import("./api");
  let index = 0;
  t.mock.method(
    globalThis,
    "fetch",
    async () =>
      new Response(
        JSON.stringify({
          request_id:
            index++ === 0 ? "request_trace123" : "untrusted/id?value=secret",
          error: { code: "CAPABILITY_UNAVAILABLE", message: "服务不可用" },
        }),
        { status: 503 },
      ),
  );
  let first: any;
  try {
    await api("/failure");
  } catch (e) {
    first = e;
  }
  assert.equal(first.requestId, "request_trace123");
  assert.match(formatAPIError(first), /request_trace123/);
  let second: any;
  try {
    await api("/failure");
  } catch (e) {
    second = e;
  }
  assert.equal(second.requestId, undefined);
  assert.equal(formatAPIError(second), "服务不可用");
});
