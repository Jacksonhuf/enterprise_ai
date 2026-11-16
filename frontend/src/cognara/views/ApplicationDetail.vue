<template>
  <section v-if="application">
    <header class="page-header">
      <div>
        <div class="breadcrumb"><router-link to="/build/apps">认知应用管理</router-link> / 应用详情</div>
        <h1>{{ application.draft.basic.name }}</h1>
        <p class="muted">{{ application.draft.basic.description }}</p>
      </div>
      <router-link :to="`/build/apps/${application.app_id}/edit`"><t-button theme="primary">编辑草稿</t-button></router-link>
    </header>
    <t-tabs v-model="tab"><t-tab-panel value="overview" label="概览" /><t-tab-panel value="releases" label="发布历史" /></t-tabs>
    <div v-if="tab === 'overview'" class="form-panel">
      <div class="detail-list">
        <div><span>当前状态</span><strong>{{ application.current_release_id ? "已发布" : "草稿" }}</strong></div>
        <div><span>草稿主模型</span><strong>{{ modelLabel(application.draft.model_prompt.model_id, models) }}</strong></div>
        <div><span>当前发布主模型</span><strong>{{ current ? modelLabel(current.config_snapshot.model_prompt.model_id, models) : "尚未发布" }}</strong></div>
        <div><span>草稿能力引用</span><strong>{{ application.draft.capability_refs.asset_refs.length }} 项资产 · {{ application.draft.capability_refs.service_refs.length }} 项服务</strong></div>
        <div><span>草稿欢迎语</span><strong>{{ application.draft.conversation_defaults.welcome || "未配置" }}</strong></div>
      </div>
      <t-alert theme="info" message="草稿、当前发布、使用授权分别管理。修改草稿不会自动改变线上配置。" />
      <t-button v-if="current" class="spaced" variant="outline" @click="selected = current">查看草稿与当前发布差异</t-button>
    </div>
    <template v-else>
      <t-alert v-if="history.error" theme="error" :message="history.error" />
      <t-button v-if="history.error" variant="text" @click="retryHistory">重试加载</t-button>
      <t-loading :loading="history.loading">
        <table v-if="history.items.length" class="data-table">
          <thead><tr><th>发布版本</th><th>发布时间</th><th>发布说明</th><th>状态</th><th>操作</th></tr></thead>
          <tbody><tr v-for="r in history.items" :key="r.release_id">
            <td>{{ r.release_id }}</td><td>{{ date(r.published_at) }}</td><td>{{ r.release_note || "无说明" }}</td>
            <td><span class="badge" :class="r.release_id === application.current_release_id ? 'success' : ''">{{ r.release_id === application.current_release_id ? "当前" : "历史" }}</span></td>
            <td><t-button variant="text" @click="selected = r">查看配置与差异</t-button></td>
          </tr></tbody>
        </table>
        <p v-else-if="!history.loading && !history.error" class="muted">尚无发布记录。请先编辑草稿并完成发布检查。</p>
      </t-loading>
      <t-button v-if="history.nextCursor !== null" class="spaced" :loading="history.loading" @click="history.next()">加载更多发布记录</t-button>
    </template>
    <t-drawer :visible="!!selected" header="发布配置与差异" size="760px" :footer="false" @close="selected = null">
      <template v-if="selected">
        <div class="detail-list">
          <div><span>发布版本</span><strong>{{ selected.release_id }}</strong></div>
          <div><span>应用名称</span><strong>{{ selected.config_snapshot.basic.name }}</strong></div>
          <div><span>主模型</span><strong>{{ modelLabel(selected.config_snapshot.model_prompt.model_id, models) }}</strong></div>
          <div><span>发布时间</span><strong>{{ date(selected.published_at) }}</strong></div>
          <div><span>发布说明</span><p>{{ selected.release_note || "无说明" }}</p></div>
          <div><span>提示词</span><p class="review-text">{{ selected.config_snapshot.model_prompt.prompt }}</p></div>
        </div>
        <label class="field-label">比较目标</label>
        <t-radio-group v-model="compareTarget" variant="default-filled"><t-radio-button value="draft">当前草稿</t-radio-button><t-radio-button v-if="current && current.release_id !== selected.release_id" value="current">当前发布</t-radio-button></t-radio-group>
        <p class="muted">左侧为所选发布，右侧为{{ compareTarget === 'current' ? '当前发布' : '当前草稿' }}。仅显示变化项，不执行回滚或发布。</p>
        <table v-if="differences.length" class="data-table config-review"><thead><tr><th>配置项</th><th>所选发布</th><th>比较目标</th></tr></thead><tbody><tr v-for="row in differences" :key="row.label"><td>{{ row.label }}</td><td class="review-text">{{ row.before || "（空）" }}</td><td class="review-text">{{ row.after || "（空）" }}</td></tr></tbody></table>
        <t-alert v-else theme="success" message="两份配置没有差异。" />
        <t-alert class="spaced" theme="info" message="历史发布保持只读，草稿变更不会自动改变当前发布。" />
      </template>
    </t-drawer>
  </section>
  <section v-else><t-alert v-if="error" theme="error" :message="error" /><t-button v-if="error" @click="load">重新加载</t-button><t-loading v-else :loading="true" text="正在读取应用详情" /></section>
</template>
<script setup lang="ts">
import { formatDateTime } from "../dateTime";
import { useCognaraAuth } from "../stores/auth";
const preferences = useCognaraAuth();
import { Alert as TAlert } from "tdesign-vue-next/es/alert";
import { Button as TButton } from "tdesign-vue-next/es/button";
import { Drawer as TDrawer } from "tdesign-vue-next/es/drawer";
import { Loading as TLoading } from "tdesign-vue-next/es/loading";
import { RadioButton as TRadioButton } from "tdesign-vue-next/es/radio";
import { RadioGroup as TRadioGroup } from "tdesign-vue-next/es/radio";
import { TabPanel as TTabPanel } from "tdesign-vue-next/es/tabs";
import { Tabs as TTabs } from "tdesign-vue-next/es/tabs";
import { ref, reactive, computed, onMounted, watch } from "vue";
import { useRoute } from "vue-router";
import { api, prefix, formatAPIError } from "../api";
import { PagedCollection } from "../pagedCollection";
import { compareConfig, modelLabel } from "../configReview";
import type { Application, Release } from "../types";
const route = useRoute();
const application = ref<Application | null>(null), selected = ref<Release | null>(null), models = ref<{id:string;name:string}[]>([]);
const tab = ref(route.query.tab === "releases" ? "releases" : "overview"), error = ref(""), compareTarget = ref("draft");
const current = computed(() => application.value?.releases?.find(r => r.release_id === application.value?.current_release_id));
const differences = computed(() => selected.value && application.value ? compareConfig(selected.value.config_snapshot, compareTarget.value === "current" && current.value ? current.value.config_snapshot : application.value.draft) : []);
watch(selected, () => { compareTarget.value = "draft"; });
const history = reactive(new PagedCollection<Release>(async (_query, cursor) => {
  const params = new URLSearchParams(); if (cursor !== null) params.set("cursor", cursor);
  return api(prefix + `/apps/${route.params.appId}/releases?` + params);
}, item => item.release_id));
function retryHistory() { if (history.items.length && history.nextCursor !== null) void history.next(); else void history.reset(); }
function date(value:string) { return formatDateTime(value, preferences.timezone); }
async function load() {
  error.value = "";
  try { application.value = await api(prefix + `/apps/${route.params.appId}`); await history.reset(); }
  catch (e) { error.value = formatAPIError(e); }
}
onMounted(() => { void load(); void api<{items:{id:string;name:string}[]}>(prefix + "/models").then(data => { models.value = data.items; }).catch(() => {}); });
</script>
<style scoped>
.review-text { white-space: pre-wrap; overflow-wrap: anywhere; }
.config-review { table-layout: fixed; }
.config-review th:first-child { width: 100px; }
</style>
