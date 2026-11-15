<template>
  <section v-if="application">
    <header class="page-header">
      <div>
        <div class="breadcrumb">
          <router-link to="/build/apps">认知应用管理</router-link> / 应用配置
        </div>
        <h1>{{ config.basic.name || "新应用配置" }}</h1>
        <p class="muted">草稿与线上发布分别管理；一个主模型，显式选择能力。</p>
      </div>
      <div class="header-actions">
        <t-button @click="save" :loading="saving">保存草稿</t-button
        ><t-button theme="primary" @click="checkPublish" :loading="checking"
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
    <div class="form-panel">
      <t-form label-align="left" :label-width="170"
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
              :options="modelOptions" /></t-form-item
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
                closable
                @close="
                  config.capability_refs.asset_refs =
                    config.capability_refs.asset_refs.filter(
                      (x) => x.asset_id !== a.asset_id,
                    )
                "
                >{{ assetName(a.asset_id) }}</t-tag><t-button
                variant="outline"
                @click="openAssets"
                >选择资产</t-button
              >
            </div></t-form-item
          ><t-form-item label="认知服务"
            ><div class="reference-box">
              <t-tag
                v-for="s in config.capability_refs.service_refs"
                :key="s.service_id"
                closable
                @close="config.capability_refs.service_refs = []"
                >补货缺口分析 · {{ s.version }}</t-tag
              ><t-button variant="outline" @click="openServices"
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
      :confirm-btn="'确认选择'"
      @confirm="confirmPicker"
      ><t-input v-model="pickerQuery" placeholder="搜索名称与用途" clearable />
      <div class="selector-items">
        <label v-for="x in pickerRows" :key="x.id" class="selector-item"
          ><t-checkbox
            :checked="selected.includes(x.id)"
            @change="
              (v: boolean) =>
                (selected = v
                  ? [...selected, x.id]
                  : selected.filter((i) => i !== x.id))
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
      size="640px"
      :confirm-btn="{
        content: '确认发布',
        disabled: check?.status !== 'PASS',
        loading: publishing,
      }"
      @confirm="publish"
      ><t-alert
        :theme="check?.status === 'PASS' ? 'success' : 'warning'"
        :message="
          check?.status === 'PASS'
            ? '检查通过，可确认发布'
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
      <label class="field-label">发布说明</label
      ><t-textarea v-model="releaseNote" /><t-alert
        class="spaced"
        theme="info"
        message="发布不会自动授予使用权。用户和应用的资源权限仍须由企业管理员配置。"
    /></t-drawer>
  </section>
  <t-loading v-else :loading="true" text="正在读取应用配置" />
</template>
<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRoute, useRouter, onBeforeRouteLeave } from "vue-router";
import { MessagePlugin } from "tdesign-vue-next";
import { api, put, post, prefix } from "../api";
import type { Application, Config, Asset, Service } from "../types";
const modelOptions = ref([
  { label: "确定性验证 · 不调用模型", value: "validation-only" },
]);
const route = useRoute(),
  router = useRouter(),
  application = ref<Application | null>(null),
  config = ref<Config>({
    basic: { name: "", description: "" },
    model_prompt: { model_id: "validation-only", prompt: "" },
    capability_refs: { asset_refs: [], service_refs: [], tool_names: [] },
    conversation_defaults: { welcome: "", suggested_questions: [] },
  }),
  tab = ref("basic"),
  error = ref(""),
  saved = ref(false),
  saving = ref(false),
  checking = ref(false),
  publishing = ref(false),
  baseline = ref(""),
  publishVisible = ref(false),
  check = ref<any>(null),
  releaseNote = ref(""),
  pickerVisible = ref(false),
  pickerKind = ref("assets"),
  pickerQuery = ref(""),
  selected = ref<string[]>([]),
  assets = ref<Asset[]>([]),
  services = ref<Service[]>([]);
const dirty = computed(() => baseline.value !== JSON.stringify(config.value));
const pickerRows = computed(() =>
  (pickerKind.value === "assets"
    ? assets.value.map((a) => ({
        id: a.asset_id,
        name: a.name,
        description: a.summary,
      }))
    : services.value.map((s) => ({
        id: s.service_id,
        name: s.name,
        description: s.service_version + " · 内部应用渠道",
      }))
  ).filter((x) => x.name.includes(pickerQuery.value)),
);
function assetName(id: string) {
  return (
    assets.value.find((x) => x.asset_id === id)?.name ||
    {
      "inventory-policy": "渠道库存管理制度",
      "chip-gap-metric": "物料原始缺口",
    }[id] ||
    "已引用资产"
  );
}
onMounted(async () => {
  try {
    modelOptions.value = (await api(prefix + "/models")).items.map(
      (m: any) => ({
        label: m.name,
        value: m.id,
        disabled: m.id !== "validation-only" && !m.credential_configured,
      }),
    );
    application.value = await api<Application>(
      prefix + `/apps/${route.params.appId}`,
    );
    config.value = structuredClone(application.value.draft);
    baseline.value = JSON.stringify(config.value);
  } catch (e) {
    error.value = (e as Error).message;
  }
});
onBeforeRouteLeave(
  () => !dirty.value || window.confirm("有尚未保存的修改，确定离开吗？"),
);
async function save() {
  saving.value = true;
  error.value = "";
  try {
    application.value = await put<Application>(
      prefix + `/apps/${route.params.appId}`,
      {
        expected_draft_revision: application.value!.draft_revision,
        ...config.value,
      },
    );
    baseline.value = JSON.stringify(config.value);
    saved.value = true;
    return true;
  } catch (e) {
    error.value = (e as Error).message;
    return false;
  } finally {
    saving.value = false;
  }
}
async function checkPublish() {
  checking.value = true;
  try {
    if (dirty.value && !(await save())) return;
    check.value = await post(
      prefix + `/apps/${route.params.appId}/release-checks`,
      { expected_draft_revision: application.value!.draft_revision },
    );
    publishVisible.value = true;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    checking.value = false;
  }
}
async function publish() {
  publishing.value = true;
  try {
    await post(prefix + `/apps/${route.params.appId}/releases`, {
      check_id: check.value.check_id,
      expected_draft_revision: application.value!.draft_revision,
      expected_current_release_id: application.value!.current_release_id,
      release_note: releaseNote.value,
    });
    baseline.value = JSON.stringify(config.value);
    MessagePlugin.success("应用已发布；使用授权独立管理");
    router.push(`/build/apps/${route.params.appId}`);
  } catch (e) {
    error.value = (e as Error).message;
    publishVisible.value = false;
  } finally {
    publishing.value = false;
  }
}
async function openAssets() {
  try {
    assets.value = (await api(prefix + "/selectors/assets")).items;
    pickerKind.value = "assets";
    selected.value = config.value.capability_refs.asset_refs.map(
      (x) => x.asset_id,
    );
    pickerVisible.value = true;
  } catch (e) {
    error.value = (e as Error).message;
  }
}
async function openServices() {
  try {
    services.value = (await api(prefix + "/selectors/services")).items;
    pickerKind.value = "services";
    selected.value = config.value.capability_refs.service_refs.map(
      (x) => x.service_id,
    );
    pickerVisible.value = true;
  } catch (e) {
    error.value = (e as Error).message;
  }
}
function confirmPicker() {
  if (pickerKind.value === "assets")
    config.value.capability_refs.asset_refs = selected.value.map(
      (asset_id) => ({ asset_id }),
    );
  else
    config.value.capability_refs.service_refs = services.value
      .filter((x) => selected.value.includes(x.service_id))
      .map((x) => ({ service_id: x.service_id, version: x.service_version }));
  pickerVisible.value = false;
}
</script>
