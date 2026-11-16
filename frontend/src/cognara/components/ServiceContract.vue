<template>
  <div class="service-contract">
    <p class="muted">必填以所在父对象存在为前提。表格仅辅助阅读，完整约束以原始定义为准。</p>
    <t-alert v-for="notice in view.notices" :key="notice" class="spaced" theme="warning" :message="notice" />
    <div v-if="view.rows.length" class="contract-table-wrap">
      <table class="data-table contract-table"><thead><tr><th>字段 / 路径</th><th>类型</th><th>父对象内必填</th><th>约束与说明</th></tr></thead>
        <tbody><tr v-for="row in view.rows" :key="row.path">
          <td><strong :style="{paddingLeft: row.depth * 10 + 'px'}">{{ row.name }}</strong><code>{{ row.path }}</code></td>
          <td>{{ row.type }}</td><td>{{ row.required }}</td>
          <td><p v-if="row.description">{{ row.description }}</p><span class="contract-text">{{ row.constraints }}</span></td>
        </tr></tbody>
      </table>
    </div>
    <details v-if="view.raw" class="contract-source">
      <summary>原始定义{{ view.rawTruncated ? "（已截断）" : "" }}</summary>
      <pre>{{ view.raw }}</pre>
    </details>
  </div>
</template>
<script setup lang="ts">
import {computed} from "vue";
import {Alert as TAlert} from "tdesign-vue-next/es/alert";
import {inspectContract} from "../serviceContract";
const props=defineProps<{schema:unknown}>();
const view=computed(()=>inspectContract(props.schema));
</script>
<style scoped>
.contract-table-wrap { overflow-x: auto; margin-top: 16px; }
.contract-table { min-width: 550px; table-layout: fixed; }
.contract-table th:nth-child(1) { width: 27%; }
.contract-table th:nth-child(2) { width: 17%; }
.contract-table th:nth-child(3) { width: 16%; }
.contract-table td { overflow-wrap: anywhere; vertical-align: top; }
.contract-table code { display: block; margin-top: 6px; color: var(--td-text-color-secondary); font-size: 12px; }
.contract-table p { margin: 0 0 8px; }
.contract-text { white-space: pre-wrap; }
.contract-source { margin-top: 20px; border: 1px solid var(--td-component-border); border-radius: var(--td-radius-medium); padding: 12px 16px; }
.contract-source summary { cursor: pointer; color: var(--td-text-color-primary); }
.contract-source pre { white-space: pre-wrap; overflow-wrap: anywhere; max-height: 320px; overflow: auto; font-size: 12px; color: var(--td-text-color-secondary); }
</style>
