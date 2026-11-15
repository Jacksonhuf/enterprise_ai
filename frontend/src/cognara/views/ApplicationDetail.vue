<template>
  <section v-if="application">
    <header class="page-header">
      <div>
        <div class="breadcrumb">
          <router-link to="/build/apps">认知应用管理</router-link> / 应用详情
        </div>
        <h1>{{ application.draft.basic.name }}</h1>
        <p class="muted">{{ application.draft.basic.description }}</p>
      </div>
      <router-link :to="`/build/apps/${application.app_id}/edit`"
        ><t-button theme="primary">编辑草稿</t-button></router-link
      >
    </header>
    <t-tabs v-model="tab"
      ><t-tab-panel value="overview" label="概览" /><t-tab-panel
        value="releases"
        label="发布历史"
    /></t-tabs>
    <div v-if="tab === 'overview'" class="form-panel">
      <div class="detail-list">
        <div>
          <span>当前状态</span
          ><strong>{{
            application.current_release_id ? "已发布" : "草稿"
          }}</strong>
        </div>
        <div><span>主模型</span><strong>验证运行 · 未启用模型解释</strong></div>
        <div>
          <span>能力引用</span
          ><strong
            >{{ application.draft.capability_refs.asset_refs.length }}项资产 ·
            {{
              application.draft.capability_refs.service_refs.length
            }}项服务</strong
          >
        </div>
        <div>
          <span>对话欢迎语</span
          ><strong>{{
            application.draft.conversation_defaults.welcome || "未配置"
          }}</strong>
        </div>
      </div>
      <t-alert
        theme="info"
        message="草稿、当前发布、使用授权分别管理。修改草稿不会自动改变线上配置。"
      />
    </div>
    <table v-else class="data-table">
      <thead>
        <tr>
          <th>发布版本</th>
          <th>发布时间</th>
          <th>发布说明</th>
          <th>状态</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(r, i) in releases" :key="r.release_id">
          <td>发布记录 {{ releases.length - i }}</td>
          <td>{{ new Date(r.published_at).toLocaleString("zh-CN") }}</td>
          <td>{{ r.release_note || "无说明" }}</td>
          <td>
            <span
              class="badge"
              :class="
                r.release_id === application.current_release_id ? 'success' : ''
              "
              >{{
                r.release_id === application.current_release_id
                  ? "当前"
                  : "历史"
              }}</span
            >
          </td>
          <td>
            <t-button variant="text" @click="selected = r">配置摘要</t-button>
          </td>
        </tr>
      </tbody>
    </table>
    <t-drawer
      :visible="!!selected"
      header="发布版配置摘要"
      size="640px"
      :footer="false"
      @close="selected = null"
      ><template v-if="selected"
        ><div class="detail-list">
          <div>
            <span>应用名称</span
            ><strong>{{ selected.config_snapshot.basic.name }}</strong>
          </div>
          <div>
            <span>发布时间</span
            ><strong>{{
              new Date(selected.published_at).toLocaleString("zh-CN")
            }}</strong>
          </div>
          <div>
            <span>提示词</span>
            <p>{{ selected.config_snapshot.model_prompt.prompt }}</p>
          </div>
          <div>
            <span>资产与服务</span
            ><strong
              >{{
                selected.config_snapshot.capability_refs.asset_refs.length
              }}项资产 /
              {{
                selected.config_snapshot.capability_refs.service_refs.length
              }}项服务</strong
            >
          </div>
        </div>
        <t-alert
          theme="info"
          message="历史发布只读，不提供未确认的回滚或撤回操作。" /></template
    ></t-drawer>
  </section>
  <t-alert v-else-if="error" theme="error" :message="error" />
</template>
<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRoute } from "vue-router";
import { api, prefix } from "../api";
import type { Application, Release } from "../types";
const route = useRoute(),
  application = ref<Application | null>(null),
  releases = ref<Release[]>([]),
  selected = ref<Release | null>(null),
  tab = ref(route.query.tab === "releases" ? "releases" : "overview"),
  error = ref("");
onMounted(async () => {
  try {
    application.value = await api(prefix + `/apps/${route.params.appId}`);
    releases.value = (
      await api(prefix + `/apps/${route.params.appId}/releases`)
    ).items;
  } catch (e) {
    error.value = (e as Error).message;
  }
});
</script>
