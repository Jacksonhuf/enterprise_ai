<template>
  <section>
    <header class="page-header">
      <div>
        <h1>{{ isServices ? "认知服务" : "认知资产目录" }}</h1>
        <p class="muted">
          在应用中引用已发布的业务能力；资产编制由业务服务负责。
        </p>
      </div>
    </header>
    <t-alert
      theme="info"
      message="当前目录来自供应链验证集，尚未连接真实业务服务。"
    />
    <div class="list-toolbar">
      <t-input v-model="query" placeholder="搜索名称" clearable /><t-button
        @click="load"
        >刷新</t-button
      >
    </div>
    <t-alert v-if="error" theme="error" :message="error" />
    <div class="card-grid">
      <article
        v-for="item in filtered"
        :key="item.asset_id || item.service_id"
        class="resource-card"
      >
        <div class="card-top">
          <h2>
            {{
              item.name || item.display_name || item.asset_id || item.service_id
            }}
          </h2>
          <span class="badge success">已发布</span>
        </div>
        <p class="muted">
          {{ item.description || "经过授权筛选的业务能力引用" }}
        </p>
        <div class="card-footer">
          <span class="meta"
            >{{ item.asset_type || "认知服务" }} ·
            {{ item.version || item.resolved_version || "当前发布版" }}</span
          ><t-button variant="text" @click="selected = item">详情</t-button>
        </div>
      </article>
    </div>
    <EmptyState
      v-if="!filtered.length"
      title="暂无可引用内容"
      description="请检查授权范围或搜索条件。"
    /><t-drawer
      :visible="!!selected"
      header="引用详情"
      :footer="false"
      @close="selected = null"
      ><div class="detail-list">
        <div v-for="(value, key) in selected" :key="key">
          <span>{{ key }}</span>
          <p>{{ typeof value === "object" ? JSON.stringify(value) : value }}</p>
        </div>
      </div></t-drawer
    >
  </section>
</template>
<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRoute } from "vue-router";
import { api, prefix } from "../api";
import EmptyState from "@/components/EmptyState.vue";
const route = useRoute(),
  isServices = computed(() => route.path === "/services"),
  items = ref<any[]>([]),
  query = ref(""),
  error = ref(""),
  selected = ref<any>(null);
const filtered = computed(() =>
  items.value.filter((i) => JSON.stringify(i).includes(query.value)),
);
async function load() {
  try {
    const d = await api(
      prefix + "/selectors/" + (isServices.value ? "services" : "assets"),
    );
    items.value = d.items || d.services || d.assets || [];
  } catch (e) {
    error.value = (e as Error).message;
  }
}
onMounted(load);
</script>
