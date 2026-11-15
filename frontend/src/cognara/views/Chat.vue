<template>
  <section class="chat-page">
    <header class="chat-header">
      <div>
        <h1>{{ application.name || "应用对话" }}</h1>
        <span class="meta">当前应用会话 · 结果附来源与限制</span>
      </div>
      <span class="badge success">已发布</span>
    </header>
    <t-alert v-if="error" theme="error" :message="error" />
    <t-alert v-if="history.error" theme="error" :message="history.error" />
    <div class="history-actions">
      <t-button :disabled="running" :loading="history.loading" variant="text" @click="refreshHistory">刷新记录</t-button>
      <t-button v-if="history.cursor" :disabled="running" :loading="history.loading" variant="outline" @click="loadOlder">加载较早消息</t-button>
    </div>
    <div ref="scrollArea" class="chat-scroll">
      <div v-if="!messages.length" class="chat-welcome">
        <img src="/brand/cognara-mark.svg" alt="" />
        <h2>从一个业务问题开始</h2>
        <p class="muted">{{ application.welcome }}</p>
        <div class="suggestions">
          <button
            v-for="q in application.suggested_questions"
            :key="q"
            @click="draft = q"
          >
            {{ q }}
          </button>
        </div>
      </div>
      <article
        v-for="m in messages"
        :key="m.id"
        class="message"
        :class="m.role"
      >
        <div class="message-label">
          {{ m.role === "user" ? "你" : application.name }}
        </div>
        <t-tag v-if="m.content_redacted" theme="warning" variant="light"
          >访问受限</t-tag
        >
        <div class="message-content">{{ m.content }}</div>
        <div v-if="m.result?.result?.rows" class="result-card">
          <div class="result-title">
            物料原始缺口 <span class="badge">单位：件</span>
          </div>
          <table class="data-table compact">
            <thead>
              <tr>
                <th
                  v-for="c in m.result.result.columns"
                  :key="c.name"
                  :class="{ numeric: c.data_type === 'DECIMAL' }"
                >
                  {{ c.name }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, i) in m.result.result.rows" :key="i">
                <td
                  v-for="(cell, j) in row"
                  :key="j"
                  :class="{ numeric: j > 0 }"
                >
                  {{ cell }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="m.role === 'assistant'" class="message-actions">
          <t-button
            v-if="m.execution_id"
            variant="outline"
            size="small"
            @click="showEvidence(m)"
            >查看依据与来源</t-button
          ><t-button variant="text" size="small" @click="copy(m.content)"
            >复制</t-button
          ><span class="meta">{{
            new Date(m.created_at).toLocaleTimeString("zh-CN")
          }}</span>
        </div>
      </article>
      <div v-if="running" class="processing">
        <t-loading size="small" /> 正在核对授权、业务时点与物料缺口…
      </div>
    </div>
    <div class="composer">
      <t-textarea
        v-model="draft"
        placeholder="例如：SO1002需求增加到350后，有多少物料缺口？"
        :autosize="{ minRows: 2, maxRows: 5 }"
        :maxlength="4000"
        @keydown="keydown"
      />
      <div class="composer-bottom">
        <span class="meta">Enter发送 · Shift+Enter换行</span
        ><t-button v-if="running" variant="outline" @click="stopReceiving"
          >停止接收</t-button
        ><t-button
          theme="primary"
          :loading="running"
          :disabled="!draft.trim() || running"
          @click="send"
          >发送</t-button
        >
      </div>
      <p class="chat-disclaimer">
        结果用于验证与业务复核。未纳入替代料及后续到货，不构成交期承诺。
      </p>
    </div>
    <t-drawer
      v-model:visible="evidenceVisible"
      header="依据与来源"
      size="640px"
      :footer="false"
      ><t-loading :loading="evidenceLoading"
        ><template v-if="evidence"
          ><t-tag theme="success" variant="light">受权只读投影</t-tag>
          <div class="detail-list">
            <div>
              <span>业务时点</span><strong>2026-04-02 · 历史验证快照</strong>
            </div>
            <div>
              <span>计算依据</span>
              <p>{{ evidence.calculation_basis.formula }}</p>
            </div>
            <div>
              <span>来源</span>
              <p v-for="(s, i) in evidence.source_references" :key="i">
                供应链合成种子 · 订单 {{ s.order }} · {{ s.as_of }}
              </p>
            </div>
          </div>
          <t-alert
            v-for="limit in evidence.calculation_basis.limitations"
            :key="limit"
            class="spaced"
            theme="warning"
            :message="limit" /></template
        ><t-alert
          v-if="evidenceError"
          theme="error"
          :message="evidenceError" /></t-loading
    ></t-drawer>
  </section>
</template>
<script setup lang="ts">
import { ref, reactive, computed, onMounted, nextTick } from "vue";
import { MessageHistory } from "../messageHistory";
import { useRoute, onBeforeRouteLeave } from "vue-router";
import { MessagePlugin } from "tdesign-vue-next";
import { api, prefix, sendTurn, formatAPIError } from "../api";
import type { Message } from "../types";
const route = useRoute(),
  application = ref<any>({ suggested_questions: [] }),
  draft = ref(""),
  running = ref(false),
  error = ref(""),
  scrollArea = ref<HTMLElement | null>(null),
  evidenceVisible = ref(false),
  evidence = ref<any>(null),
  evidenceError = ref(""),
  evidenceLoading = ref(false);
const history = reactive(new MessageHistory(cursor => {
  const params = new URLSearchParams({ limit: "50" });
  if (cursor !== null) params.set("before_message_id", cursor);
  return api(`/api/v1/messages/${encodeURIComponent(String(route.params.sessionId))}/load?` + params);
}));
const messages = computed(() => history.items);
async function refreshHistory() {
  evidenceVisible.value = false; evidence.value = null;
  await history.refresh(); await bottom();
}
async function loadOlder() {
  const area = scrollArea.value;
  const previousHeight = area?.scrollHeight || 0, previousTop = area?.scrollTop || 0;
  await history.older(); await nextTick();
  if (area) area.scrollTop = previousTop + area.scrollHeight - previousHeight;
}
let controller: AbortController | null = null;
function stopReceiving() {
  controller?.abort();
}
async function bottom() {
  await nextTick();
  scrollArea.value?.scrollTo({
    top: scrollArea.value.scrollHeight,
    behavior: "smooth",
  });
}
onMounted(async () => {
  try {
    const data = await api(`/api/v1/sessions/${route.params.sessionId}?include_messages=false`);
    if (data.session.app_id !== route.params.appId)
      throw new Error("会话与应用不匹配");
    application.value = data.application;
    await refreshHistory();
  } catch (e) {
    error.value = formatAPIError(e);
  }
});
onBeforeRouteLeave(() => {
  controller?.abort();
  return true;
});
function keydown(e: KeyboardEvent) {
  if (e.key === "Enter" && !e.shiftKey && !e.isComposing) {
    e.preventDefault();
    send();
  }
}
async function send() {
  if (running.value || !draft.value.trim()) return;
  const query = draft.value;
  draft.value = "";
  error.value = "";
  running.value = true;
  controller = new AbortController();
  try {
    await sendTurn(
      String(route.params.sessionId),
      query,
      (type, data) => {
        if (type === "answer") history.items.push(data);
      },
      controller.signal,
    );
    await refreshHistory();
  } catch (e) {
    if ((e as Error).name === "AbortError") {
      await refreshHistory();
      error.value = "已停止接收并尝试刷新历史。停止接收不代表远端取消，请核对结果后再决定是否重新提问。";
    } else {
      error.value = formatAPIError(e);
      draft.value = query;
    }
  } finally {
    running.value = false;
  }
}
async function showEvidence(m: Message) {
  evidenceVisible.value = true;
  evidence.value = null;
  evidenceError.value = "";
  evidenceLoading.value = true;
  try {
    evidence.value = await api(
      prefix +
        `/evidence/${m.execution_id}?session_id=${route.params.sessionId}&message_id=${m.id}`,
    );
  } catch (e) {
    evidenceError.value = formatAPIError(e);
  } finally {
    evidenceLoading.value = false;
  }
}
async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    MessagePlugin.success("已复制");
  } catch {
    MessagePlugin.warning("当前环境无法访问剪贴板");
  }
}
</script>

<style scoped>
.history-actions { display: flex; gap: 8px; padding: 8px 20px; }
</style>
