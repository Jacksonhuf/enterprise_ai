<template>
  <section v-if="application">
    <header class="page-header">
      <div>
        <div class="breadcrumb">
          <router-link to="/build/apps">认知应用管理</router-link> / 应用配置
        </div>
        <h1>{{ config.basic.name || "新应用配置" }}</h1>
        <p class="muted">草稿与线上发布分别管理；一个主模型，显式选择能力。</p><t-tag v-if="dirty" theme="warning" variant="light">有未保存修改</t-tag>
      </div>
      <div class="header-actions">
        <t-button :disabled="state.busy" @click="reloadDraft">重新加载</t-button><t-button @click="save" :loading="saving" :disabled="!state.canSave">保存草稿</t-button
        ><t-button theme="primary" @click="checkPublish" :loading="checking" :disabled="!state.canCheck"
          >检查并发布</t-button
        >
      </div>
    </header>
    <t-alert v-if="error" theme="error" :message="error" /><t-alert
      v-if="saved"
      theme="success"
      message="草稿已保存，当前线上发布未改变。"
    /><t-tabs v-model="tab"
      ><t-tab-panel value="basic" label="基本信息" /><t-tab-panel
        value="model"
        label="模型与提示词" /><t-tab-panel
        value="refs"
        label="能力引用" /><t-tab-panel value="conversation" label="对话默认项"
    /></t-tabs>
    <t-alert v-if="state.conflict" theme="warning" message="当前草稿或发布已被修改。输入仍保留，请重新加载后核对。" />
    <t-alert v-if="tab === 'model' && modelError" theme="warning" :message="modelError" />
    <t-button v-if="tab === 'model' && modelError" variant="text" @click="loadModels">重新加载模型目录</t-button>
    <div class="form-panel">
      <t-form label-align="left" :label-width="170" :disabled="state.busy"
        ><template v-if="tab === 'basic'"
          ><t-form-item label="应用名称" required-mark
            ><t-input
              v-model="config.basic.name"
              :maxlength="80" /></t-form-item
          ><t-form-item label="业务描述"
            ><t-textarea
              v-model="config.basic.description"
              :autosize="{ minRows: 3, maxRows: 6 }" /></t-form-item></template
        ><template v-if="tab === 'model'"
          ><t-form-item label="主模型" required-mark
            ><t-select
              v-model="config.model_prompt.model_id"
              :options="availableModels" /></t-form-item
          ><t-form-item label="提示词" required-mark
            ><t-textarea
              v-model="config.model_prompt.prompt"
              :autosize="{ minRows: 10, maxRows: 18 }"
              :maxlength="16000" /></t-form-item
          ><t-alert
            theme="info"
            message="验证模式不调用模型；选择已配置模型后使用原有Agent引擎，业务数值仍由受控服务计算。" /></template
        ><template v-if="tab === 'refs'"
          ><t-form-item label="认知资产"
            ><div class="reference-box">
              <t-tag
                v-for="a in config.capability_refs.asset_refs"
                :key="a.asset_id"
                :closable="!state.busy"
                @close="
                  config.capability_refs.asset_refs =
                    config.capability_refs.asset_refs.filter(
                      (x) => x.asset_id !== a.asset_id,
                    )
                "
                >{{ assetName(a.asset_id) }}</t-tag><t-button
                variant="outline"
                :loading="pickerLoading" @click="openAssets"
                >选择资产</t-button
              >
            </div></t-form-item
          ><t-form-item label="认知服务"
            ><div class="reference-box">
              <t-tag
                v-for="s in config.capability_refs.service_refs"
                :key="serviceKey(s)"
                :closable="!state.busy"
                @close="config.capability_refs.service_refs = config.capability_refs.service_refs.filter(x => serviceKey(x) !== serviceKey(s))"
                >{{ services.find(x => x.service_id === s.service_id && x.service_version === s.version)?.name || s.service_id }} · {{ s.version }}</t-tag
              ><t-button variant="outline" :loading="pickerLoading" @click="openServices"
                >选择服务版本</t-button
              >
            </div></t-form-item
          ><t-form-item label="受控能力"
            ><t-checkbox-group
              v-model="config.capability_refs.tool_names"
              :options="[
                { label: '已授权服务调用', value: 'cognitive_call_service' },
                {
                  label: '资产上下文读取',
                  value: 'cognitive_get_asset_context',
                },
                { label: '资产发现', value: 'cognitive_discover_assets' },
                { label: '正式指标执行', value: 'cognitive_execute' },
              ]" /></t-form-item
          ><t-alert
            theme="info"
            message="引用不等于消费授权。应用使用与数据范围由企业管理员独立授权。" /></template
        ><template v-if="tab === 'conversation'"
          ><t-form-item label="欢迎语"
            ><t-textarea
              v-model="config.conversation_defaults.welcome" /></t-form-item
          ><t-form-item label="示例问题"
            ><div class="question-fields">
              <t-input
                v-for="(_, i) in config.conversation_defaults
                  .suggested_questions"
                :key="i"
                v-model="config.conversation_defaults.suggested_questions[i]"
              /><t-button
                variant="outline"
                @click="
                  config.conversation_defaults.suggested_questions.push('')
                "
                >增加示例问题</t-button
              >
            </div></t-form-item
          ></template
        ></t-form
      >
    </div>
    <t-drawer
      v-model:visible="pickerVisible"
      :header="pickerKind === 'assets' ? '选择认知资产' : '选择服务版本'"
      size="640px"
      :confirm-btn="{content:'确认选择',disabled:state.busy}"
      @confirm="confirmPicker"
      ><t-input v-model="pickerQuery" placeholder="搜索名称与用途" clearable />
      <div class="selector-items">
        <label v-for="x in pickerRows" :key="x.id" class="selector-item"
          ><t-checkbox
            :checked="selected.includes(x.id)"
            @change="
              (v: boolean) => togglePicker(x.id, v)
            "
          />
          <div>
            <strong>{{ x.name }}</strong>
            <p class="muted">{{ x.description }}</p>
          </div></label
        >
      </div>
      <t-alert
        theme="info"
        message="仅展示当前有权发现的资产或内部开放服务；运行时仍重新鉴权。" /></t-drawer
    ><t-drawer
      v-model:visible="publishVisible"
      header="发布检查与确认"
      size="760px"
      :confirm-btn="{
        content: '确认发布',
        disabled: !state.canPublishAt(clock),
        loading: publishing,
      }"
      @confirm="publish"
      ><t-alert
        :theme="state.checkCurrentAt(clock) ? 'success' : 'warning'"
        :message="
          check?.status === 'PASS'
            ? state.checkCurrentAt(clock) ? '检查通过，可确认发布' : '检查已过期或草稿有变化，请重新检查'
            : '发布检查存在阻塞项'
        " />
      <div class="detail-list">
        <div>
          <span>配置完整性</span><strong>名称、单主模型与提示词</strong>
        </div>
        <div>
          <span>能力引用</span
          ><strong
            >{{ config.capability_refs.asset_refs.length }}项资产 /
            {{ config.capability_refs.service_refs.length }}项服务</strong
          >
        </div>
        <div>
          <span>生效规则</span
          ><strong>产生不可变快照；下一轮使用当前发布</strong>
        </div>
      </div>
      <p v-for="issue in check?.issues" :key="issue" class="warning-text">
        {{ issue }}
      </p>
      <p v-if="check?.valid_until" class="meta">检查有效至 {{ formatDateTime(check.valid_until, preferences.timezone) }}</p>
      <section v-if="state.review" class="publish-review">
        <h2>本次发布变更</h2>
        <div class="detail-list">
          <div><span>对比基线</span><strong>{{ state.review.firstPublication ? "首次发布 · 尚无线上的发布版本" : state.review.currentReleaseId }}</strong></div>
          <div><span>检查草稿版本</span><strong>{{ state.review.revision }}</strong></div>
        </div>
        <p class="muted">以下是本次检查时的配置快照。草稿或当前发布变化后需重新检查；确认前线上配置保持不变。</p>
        <div v-if="state.review.rows.length" class="publish-review-scroll">
          <table class="data-table publish-review-table"><thead><tr><th>配置项</th><th>当前发布</th><th>本次发布</th></tr></thead>
            <tbody><tr v-for="row in state.review.rows" :key="row.label"><td>{{ row.label }}</td><td>{{ row.before || "（空）" }}</td><td>{{ row.after || "（空）" }}</td></tr></tbody>
          </table>
        </div>
        <t-alert v-else theme="info" message="配置与当前发布一致，确认仍会产生新发布版本。" />
      </section>
      <label class="field-label">发布说明</label
      ><t-textarea v-model="releaseNote" /><t-alert
        class="spaced"
        theme="info"
        message="发布不会自动授予使用权。用户和应用的资源权限仍须由企业管理员配置。"
    /></t-drawer>
  </section>
  <section v-else><t-alert v-if="error" theme="error" :message="error" /><t-button v-if="error" @click="reloadDraft">重新加载草稿</t-button><t-loading v-else :loading="true" text="正在读取应用配置" /></section>
</template>
<script setup lang="ts">
import { formatDateTime } from "../dateTime";
import { useCognaraAuth } from "../stores/auth";
const preferences = useCognaraAuth();
import { Alert as TAlert } from "tdesign-vue-next/es/alert";
import { Button as TButton } from "tdesign-vue-next/es/button";
import { Checkbox as TCheckbox } from "tdesign-vue-next/es/checkbox";
import { CheckboxGroup as TCheckboxGroup } from "tdesign-vue-next/es/checkbox";
import { Drawer as TDrawer } from "tdesign-vue-next/es/drawer";
import { Form as TForm } from "tdesign-vue-next/es/form";
import { FormItem as TFormItem } from "tdesign-vue-next/es/form";
import { Input as TInput } from "tdesign-vue-next/es/input";
import { Loading as TLoading } from "tdesign-vue-next/es/loading";
import { Select as TSelect } from "tdesign-vue-next/es/select";
import { TabPanel as TTabPanel } from "tdesign-vue-next/es/tabs";
import { Tabs as TTabs } from "tdesign-vue-next/es/tabs";
import { Tag as TTag } from "tdesign-vue-next/es/tag";
import { Textarea as TTextarea } from "tdesign-vue-next/es/textarea";
import { ref, computed, onMounted, onBeforeUnmount, reactive } from "vue";
import { useRoute, useRouter, onBeforeRouteLeave, onBeforeRouteUpdate } from "vue-router";
import { MessagePlugin } from "tdesign-vue-next/es/message";
import { ApplicationDraft } from "../applicationDraft";
import { loadAllOptions } from "../applicationOptions";
import { serviceKey, toggleServiceVersion, resolveServiceSelection } from "../capabilitySelection";
import { api, put, post, prefix } from "../api";
import type { Config, Asset, Service } from "../types";
const modelOptions = ref([
  { label: "确定性验证 · 不调用模型", value: "validation-only" },
]);
const route = useRoute(),
  router = useRouter(),
  tab = ref("basic"),
  publishVisible = ref(false),
  releaseNote = ref(""),
  pickerVisible = ref(false),
  pickerLoading = ref(false),
  pickerKind = ref("assets"),
  pickerQuery = ref(""),
  selected = ref<string[]>([]),
  assets = ref<Asset[]>([]),
  services = ref<Service[]>([]);
const editorRoute=route.fullPath, appId=String(route.params.appId), appPath=prefix+`/apps/${encodeURIComponent(appId)}`;
const state=reactive(new ApplicationDraft({read:()=>api(appPath),save:payload=>put(appPath,payload),check:payload=>post(appPath+"/release-checks",payload),publish:payload=>post(appPath+"/releases",payload)}));
const application=computed(()=>state.application), config=computed({get:()=>state.config,set:(value:Config)=>{state.config=value}}),
 error=computed({get:()=>state.error,set:(value:string)=>{state.error=value}}), dirty=computed(()=>state.dirty), saved=computed(()=>state.saved&&!state.dirty), saving=computed(()=>state.saving),checking=computed(()=>state.checking),publishing=computed(()=>state.publishing),check=computed(()=>state.checkResult);
const modelError=ref(""), clock=ref(Date.now());let clockTimer:ReturnType<typeof setInterval>|undefined;
const availableModels=computed(()=>modelOptions.value.some(model=>model.value===config.value.model_prompt.model_id)?modelOptions.value:[...modelOptions.value,{label:"已配置模型 · "+config.value.model_prompt.model_id,value:config.value.model_prompt.model_id,disabled:true}]);
const serviceReferences = computed(() => [
  ...config.value.capability_refs.service_refs,
  ...services.value.map(s => ({service_id:s.service_id,version:s.service_version})),
]);
const pickerRows = computed(() => {
  const rows = pickerKind.value === "assets"
    ? assets.value.map(a => ({id:a.asset_id,name:a.name,description:a.summary}))
    : services.value.map(s => ({id:serviceKey({service_id:s.service_id,version:s.service_version}),name:s.name,description:s.service_version + " · 内部应用渠道"}));
  const existing = pickerKind.value === "assets"
    ? config.value.capability_refs.asset_refs.map(ref => ({id:ref.asset_id,name:ref.asset_id,description:"已有引用 · 未在可发现目录返回；发布时仍需检查"}))
    : config.value.capability_refs.service_refs.map(ref => ({id:serviceKey(ref),name:ref.service_id,description:ref.version + " · 已有引用；发布时仍需检查"}));
  const known = new Set(rows.map(row=>row.id));
  return [...rows,...existing.filter(row=>!known.has(row.id))].filter(row => (row.name + row.description).toLowerCase().includes(pickerQuery.value.toLowerCase()));
});
function togglePicker(id:string, checked:boolean) {
  selected.value = pickerKind.value === "services"
    ? toggleServiceVersion(selected.value,id,checked,serviceReferences.value)
    : checked ? [...new Set([...selected.value,id])] : selected.value.filter(value=>value!==id);
}
function assetName(id: string) {
  return (
    assets.value.find((x) => x.asset_id === id)?.name ||
    {
      "inventory-policy": "渠道库存管理制度",
      "chip-gap-metric": "物料原始缺口",
    }[id] ||
    id
  );
}
async function loadModels(){
  modelError.value="";
  try{modelOptions.value=(await api(prefix+"/models")).items.map((m:any)=>({label:m.name,value:m.id,disabled:m.id!=="validation-only"&&!m.credential_configured}));}
  catch(e){modelError.value="模型目录读取失败："+(e as Error).message;}
}
onMounted(()=>{void state.load();void loadModels();clockTimer=setInterval(()=>{clock.value=Date.now()},1000)});
onBeforeUnmount(()=>{clearInterval(clockTimer);state.invalidate();});
function confirmNavigation(){return !dirty.value||window.confirm("有尚未保存的修改，确定离开吗？")}
onBeforeRouteLeave(confirmNavigation);onBeforeRouteUpdate(confirmNavigation);
async function reloadDraft(){if(dirty.value&&!window.confirm("重新加载将替换当前未保存草稿，是否继续？"))return;publishVisible.value=false;pickerVisible.value=false;await state.load();}
async function save(){await state.save();}
async function checkPublish(){if(await state.checkPublish())publishVisible.value=true;}
async function publish(){
  const released=await state.publish(releaseNote.value);if(!released||route.fullPath!==editorRoute)return;
  publishVisible.value=false;
  if(state.dirty){MessagePlugin.warning("发布已完成，当前仍有未保存修改");return;}
  MessagePlugin.success("应用已发布；使用授权独立管理");
  router.push(`/build/apps/${appId}`);
}
async function loadOptions<T>(kind:string, key:(value:T)=>string) {
  return loadAllOptions<T>(cursor => {
    const params = new URLSearchParams({limit:"100"});
    if (cursor !== null) params.set("cursor",cursor);
    return api(prefix + "/selectors/" + kind + "?" + params);
  }, key);
}
async function openAssets() {
  if (pickerLoading.value) return;
  pickerLoading.value=true; error.value="";
  try {
    assets.value = await loadOptions<Asset>("assets", asset=>asset.asset_id);
    pickerKind.value = "assets"; pickerQuery.value = "";
    selected.value = config.value.capability_refs.asset_refs.map(x=>x.asset_id);
    pickerVisible.value = true;
  } catch (e) { error.value=(e as Error).message; }
  finally { pickerLoading.value=false; }
}
async function openServices() {
  if (pickerLoading.value) return;
  pickerLoading.value=true; error.value="";
  try {
    services.value = await loadOptions<Service>("services", service=>serviceKey({service_id:service.service_id,version:service.service_version}));
    pickerKind.value = "services"; pickerQuery.value = "";
    selected.value = config.value.capability_refs.service_refs.map(serviceKey);
    pickerVisible.value = true;
  } catch (e) { error.value=(e as Error).message; }
  finally { pickerLoading.value=false; }
}
function confirmPicker() {
  if(state.busy)return;
  try {
    if (pickerKind.value === "assets") config.value.capability_refs.asset_refs=selected.value.map(asset_id=>({asset_id}));
    else config.value.capability_refs.service_refs=resolveServiceSelection(selected.value,serviceReferences.value);
    pickerVisible.value=false;
  } catch (e) { error.value=(e as Error).message; }
}
</script>

<style scoped>
.publish-review { margin: 20px 0; }
.publish-review-scroll { max-height: 360px; overflow: auto; }
.publish-review-table { table-layout: fixed; }
.publish-review-table th:first-child { width: 100px; }
.publish-review-table td { white-space: pre-wrap; overflow-wrap: anywhere; vertical-align: top; }
</style>
