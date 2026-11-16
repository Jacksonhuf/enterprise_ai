<template>
  <router-view v-if="route.meta.public" />
  <div v-else class="app-shell" :data-density="auth.density">
    <aside class="sidebar" :class="{ collapsed }">
      <div class="sidebar-brand">
        <Brand :compact="collapsed" /><button
          class="icon-button collapse"
          aria-label="折叠或展开侧栏"
          @click="collapsed = !collapsed"
        >
          {{ collapsed ? "›" : "‹" }}
        </button>
      </div>
      <template v-if="!collapsed"
        ><t-input v-if="!isChat" v-model="navQuery" placeholder="搜索功能" clearable />
        <div class="nav-scroll">
          <template v-if="isChat"
            ><router-link class="nav-link" to="/apps">‹ 返回应用</router-link>
            <div class="nav-group-label">当前应用</div>
            <button class="nav-link" :disabled="creatingSession" @click="newChat">＋ 新建对话</button>
            <div class="nav-group-label">历史对话</div>
            <t-input v-model="historyQuery" placeholder="搜索对话标题" clearable @enter="loadSessions" />
            <button class="nav-link" :disabled="sessionPages.loading" @click="loadSessions">搜索 / 刷新历史</button>
            <p v-if="sessionPages.error || sessionError" class="meta">{{ sessionPages.error || sessionError }}</p>
            <div v-for="s in sessionPages.items" :key="s.session_id" class="session-history-row">
              <router-link class="nav-link" :to="`/apps/${s.app_id}/chat/${s.session_id}`">{{ s.title }}</router-link>
              <button class="icon-button" :aria-label="`重命名：${s.title}`" @click="renameDraft.open(s)">✎</button>
            </div>
            <button v-if="sessionPages.nextCursor" class="nav-link" :disabled="sessionPages.loading" @click="sessionPages.next()">加载更多对话</button>
            <p v-if="!sessionPages.loading && !sessionPages.items.length && !sessionPages.error" class="meta">暂无匹配的对话</p>
            </template
          ><template v-else
            ><div
              v-for="group in visibleGroups"
              :key="group.label"
              class="nav-group"
            >
              <div class="nav-group-label">{{ group.label }}</div>
              <router-link
                v-for="item in group.items"
                :key="item.path"
                class="nav-link"
                :to="item.path"
                >{{ item.label }}</router-link
              >
            </div></template
          >
        </div></template
      >
      <nav v-else class="compact-nav">
        <router-link to="/apps" title="应用">应</router-link
        ><router-link v-if="auth.canBuild" to="/build/apps" title="应用管理"
          >构</router-link
        ><router-link
          v-if="auth.isAdmin"
          to="/governance/operations"
          title="运行与运营"
          >管</router-link
        >
      </nav>
      <div class="account-area">
        <button
          class="account-button"
          @click="accountOpen = !accountOpen"
          aria-label="账户菜单"
          :aria-expanded="accountOpen"
        >
          <span class="avatar">{{
            auth.user?.name?.slice(0, 1) || "人"
          }}</span><span v-if="!collapsed"
            ><strong>{{ auth.user?.name }}</strong
            ><small>演示企业 · {{ roleLabel }}</small></span
          >
        </button>
        <div v-if="accountOpen" class="account-popover">
          <strong>{{ auth.user?.name }}</strong>
          <p class="muted">演示企业 · 验证环境</p>
          <router-link @click="accountOpen = false" to="/settings/profile"
            >个人资料</router-link
          ><router-link @click="accountOpen = false" to="/settings/general"
            >通用偏好</router-link
          ><button @click="logout">退出当前账户</button>
        </div>
      </div>
    </aside>
    <section class="workspace">
      <div class="validation-banner">
        <span class="status-dot"></span>验证环境 · 合成数据 · 2026-04-02历史时点
        <span class="banner-side"
          >业务数据为验证集；{{
            auth.modelsConfigured ? "模型按应用发布配置运行" : "模型未配置"
          }}</span
        >
      </div>
      <main class="page-canvas"><router-view :key="route.fullPath" /></main>
    </section>
    <t-dialog v-model:visible="renameDraft.visible" header="重命名对话" :confirm-btn="{ content: '保存', loading: renameDraft.saving, disabled: renameDraft.conflict }" :close-on-overlay-click="false" @confirm="saveRename">
      <t-input v-model="renameDraft.title" :disabled="renameDraft.saving" :maxlength="80" placeholder="输入对话标题" />
      <p v-if="renameDraft.conflict" class="meta">关闭窗口，刷新历史后重新打开以获取最新标题。</p>
      <p v-if="renameDraft.error" class="meta">{{ renameDraft.error }}</p>
    </t-dialog>
  </div>
</template>
<script setup lang="ts">
import { Dialog as TDialog } from "tdesign-vue-next/es/dialog";
import { Input as TInput } from "tdesign-vue-next/es/input";
import { computed, ref, watch, reactive } from "vue";
import { useRoute, useRouter } from "vue-router";
import Brand from "./components/Brand.vue";
import { useCognaraAuth } from "./stores/auth";
import { api, post, put, prefix, formatAPIError } from "./api";
import { SessionRename } from "./sessionRename";
import { PagedCollection } from "./pagedCollection";
import { sessionHistoryVersion } from "./sessionHistory";
const auth = useCognaraAuth(),
  route = useRoute(),
  router = useRouter();
const collapsed = ref(localStorage.getItem("cognara_collapsed") === "true"),
  accountOpen = ref(false),
  navQuery = ref(""),
  sessionError = ref("");
const historyQuery = ref("");
const sessionPages = reactive(new PagedCollection<any>((query, cursor) => {
  const params = new URLSearchParams({ app_id: String(route.params.appId), query, limit: "20" });
  if (cursor) params.set("cursor", cursor);
  return isChat.value ? api("/api/v1/sessions?" + params) : Promise.resolve({ items: [] });
}, item => item.session_id));
async function loadSessions() { sessionError.value = ""; await sessionPages.reset(historyQuery.value); }
const renameDraft = reactive(new SessionRename((id, payload) => put(`/api/v1/sessions/${encodeURIComponent(id)}`, payload)));
async function saveRename() { if (await renameDraft.save()) await loadSessions(); }
watch(sessionHistoryVersion, () => { if (isChat.value) void loadSessions(); });
watch(collapsed, (v) => localStorage.setItem("cognara_collapsed", String(v)));
const isChat = computed(() => route.path.includes("/chat/"));
const roleLabel = computed(
  () =>
    ({ user: "业务用户", engineer: "应用开发者", admin: "管理员" })[
      auth.user?.role || "user"
    ],
);
const visibleGroups = computed(() => {
  const groups = [{ label: "应用", items: [{ label: "应用", path: "/apps" }] }];
  if (auth.canBuild)
    groups.push(
      {
        label: "认知资产",
        items: [{ label: "认知资产目录", path: "/assets" }],
      },
      {
        label: "构建",
        items: [
          { label: "认知服务", path: "/services" },
          { label: "认知应用管理", path: "/build/apps" },
        ],
      },
    );
  if (auth.isAdmin)
    groups.push({
      label: "治理",
      items: [
        { label: "身份与权限", path: "/governance/access" },
        { label: "审计与追溯", path: "/governance/audit" },
        { label: "平台运行与运营", path: "/governance/operations" },
        { label: "平台配置", path: "/settings/models" },
        { label: "企业管理", path: "/settings/tenant" },
      ],
    });
  return groups
    .map((g) => ({
      ...g,
      items: g.items.filter((i) => i.label.includes(navQuery.value)),
    }))
    .filter((g) => g.items.length);
});
watch(
  () => route.fullPath,
  async () => {
    accountOpen.value = false; renameDraft.close(); historyQuery.value = "";
    await loadSessions();
  },
  { immediate: true },
);
const creatingSession = ref(false);
async function newChat() {
  if (creatingSession.value) return;
  creatingSession.value = true;
  try {
    const s = await post(prefix + `/apps/${route.params.appId}/sessions`, {});
    router.push(`/apps/${route.params.appId}/chat/${s.session_id}`);
  } catch (e) { sessionError.value = formatAPIError(e); }
  finally { creatingSession.value = false; }
}
function logout() {
  auth.logout();
  router.push("/login");
}
</script>

<style scoped>
.session-history-row { display:flex; align-items:center; gap:4px; }
.session-history-row .nav-link { flex:1; min-width:0; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
</style>
