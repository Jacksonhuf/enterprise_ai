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
      <t-button v-if="manage" theme="primary" :disabled="creation.blocked" @click="creation.open()"
        >新建应用</t-button
      >
    </header>
    <t-alert v-if="creation.blocked && !creation.visible" theme="error" :message="creation.error" /><t-button v-if="creation.blocked && !creation.visible" @click="reviewCreatedApps">刷新应用列表核对</t-button>
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
                <td><time :datetime="item.updated_at">{{ date(item.updated_at) }}</time></td>
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
        ><t-button v-if="manage" theme="primary" :disabled="creation.blocked" @click="creation.open()"
          >新建应用</t-button
        ></EmptyState
      ></t-loading
    ><div v-if="collection.nextCursor !== null" class="spaced">
      <t-button :loading="loading" @click="collection.next()">加载更多应用</t-button>
    </div><t-drawer
      :visible="creation.visible" header="创建认知应用" size="760px" :footer="false"
      :close-btn="!creation.creating" :close-on-overlay-click="false" :close-on-esc-keydown="!creation.creating" @close="creation.close()">
      <div class="creation-form">
        <t-radio-group v-model="creation.mode" variant="default-filled" :disabled="creation.creating || creation.blocked">
          <t-radio-button value="BLANK">空白创建</t-radio-button><t-radio-button value="TEMPLATE">从现有应用复制</t-radio-button>
        </t-radio-group>
        <t-form label-align="top" class="spaced" :disabled="creation.creating || creation.blocked">
          <t-form-item label="应用名称" required-mark><t-input v-model="creation.name" :maxlength="80" placeholder="如：渠道补货助手" /></t-form-item>
          <t-form-item label="业务描述"><t-textarea v-model="creation.description" placeholder="说明要解决的业务问题" /></t-form-item>
          <t-form-item v-if="creation.mode === 'TEMPLATE'" label="模板来源" required-mark>
            <t-select :value="creation.templateId" filterable clearable :loading="creation.templatesLoading"
              :options="creation.options.map(x=>({label:x.draft.basic.name,value:x.app_id}))"
              @change="value=>creation.selectTemplate(String(value || ''))" />
          </t-form-item>
        </t-form>
        <template v-if="creation.mode === 'TEMPLATE'">
          <t-button variant="text" :disabled="creation.creating || creation.blocked" :loading="creation.templatesLoading" @click="creation.loadTemplates()">刷新模板目录</t-button>
          <t-button v-if="creation.templateId" variant="text" :disabled="creation.creating || creation.blocked" :loading="creation.previewLoading" @click="creation.selectTemplate(creation.templateId)">刷新模板预览</t-button>
          <t-loading :loading="creation.previewLoading">
            <section v-if="creation.template" class="creation-preview">
              <h2>复制内容预览</h2>
              <div class="detail-list"><div><span>模板应用</span><strong>{{ creation.template.draft.basic.name }}</strong></div><div><span>模板草稿版本</span><strong>{{ creation.template.draft_revision }}</strong></div></div>
              <p class="muted">新应用名称和描述采用上方输入；其余配置按此版本复制。模板更新后需刷新预览。</p>
              <div class="creation-preview-scroll"><table class="data-table"><thead><tr><th>配置项</th><th>复制内容</th></tr></thead>
                <tbody><tr v-for="row in previewRows" :key="row.label"><td>{{ row.label }}</td><td>{{ row.value || "（空）" }}</td></tr></tbody>
              </table></div>
            </section>
          </t-loading>
        </template>
        <t-alert v-if="creation.error" class="spaced" theme="error" :message="creation.error" />
        <t-alert class="spaced" theme="info" message="复制只生成独立草稿，不继承发布、会话或授权；后续仍需检查、发布并授权。" />
        <div class="spaced">
          <t-button v-if="creation.blocked" :loading="reviewingCreation" @click="reviewCreatedApps">刷新应用列表核对</t-button>
          <t-button v-else theme="primary" :loading="creation.creating" :disabled="!creation.canCreate" @click="create">创建并配置</t-button>
          <t-button variant="text" :disabled="creation.creating" @click="creation.close()">取消</t-button>
        </div>
      </div>
    </t-drawer>
  </section>
</template>
<script setup lang="ts">
import { formatDateTime } from "../dateTime";
import { useCognaraAuth } from "../stores/auth";
const preferences = useCognaraAuth();
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
import { ref, computed, reactive, watch, onBeforeUnmount, nextTick } from "vue";
import {ApplicationCreation} from "../applicationCreation";
import {loadApplicationOptions} from "../applicationOptions";
import {configEntries} from "../configReview";
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

  query = ref("");
const reviewingCreation=ref(false);
const creation=reactive(new ApplicationCreation({
 list:()=>loadApplicationOptions(cursor=>{
  const params=new URLSearchParams({purpose:"MANAGE",limit:"100"});if(cursor!==null)params.set("cursor",cursor);
  return api(prefix+"/apps?"+params);
 }),
 read:id=>api(prefix+"/apps/"+encodeURIComponent(id)),
 create:body=>api(prefix+"/apps",{method:"POST",body:JSON.stringify(body),signal:AbortSignal.timeout(15000)})
}));
const previewRows=computed(()=>creation.template?configEntries(creation.template.draft).filter(row=>!["应用名称","业务描述"].includes(row.label)):[]);
watch(()=>creation.mode,mode=>{void creation.selectTemplate("");if(mode==="TEMPLATE")void creation.loadTemplates()});
onBeforeUnmount(()=>creation.invalidate());
const collection = reactive(new PagedCollection<Application>(async (query, cursor) => {
  const params = new URLSearchParams({ purpose: manage.value ? "MANAGE" : "USE", query });
  if (cursor !== null) params.set("cursor", cursor);
  return api(prefix + "/apps?" + params);
}, item => item.app_id));
const items = computed(() => collection.items), loading = computed(() => collection.loading);
watch([query, manage], () => {
  if(!reviewingCreation.value)void collection.reset(query.value);
}, { immediate: true });
function retry() {
  if (collection.items.length && collection.nextCursor !== null) void collection.next();
  else void collection.reset(query.value);
}
function date(v: string) { return formatDateTime(v, preferences.timezone); }
async function enter(a: Application) {
  try {
    const s = await post(prefix + `/apps/${a.app_id}/sessions`, {});
    router.push(`/apps/${a.app_id}/chat/${s.session_id}`);
  } catch (e) {
    error.value = (e as Error).message;
  }
}
async function create() {
 const result=await creation.create();if(result)await router.push(`/build/apps/${result.app_id}/edit`);
}
async function reviewCreatedApps(){
 if(reviewingCreation.value)return;
 reviewingCreation.value=true;
 try{
  query.value=creation.name.trim();await nextTick();await collection.reset(query.value);
  if(!collection.error&&!collection.loading){creation.blocked=false;creation.close()}
 }finally{reviewingCreation.value=false}
}
</script>

<style scoped>
.creation-preview { margin-top: 16px; }
.creation-preview-scroll { max-height: 300px; overflow: auto; }
.creation-preview td { white-space: pre-wrap; overflow-wrap: anywhere; vertical-align: top; }
.creation-preview th:first-child { width: 110px; }
.resource-search-toolbar { display:flex; align-items:center; flex-wrap:wrap; max-width:100%; }
</style>
