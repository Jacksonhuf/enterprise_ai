<template>
  <section>
    <header class="page-header">
      <div>
        <h1>{{ title }}</h1>
        <p class="muted">企业级连接、能力与个人偏好</p>
      </div>
      <t-button v-if="editable" theme="primary" :loading="saving" @click="save"
        >保存</t-button
      >
    </header>
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
        <t-alert v-if="error" theme="error" :message="error" /><template
          v-if="editable"
          ><t-form label-align="top"
            ><t-form-item v-for="f in fields" :key="f.key" :label="f.label"
              ><t-input
                v-model="values[f.key]"
                :placeholder="f.placeholder" /></t-form-item></t-form
          ><t-alert
            theme="info"
            message="仅保存本地验证环境的非敏感偏好。" /></template
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
import { ref, computed, onMounted } from "vue";
import { useRoute } from "vue-router";
import { MessagePlugin } from "tdesign-vue-next";
import { api, put, post, prefix } from "../api";
import { useCognaraAuth } from "../stores/auth";
const models = ref<any[]>([]),
  testing = ref("");
const route = useRoute(),
  auth = useCognaraAuth(),
  error = ref(""),
  saving = ref(false),
  values = ref<Record<string, string>>({});
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
onMounted(async () => {
  try {
    if (["models", "model-catalog"].includes(section.value))
      models.value = (await api(prefix + "/models")).items;
    values.value = (await api(prefix + "/settings/" + section.value)).values;
  } catch (e) {
    error.value = (e as Error).message;
  }
});
async function save() {
  saving.value = true;
  try {
    await put(prefix + "/settings/" + section.value, { values: values.value });
    MessagePlugin.success("已保存");
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    saving.value = false;
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
    error.value = (e as Error).message;
    models.value = (await api(prefix + "/models")).items;
  } finally {
    testing.value = "";
  }
}
</script>
