<template>
  <section>
    <header class="page-header">
      <div>
        <h1>{{ pageTitle }}</h1>
        <p class="muted">{{ pageDescription }}</p>
      </div>
      <t-button
        v-if="section === 'grants'"
        theme="primary"
        @click="grantVisible = true"
        >新增授权</t-button
      ><t-button v-else @click="load">刷新</t-button>
    </header>
    <t-alert v-if="error" theme="error" :message="error" /><t-tabs
      v-if="isAccess"
      :value="section"
      @change="changeTab"
      ><t-tab-panel
        v-for="(label, key) in accessTabs"
        :key="key"
        :value="key"
        :label="label" /></t-tabs
    ><t-tabs v-if="isOperations" :value="section" @change="changeTab"
      ><t-tab-panel
        v-for="(label, key) in operationTabs"
        :key="key"
        :value="key"
        :label="label"
    /></t-tabs>
    <div class="list-toolbar">
      <t-input
        v-model="query"
        placeholder="搜索请求编号、用户标识、应用或资源"
        clearable
      /><span class="meta">{{
        data.sampled_at ? new Date(data.sampled_at).toLocaleString("zh-CN") : ""
      }}</span>
    </div>
    <div v-if="paged" class="list-toolbar governance-filters">
      <t-select v-if="section === 'audit'" v-model="outcomeFilter" :options="outcomeOptions" placeholder="审计结果" />
      <t-select v-if="section === 'exceptions' || section === 'grants'" v-model="statusFilter" :options="statusOptions" placeholder="处理状态" />
      <t-select v-if="section === 'usage'" v-model="unitFilter" :options="unitOptions" placeholder="用量单位" />
      <t-date-range-picker v-if="section !== 'grants'" v-model="timeRange" enable-time-picker clearable value-type="time-stamp" />
      <t-button variant="text" @click="clearFilters">清空筛选</t-button>
      <span class="meta">已加载 {{ collection.items.length }} 条 · 按接收顺序排列</span>
    </div>
    <t-alert v-if="paged && collection.error" theme="error" :message="collection.error" />
    <t-button v-if="paged && collection.error" variant="text" @click="load">重新加载列表</t-button>
    <t-loading :loading="loading"
      ><div v-if="section === 'status'" class="status-grid">
        <article
          v-for="(m, i) in data.modules || []"
          :key="m.module"
          class="status-card"
        >
          <div class="card-top">
            <h2>{{ moduleName(m.module) }}</h2>
            <span
              class="badge"
              :class="m.status === 'UP' ? 'success' : 'warning'"
              >{{ statusName(m.status) }}</span
            >
          </div>
          <p class="muted">{{ m.summary }}</p>
        </article>
        <t-alert
          class="full-width"
          theme="warning"
          message="当前是验证适配器状态；原文档解析服务未启用，不能据此判断生产服务可用。"
        />
      </div>
      <template v-else-if="filtered.length"
        ><table class="data-table">
          <thead>
            <tr>
              <th v-for="c in columns" :key="c.key">{{ c.label }}</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(row, i) in filtered"
              :key="row.id || row.grant_id || row.event_id || i"
            >
              <td v-for="c in columns" :key="c.key">
                <span
                  v-if="c.key === 'active'"
                  class="badge"
                  :class="row.active ? 'success' : ''"
                  >{{ row.active ? "有效" : "已撤销" }}</span><template
                  v-else
                  >{{ display(row[c.key]) }}</template
                >
              </td>
              <td class="table-actions">
                <t-button variant="text" @click="selected = row">详情</t-button
                ><t-button
                  v-if="section === 'grants' && row.active"
                  variant="text"
                  theme="danger"
                  @click="confirmRevoke(row)"
                  >撤销</t-button
                ><t-button
                  v-for="action in exceptionActions(row)"
                  :key="action.status"
                  variant="text"
                  @click="transitionException(row, action)"
                  >{{ action.label }}</t-button
                >
              </td>
            </tr>
          </tbody>
        </table></template
      ><EmptyState
        v-else
        :title="
          section === 'exceptions'
            ? '当前没有已记录异常'
            : section === 'usage'
              ? '当前尚无用量事件'
              : '没有匹配记录'
        "
        description="仅展示实际接收到的记录，不用固定统计填充空状态。" /></t-loading
    ><t-button v-if="paged && collection.nextCursor !== null" class="spaced" :loading="loading" @click="collection.next()">加载更多记录</t-button><t-alert
      v-if="section === 'grants'"
      class="spaced"
      theme="info"
      message="用户的应用使用权、应用的服务调用权和实际数据范围分别检查。管理角色不自动授予全部数据。"
    /><t-drawer
      :visible="!!selected"
      :header="section === 'audit' ? '审计事件详情' : '对象详情'"
      size="640px"
      :footer="false"
      @close="selected = null"
      ><div v-if="selected" class="detail-list">
        <div v-for="c in columns" :key="c.key">
          <span>{{ c.label }}</span><strong>{{
            display(selected[c.key])
          }}</strong>
        </div>
        <div v-if="selected.request_id">
          <span>请求编号</span><code>{{ selected.request_id }}</code>
        </div>
        <div v-if="selected.code">
          <span>错误类型</span><strong>{{ selected.code }}</strong>
        </div>
        <div v-if="selected.route">
          <span>接口</span><code>{{ selected.route }}</code>
        </div>
        <div v-if="selected.execution_id">
          <span>执行关联</span><code>{{ selected.execution_id }}</code>
        </div>
      </div>
      <t-alert
        theme="info"
        message="关键审计记录与业务执行证据分别管理，不能据此读取完整原始业务结果。" /></t-drawer
    ><t-drawer
      v-model:visible="grantVisible"
      header="新增资源授权"
      size="640px"
      :confirm-btn="{ content: '保存授权', loading: saving, disabled: appsLoading || !!appsError }"
      @confirm="saveGrant"
      ><t-alert v-if="appsError" theme="error" :message="appsError" />
      <t-button v-if="appsError" variant="text" @click="loadApps">重新加载应用选项</t-button>
      <t-loading :loading="appsLoading"><t-form label-align="top"
        ><t-form-item label="主体类型"
          ><t-select
            v-model="grant.subject_type"
            :options="[
              { label: '用户组', value: 'GROUP' },
              { label: '用户', value: 'USER' },
              { label: '应用身份', value: 'APPLICATION' },
            ]"
            @change="resetGrant" /></t-form-item
        ><t-form-item label="授权主体"
          ><t-select
            v-model="grant.subject_id"
            filterable
            :options="subjectOptions" /></t-form-item
        ><t-form-item label="资源类型"
          ><t-select
            v-model="grant.resource_type"
            :options="
              grant.subject_type === 'APPLICATION'
                ? [
                    { label: '认知服务', value: 'SERVICE' },
                    { label: '认知资产', value: 'ASSET' },
                  ]
                : [{ label: '应用', value: 'APPLICATION' }]
            "
            @change="resetResource" /></t-form-item
        ><t-form-item label="资源"
          ><t-select
            v-model="grant.resource_id"
            filterable
            :options="resourceOptions"
            @change="resetAction" /></t-form-item
        ><t-form-item label="允许操作"
          ><t-select
            v-model="grant.action"
            :options="grantActions" /></t-form-item
        ><t-form-item label="数据范围"
          ><t-select
            v-model="grant.scope"
            :options="[
              { label: '华东渠道 · 验证范围', value: 'east' },
            ]" /></t-form-item></t-form></t-loading
      ><t-alert
        theme="info"
        message="当前验证集只支持显式列出的主体、资源和操作；不接受任意范围表达式。"
    /></t-drawer>
  </section>
</template>
<script setup lang="ts">
import { Alert as TAlert } from "tdesign-vue-next/es/alert";
import { Button as TButton } from "tdesign-vue-next/es/button";
import { DateRangePicker as TDateRangePicker } from "tdesign-vue-next/es/date-picker";
import { Drawer as TDrawer } from "tdesign-vue-next/es/drawer";
import { Form as TForm } from "tdesign-vue-next/es/form";
import { FormItem as TFormItem } from "tdesign-vue-next/es/form";
import { Input as TInput } from "tdesign-vue-next/es/input";
import { Loading as TLoading } from "tdesign-vue-next/es/loading";
import { Select as TSelect } from "tdesign-vue-next/es/select";
import { TabPanel as TTabPanel } from "tdesign-vue-next/es/tabs";
import { Tabs as TTabs } from "tdesign-vue-next/es/tabs";
import { ref, computed, reactive, watch, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { DialogPlugin } from "tdesign-vue-next/es/dialog";
import { MessagePlugin } from "tdesign-vue-next/es/message";
import { api, post, prefix, formatAPIError } from "../api";
import { PagedCollection } from "../pagedCollection";
import { governanceParams, pagedGovernance } from "../governanceFilters";
import { loadApplicationOptions } from "../applicationOptions";
import EmptyState from "@/components/EmptyState.vue";
const route = useRoute(),
  router = useRouter(),
  staticLoading = ref(true),
  error = ref(""),
  data = ref<any>({ items: [] }),
  query = ref(""),
  statusFilter = ref(""), outcomeFilter = ref(""), unitFilter = ref(""),
  timeRange = ref<(string|number|Date)[]>([]),
  selected = ref<any>(null),
  grantVisible = ref(false),
  saving = ref(false),
  apps = ref<any[]>([]),
  appsLoading = ref(false),
  appsError = ref(""),
  grant = ref({
    subject_type: "GROUP",
    subject_id: "all-users",
    resource_type: "APPLICATION",
    resource_id: "channel-replenishment",
    action: "application.use",
    scope: "east",
  });
const accessTabs = {
    users: "企业用户",
    groups: "用户组",
    clients: "应用与Agent身份",
    roles: "角色与权限",
    grants: "资源授权",
  },
  operationTabs = {
    status: "运行状态",
    exceptions: "异常管理",
    usage: "基础用量",
    settings: "公共配置",
  };
const isAccess = computed(() => route.path.endsWith("/access")),
  isOperations = computed(() => route.path.endsWith("/operations")),
  section = computed(() =>
    route.path.endsWith("/audit")
      ? "audit"
      : String(route.query.tab || (isAccess.value ? "users" : "status")),
  );
const pageTitle = computed(() =>
    isAccess.value
      ? "身份与权限"
      : isOperations.value
        ? "平台运行与运营"
        : "审计与追溯",
  ),
  pageDescription = computed(() =>
    isAccess.value
      ? "维护身份关系、允许操作与资源数据范围。"
      : isOperations.value
        ? "查看真实采样状态、异常记录和基础用量。"
        : "按真实用户、应用身份和执行关联核对关键事件。",
  );
const columnMap: Record<string, { key: string; label: string }[]> = {
  users: [
    { key: "name", label: "用户" },
    { key: "source", label: "身份来源" },
    { key: "group", label: "成员范围" },
    { key: "status", label: "状态" },
  ],
  groups: [
    { key: "name", label: "用户组" },
    { key: "members", label: "成员数" },
    { key: "source", label: "身份来源" },
    { key: "status", label: "状态" },
  ],
  clients: [
    { key: "name", label: "主体名称" },
    { key: "type", label: "类型" },
    { key: "owner", label: "责任归属" },
    { key: "status", label: "状态" },
  ],
  roles: [
    { key: "name", label: "角色" },
    { key: "operations", label: "允许操作" },
    { key: "scope", label: "适用范围" },
  ],
  grants: [
    { key: "subject_id", label: "授权主体" },
    { key: "resource_id", label: "目标资源" },
    { key: "action", label: "允许操作" },
    { key: "scope", label: "数据范围" },
    { key: "active", label: "状态" },
  ],
  audit: [
    { key: "occurred_at", label: "发生时间" },
    { key: "actor_user_id", label: "真实用户" },
    { key: "action", label: "操作" },
    { key: "outcome", label: "结果" },
  ],
  exceptions: [
    { key: "name", label: "异常" },
    { key: "module", label: "功能" },
    { key: "request_id", label: "关联请求" },
    { key: "status", label: "状态" },
  ],
  usage: [
    { key: "capability", label: "能力" },
    { key: "count", label: "数量" },
    { key: "unit", label: "单位" },
    { key: "occurred_at", label: "记录时间" },
  ],
  settings: [
    { key: "name", label: "公共设置" },
    { key: "value", label: "当前值" },
  ],
};
const paged = computed(() => pagedGovernance(section.value));
const collection = reactive(new PagedCollection<any>(async (encoded, cursor) => {
  const filters = JSON.parse(encoded);
  const response = await api(prefix + "/governance/" + section.value + "?" + governanceParams(section.value, filters, cursor));
  return response;
}, row => row.event_id || row.grant_id || row.id));
const loading = computed(() => paged.value ? collection.loading : staticLoading.value);
const filterKey = computed(() => JSON.stringify({query:query.value,status:statusFilter.value,outcome:outcomeFilter.value,unit:unitFilter.value,range:timeRange.value}));
const outcomeOptions = [{label:"全部结果",value:""},...Object.entries({success:"成功",accepted:"已接受",denied:"拒绝",failed:"失败",partial:"部分完成",canceled:"已取消"}).map(([value,label])=>({value,label}))];
const unitOptions = [{label:"全部单位",value:""}, {label:"调用次数",value:"CALL"}, {label:"词元",value:"TOKEN"}, {label:"毫秒",value:"MILLISECOND"}];
const statusOptions = computed(() => [{label:"全部状态",value:""}, ...Object.entries(section.value === "grants" ? {ACTIVE:"有效",REVOKED:"已撤销"} : {OPEN:"待处理",ACKNOWLEDGED:"已确认",RESOLVED:"已解决"}).map(([value,label])=>({value,label}))]);
function clearFilters(){query.value="";statusFilter.value="";outcomeFilter.value="";unitFilter.value="";timeRange.value=[];}
watch(filterKey, () => { if (paged.value) void load(); });
watch(() => collection.error, value => {if(value)selected.value=null;});
const columns = computed(() => columnMap[section.value] || []),
  filtered = computed(() =>
    paged.value ? collection.items : (data.value.items || []).filter((x: any) =>
      JSON.stringify(x).includes(query.value),
    ),
  );
const labels: Record<string, string> = {
  "demo-user": "张敏",
  "demo-engineer": "应用开发者",
  "demo-admin": "管理员",
  "all-users": "业务用户组",
  "channel-replenishment": "渠道补货助手",
  "replenishment-gap": "补货缺口分析",
  "chip-gap-metric": "物料原始缺口",
  "inventory-policy": "渠道库存管理制度",
  east: "华东渠道",
  "application.use": "使用应用",
  "service.invoke": "调用服务",
  "execution.invoke": "执行正式指标",
  "asset.context.read": "读取资产上下文",
  OPEN: "待处理",
  ACKNOWLEDGED: "已确认",
  RESOLVED: "已解决",
  success: "成功",
  accepted: "已接受",
  denied: "拒绝",
  failed: "失败",
  partial: "部分完成",
  canceled: "已取消",
  CALL: "次",
  TOKEN: "词元",
  MILLISECOND: "毫秒",
};
function display(v: any) {
  if (v == null) return "—";
  return labels[String(v)] || String(v);
}
function actionLabel(v: string) {
  return labels[v] || v;
}
function statusName(v: string) {
  return (
    (
      {
        UP: "可用",
        DOWN: "不可用",
        UNKNOWN: "未知",
        DEGRADED: "降级",
      } as Record<string, string>
    )[v] || v
  );
}
function moduleName(v: string) {
  return (
    (
      {
        A: "认知资产",
        B: "业务语义",
        C: "数据资源",
        D: "认知执行",
        E: "认知服务",
      } as Record<string, string>
    )[v] || v
  );
}
function changeTab(v: any) {
  router.push({ path: route.path, query: { tab: String(v) } });
}
onMounted(async () => {
  await load();
  if (isAccess.value) {
    await loadApps();
  }
});
async function loadApps() {
  if (appsLoading.value) return;
  appsLoading.value = true; appsError.value = ""; apps.value = [];
  try {
    apps.value = await loadApplicationOptions(cursor => {
      const params = new URLSearchParams({ purpose: "MANAGE", limit: "100" });
      if (cursor !== null) params.set("cursor", cursor);
      return api(prefix + "/apps?" + params);
    });
  } catch (e) { appsError.value = formatAPIError(e); }
  finally { appsLoading.value = false; }
}
async function load() {
  selected.value = null;
  error.value = "";
  if (paged.value) { await collection.reset(filterKey.value); return; }
  staticLoading.value = true;
  try { data.value = await api(prefix + "/governance/" + section.value); }
  catch (e) { error.value = formatAPIError(e); }
  finally { staticLoading.value = false; }
}
const subjectOptions = computed(() =>
  grant.value.subject_type === "APPLICATION"
    ? apps.value.map((a) => ({ label: a.draft.basic.name, value: a.app_id }))
    : grant.value.subject_type === "USER"
      ? [
          { label: "张敏", value: "demo-user" },
          { label: "应用开发者", value: "demo-engineer" },
          { label: "管理员", value: "demo-admin" },
        ]
      : [{ label: "业务用户组", value: "all-users" }],
);
const resourceOptions = computed(() =>
  grant.value.resource_type === "APPLICATION"
    ? apps.value.map((a) => ({ label: a.draft.basic.name, value: a.app_id }))
    : grant.value.resource_type === "SERVICE"
      ? [{ label: "补货缺口分析", value: "replenishment-gap" }]
      : [
          { label: "物料原始缺口", value: "chip-gap-metric" },
          { label: "渠道库存管理制度", value: "inventory-policy" },
        ],
);
function exceptionActions(row: any) {
  if (section.value !== "exceptions") return [];
  return row.status === "OPEN"
    ? [
        { label: "确认", status: "ACKNOWLEDGED" },
        { label: "标记解决", status: "RESOLVED" },
      ]
    : row.status === "ACKNOWLEDGED"
      ? [{ label: "标记解决", status: "RESOLVED" }]
      : row.status === "RESOLVED"
        ? [{ label: "重新打开", status: "OPEN" }]
        : [];
}
function transitionException(
  row: any,
  action: { label: string; status: string },
) {
  const dialog = DialogPlugin.confirm({
    header: action.label + "异常",
    body: "更新异常处理状态；业务执行需另行核对，不会自动重试。",
    confirmBtn: action.label,
    onConfirm: async () => {
      try {
        const updated = await post(
          prefix + "/governance/exceptions/" + row.id + "/status",
          { expected_revision: row.revision, status: action.status },
        );
        if (selected.value?.id === row.id) selected.value = updated;
        await load();
        MessagePlugin.success("异常状态已更新");
        dialog.hide();
      } catch (e) {
        error.value = formatAPIError(e);
        dialog.hide();
      }
    },
  });
}
function resetGrant() {
  grant.value.subject_id = subjectOptions.value[0]?.value || "";
  grant.value.resource_type =
    grant.value.subject_type === "APPLICATION" ? "SERVICE" : "APPLICATION";
  resetResource();
}
function resetResource() {
  grant.value.resource_id = resourceOptions.value[0]?.value || "";
  resetAction();
}
const grantActions = computed(() =>
  grant.value.resource_type === "ASSET" &&
  grant.value.resource_id === "chip-gap-metric"
    ? [
        { label: "读取资产上下文", value: "asset.context.read" },
        { label: "执行正式指标", value: "execution.invoke" },
      ]
    : [{ label: actionLabel(grant.value.action), value: grant.value.action }],
);
function resetAction() {
  grant.value.action =
    grant.value.resource_type === "APPLICATION"
      ? "application.use"
      : grant.value.resource_type === "SERVICE"
        ? "service.invoke"
        : grant.value.resource_id === "chip-gap-metric"
          ? "execution.invoke"
          : "asset.context.read";
}
async function saveGrant() {
  if (saving.value || appsLoading.value || appsError.value) return;
  error.value = "";
  saving.value = true;
  try {
    await post(prefix + "/governance/grants", grant.value);
    grantVisible.value = false;
    await load();
    MessagePlugin.success("授权已生效；相同有效授权不会重复创建");
  } catch (e) {
    error.value = formatAPIError(e);
  } finally {
    saving.value = false;
  }
}
function confirmRevoke(row: any) {
  const dialog = DialogPlugin.confirm({
    header: "撤销授权",
    body: `将撤销${display(row.subject_id)}对${display(row.resource_id)}的本条授权。若仍有其他有效授权，访问可能继续允许。`,
    confirmBtn: "确认撤销",
    onConfirm: async () => {
      try {
        await api(prefix + "/governance/grants/" + row.grant_id, {
          method: "DELETE",
        });
        dialog.hide();
        load();
      } catch (e) {
        error.value = formatAPIError(e);
        dialog.hide();
      }
    },
  });
}
</script>

<style scoped>
.governance-filters { flex-wrap: wrap; }
.governance-filters :deep(.t-select__wrap) { width: 180px; }
</style>
