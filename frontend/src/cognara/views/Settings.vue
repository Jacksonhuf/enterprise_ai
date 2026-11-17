<template>
  <section>
    <header class="page-header">
      <div>
        <h1>{{ title }}</h1>
        <p class="muted">企业级连接、能力与个人偏好</p>
      </div>
      <div v-if="editable" class="header-actions"><t-button :disabled="form.loading || form.saving" @click="reload">重新加载</t-button><t-button theme="primary" :loading="form.saving" :disabled="!form.canSave" @click="save">保存</t-button></div>
    </header>
    <t-alert v-if="section === 'general' && auth.preferencesError" theme="warning" :message="'界面偏好应用失败：' + auth.preferencesError"><template #operation><t-button variant="text" @click="auth.refreshPreferences()">重试应用</t-button></template></t-alert>
    <div class="settings-layout">
      <nav class="settings-nav">
        <router-link
          v-for="(label, key) in visibleSections"
          :key="key"
          :to="'/settings/' + key"
          >{{ label }}</router-link
        >
      </nav>
      <div class="panel">
        <t-loading v-if="initialLoading" size="small" text="正在读取配置" />
        <t-alert v-if="error || form.error" theme="error" :message="error || form.error" /><template
          v-if="editable"
          ><t-loading :loading="form.loading"><t-form label-align="top" :disabled="!form.loaded || form.loading || form.saving"
            ><t-form-item v-for="f in fields" :key="f.key" :label="f.label"
              ><t-select v-if="f.key === 'language'" v-model="form.values[f.key]" :options="[{label:'简体中文',value:'zh-CN'},{label:'英语',value:'en-US'}]" clearable />
              <t-select v-else-if="f.key === 'density'" v-model="form.values[f.key]" :options="[{label:'标准',value:'comfortable'},{label:'紧凑',value:'compact'}]" clearable />
              <t-textarea v-else-if="f.key === 'notice'" v-model="form.values[f.key]" :maxlength="512" :placeholder="f.placeholder" />
              <t-input v-else v-model="form.values[f.key]" :maxlength="f.key === 'display_name' ? 80 : 512" :placeholder="f.placeholder" /></t-form-item></t-form></t-loading
          ><t-alert v-if="form.conflict" class="spaced" theme="warning" message="服务器配置已变更。当前输入已保留，请重新加载后核对再保存。" /><t-alert
            theme="info"
             :message="section === 'general' ? '界面密度和时区保存后生效；清空时区使用浏览器时区。语言目前仅保存偏好。' : section === 'public' ? '企业公告保存后可在平台运行与运营的公共配置中查看；尚未提供面向所有用户的公告分发。' : '仅保存本地验证环境的非敏感偏好。'" /></template
        ><template v-else-if="['models', 'model-catalog'].includes(section)"
          ><h2>模型配置</h2>
          <p class="muted">
            模型由服务端配置，密钥通过环境变量引用，不返回浏览器。
          </p>
          <div v-for="m in models" :key="m.id" class="reference-row">
            <div>
              <strong>{{ m.name }}</strong>
              <p class="meta">{{ modelStatus(m) }}</p>
            </div>
            <t-button
              v-if="m.id !== 'validation-only'"
              :disabled="!m.credential_configured"
              :loading="testing === m.id"
              @click="testModel(m.id)"
              >测试连接</t-button
            >
          </div>
          <t-alert
            class="spaced"
            theme="info"
            message="测试成功表示模型响应有效；应用仍须选择该模型并重新发布。" /></template
        ><template v-else
          ><h2>{{ title }}</h2>
          <p class="muted">沿用已有组件组织配置，真实提供方尚未接入。</p>
          <div class="connection-empty">
            <span class="badge warning">尚未连接</span>
            <h3>等待接入真实服务</h3>
            <p>当前不保存密钥、不测试虚假连接、不启用未经验证的能力。</p>
          </div>
          <t-alert
            theme="warning"
            message="MCP、网页搜索、文档解析与Python技能沙箱当前未启用。"
        /></template>
      </div>
    </div>
  </section>
</template>
<script setup lang="ts">
import { Alert as TAlert } from "tdesign-vue-next/es/alert";
import { Button as TButton } from "tdesign-vue-next/es/button";
import { Form as TForm } from "tdesign-vue-next/es/form";
import { FormItem as TFormItem } from "tdesign-vue-next/es/form";
import { Input as TInput } from "tdesign-vue-next/es/input";
import { Loading as TLoading } from "tdesign-vue-next/es/loading";
import { Select as TSelect } from "tdesign-vue-next/es/select";
import { Textarea as TTextarea } from "tdesign-vue-next/es/textarea";
import { ref, computed, reactive, onMounted } from "vue";
import { useRoute, onBeforeRouteLeave, onBeforeRouteUpdate } from "vue-router";
import { MessagePlugin } from "tdesign-vue-next/es/message";
import { api, put, post, prefix, formatAPIError } from "../api";
import { SettingsDraft } from "../settingsDraft";

import { useCognaraAuth } from "../stores/auth";
const models = ref<any[]>([]),
  testing = ref("");
const route = useRoute(),
  auth = useCognaraAuth(),
  error = ref(""),
  initialLoading = ref(false);
const sections: Record<string, string> = {
  models: "模型配置",
  "model-catalog": "模型目录",
  mcp: "MCP工具",
  skills: "技能管理",
  websearch: "网页搜索",
  credentials: "凭据管理",
  storage: "对象存储",
  vectorstore: "向量存储",
  queues: "任务队列",
  system: "系统设置",
  tenant: "企业设置",
  "api-keys": "API密钥",
  public: "公共配置",
  profile: "个人资料",
  general: "通用偏好",
};
const section = computed(() => String(route.params.section)),
  title = computed(() => sections[section.value] || "配置不存在"),
  editable = computed(() =>
    ["profile", "general", "public"].includes(section.value),
  ),
  visibleSections = computed(() =>
    auth.isAdmin ? sections : { profile: "个人资料", general: "通用偏好" },
  ),
  fields = computed(() =>
    section.value === "profile"
      ? [
          {
            key: "display_name",
            label: "显示名称",
            placeholder: "填写显示名称",
          },
        ]
      : section.value === "public"
        ? [{ key: "notice", label: "企业公告", placeholder: "填写公共提示" }]
        : [
            { key: "language", label: "语言", placeholder: "zh-CN" },
            { key: "timezone", label: "时区", placeholder: "Asia/Shanghai" },
            { key: "density", label: "界面密度", placeholder: "comfortable" },
          ],
  );
const form=reactive(new SettingsDraft(section.value,
  ()=>api(prefix+"/settings/"+section.value),
  payload=>put(prefix+"/settings/"+section.value,payload),
));
onMounted(async()=>{
  if(editable.value){await form.load();return;}
  initialLoading.value=true;
  try{
    if(["models","model-catalog"].includes(section.value))models.value=(await api(prefix+"/models")).items;
    await api(prefix+"/settings/"+section.value);
  }catch(e){error.value=formatAPIError(e)}
  finally{initialLoading.value=false}
});
function confirmNavigation(){return !editable.value||!form.dirty||window.confirm("有尚未保存的配置，确定离开吗？");}
onBeforeRouteLeave(confirmNavigation);
onBeforeRouteUpdate(confirmNavigation);
async function reload(){
  if(form.dirty&&!window.confirm("重新加载会替换当前未保存的输入，是否继续？"))return;
  error.value=""; await form.load();
}
async function save(){
  error.value="";
  if(await form.save()){
    MessagePlugin.success("配置已保存");
    if(section.value === "general")await auth.refreshPreferences();
    if(section.value === "profile")try{await auth.refreshProfile();}catch(e){error.value="配置已保存，账户信息刷新失败："+formatAPIError(e);}
  }
}
function modelStatus(m: any) {
  return (
    (
      {
        VALIDATION: "确定性验证",
        VERIFIED: "连接已验证",
        FAILED: "连接测试失败",
        NOT_TESTED: "尚未测试",
        MISSING_CREDENTIAL: "未设置服务端凭据",
      } as Record<string, string>
    )[m.status] || "未知状态"
  );
}
async function testModel(id: string) {
  testing.value = id;
  error.value = "";
  try {
    await post(prefix + "/models/" + id + "/test", {});
    models.value = (await api(prefix + "/models")).items;
    MessagePlugin.success("模型连接测试通过");
  } catch (e) {
    error.value = formatAPIError(e);
    try { models.value = (await api(prefix + "/models")).items; } catch { /* Keep the original connection error visible. */ }
  } finally {
    testing.value = "";
  }
}
</script>
