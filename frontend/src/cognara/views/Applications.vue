<template>
  <section>
    <header class="page-header">
      <div>
        <h1>{{ manage ? "认知应用管理" : "应用" }}</h1>
        <p class="muted">
          {{
            manage
              ? "配置业务应用，分别管理草稿、发布与使用授权。"
              : "选择已发布且当前有权使用的应用。"
          }}
        </p>
      </div>
      <t-button v-if="manage" theme="primary" @click="createVisible = true"
        >新建应用</t-button
      >
    </header>
    <t-alert v-if="error" theme="error" :message="error" /><ResourceListToolbar
      v-model="scope"
      v-model:query="query"
      hide-scopes
    /><t-loading :loading="loading"
      ><div
        v-if="items.length"
        :class="manage ? 'application-table' : 'application-grid'"
      >
        <template v-if="manage"
          ><table class="data-table">
            <thead>
              <tr>
                <th>应用名称</th>
                <th>当前发布</th>
                <th>更新时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in filtered" :key="item.app_id">
                <td>
                  <strong>{{ item.draft.basic.name }}</strong>
                  <p class="meta">{{ item.draft.basic.description }}</p>
                </td>
                <td>
                  <span
                    class="badge"
                    :class="item.current_release_id ? 'success' : 'warning'"
                    >{{ item.current_release_id ? "已发布" : "草稿" }}</span
                  >
                </td>
                <td>{{ date(item.updated_at) }}</td>
                <td class="table-actions">
                  <router-link :to="`/build/apps/${item.app_id}`"
                    >详情</router-link
                  ><router-link :to="`/build/apps/${item.app_id}/edit`"
                    >配置</router-link
                  ><t-button variant="text" @click="enter(item)">进入</t-button>
                </td>
              </tr>
            </tbody>
          </table></template
        >
        <article
          v-else
          v-for="item in filtered"
          :key="item.app_id"
          class="application-card"
        >
          <div class="card-icon"><ResourceCardAvatar kind="agent" /></div>
          <span class="badge success">已发布</span>
          <h2>{{ item.draft.basic.name }}</h2>
          <p class="muted">{{ item.draft.basic.description }}</p>
          <div class="card-bottom">
            <small class="meta">营销供应链 · 验证场景</small
            ><t-button theme="primary" variant="outline" @click="enter(item)"
              >进入应用</t-button
            >
          </div>
        </article>
      </div>
      <EmptyState
        v-else
        title="当前没有可用应用"
        description="应用需要先发布并获得使用授权。"
        ><t-button v-if="manage" theme="primary" @click="createVisible = true"
          >新建应用</t-button
        ></EmptyState
      ></t-loading
    ><EmptyState
      v-if="items.length && !filtered.length"
      title="未找到匹配应用"
      description="请尝试更换或清空搜索条件。"
    /><t-drawer
      v-model:visible="createVisible"
      header="创建认知应用"
      size="600px"
      :confirm-btn="{ content: '创建并配置', loading: creating }"
      @confirm="create"
      ><t-radio-group v-model="creationMode" variant="default-filled"
        ><t-radio-button value="BLANK">空白创建</t-radio-button
        ><t-radio-button value="TEMPLATE"
          >预置模板</t-radio-button
        ></t-radio-group
      ><t-form label-align="top" class="spaced"
        ><t-form-item label="应用名称" required-mark
          ><t-input
            v-model="name"
            :maxlength="80"
            placeholder="如：渠道补货助手" /></t-form-item
        ><t-form-item label="业务描述"
          ><t-textarea
            v-model="description"
            placeholder="说明要解决的业务问题" /></t-form-item
        ><t-form-item v-if="creationMode === 'TEMPLATE'" label="模板来源"
          ><t-select
            v-model="templateId"
            :options="
              items.map((x) => ({ label: x.draft.basic.name, value: x.app_id }))
            " /></t-form-item></t-form
      ><t-alert
        theme="info"
        message="模板复制后独立维护；引用的资源仍需实际授权。保存草稿不会自动发布。"
    /></t-drawer>
  </section>
</template>
<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api, post, prefix } from "../api";
import type { Application } from "../types";
import EmptyState from "@/components/EmptyState.vue";
import ResourceListToolbar from "@/components/ResourceListToolbar.vue";
import ResourceCardAvatar from "@/components/ResourceCardAvatar.vue";
const route = useRoute(),
  router = useRouter(),
  manage = computed(() => route.path.startsWith("/build")),
  loading = ref(true),
  error = ref(""),
  items = ref<Application[]>([]),
  query = ref(""),
  scope = ref("all"),
  createVisible = ref(false),
  creating = ref(false),
  creationMode = ref("BLANK"),
  name = ref(""),
  description = ref(""),
  templateId = ref("channel-replenishment");
const filtered = computed(() =>
  items.value.filter((x) =>
    (x.draft.basic.name + x.draft.basic.description).includes(query.value),
  ),
);
function date(v: string) {
  return new Date(v).toLocaleString("zh-CN");
}
onMounted(load);
async function load() {
  try {
    items.value = (
      await api<{ items: Application[] }>(
        prefix + "/apps?purpose=" + (manage.value ? "MANAGE" : "USE"),
      )
    ).items;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}
async function enter(a: Application) {
  try {
    const s = await post(prefix + `/apps/${a.app_id}/sessions`, {});
    router.push(`/apps/${a.app_id}/chat/${s.session_id}`);
  } catch (e) {
    error.value = (e as Error).message;
  }
}
async function create() {
  if (!name.value.trim()) {
    error.value = "请输入应用名称";
    return;
  }
  creating.value = true;
  try {
    const a = await post<Application>(prefix + "/apps", {
      creation_mode: creationMode.value,
      name: name.value,
      description: description.value,
      ...(creationMode.value === "TEMPLATE"
        ? { template_id: templateId.value }
        : {}),
    });
    router.push(`/build/apps/${a.app_id}/edit`);
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    creating.value = false;
  }
}
</script>
