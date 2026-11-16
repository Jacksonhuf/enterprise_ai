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
    <t-alert v-if="error || collection.error" theme="error" :message="error || collection.error" />
    <t-button v-if="collection.error" variant="text" @click="retry">重试加载</t-button><div class="resource-search-toolbar"><ResourceSearchInput v-model="query" placeholder="搜索" /></div><t-loading :loading="loading"
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
              <tr v-for="item in items" :key="item.app_id">
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
          v-for="item in items"
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
         :title="query ? '未找到匹配应用' : '当前没有可用应用'"
        :description="query ? '请尝试更换或清空搜索条件。' : '应用需要先发布并获得使用授权。'"
        ><t-button v-if="manage" theme="primary" @click="createVisible = true"
          >新建应用</t-button
        ></EmptyState
      ></t-loading
    ><div v-if="collection.nextCursor !== null" class="spaced">
      <t-button :loading="loading" @click="collection.next()">加载更多应用</t-button>
    </div><t-drawer
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
import { Alert as TAlert } from "tdesign-vue-next/es/alert";
import { Button as TButton } from "tdesign-vue-next/es/button";
import { Drawer as TDrawer } from "tdesign-vue-next/es/drawer";
import { Form as TForm } from "tdesign-vue-next/es/form";
import { FormItem as TFormItem } from "tdesign-vue-next/es/form";
import { Input as TInput } from "tdesign-vue-next/es/input";
import { Loading as TLoading } from "tdesign-vue-next/es/loading";
import { RadioButton as TRadioButton } from "tdesign-vue-next/es/radio";
import { RadioGroup as TRadioGroup } from "tdesign-vue-next/es/radio";
import { Select as TSelect } from "tdesign-vue-next/es/select";
import { Textarea as TTextarea } from "tdesign-vue-next/es/textarea";
import { ref, computed, reactive, watch } from "vue";
import { PagedCollection } from "../pagedCollection";
import { useRoute, useRouter } from "vue-router";
import { api, post, prefix } from "../api";
import type { Application } from "../types";
import EmptyState from "@/components/EmptyState.vue";
import ResourceSearchInput from "@/components/ResourceSearchInput.vue";
import ResourceCardAvatar from "@/components/ResourceCardAvatar.vue";
const route = useRoute(),
  router = useRouter(),
  manage = computed(() => route.path.startsWith("/build")),

  error = ref(""),

  query = ref(""),
  createVisible = ref(false),
  creating = ref(false),
  creationMode = ref("BLANK"),
  name = ref(""),
  description = ref(""),
  templateId = ref("channel-replenishment");
const collection = reactive(new PagedCollection<Application>(async (query, cursor) => {
  const params = new URLSearchParams({ purpose: manage.value ? "MANAGE" : "USE", query });
  if (cursor !== null) params.set("cursor", cursor);
  return api(prefix + "/apps?" + params);
}, item => item.app_id));
const items = computed(() => collection.items), loading = computed(() => collection.loading);
watch([query, manage], () => {
  void collection.reset(query.value);
}, { immediate: true });
function retry() {
  if (collection.items.length && collection.nextCursor !== null) void collection.next();
  else void collection.reset(query.value);
}
function date(v: string) { return new Date(v).toLocaleString("zh-CN"); }
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

<style scoped>
.resource-search-toolbar { display:flex; align-items:center; flex-wrap:wrap; max-width:100%; }
</style>
