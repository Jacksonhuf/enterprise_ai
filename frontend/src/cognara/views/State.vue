<template>
  <section class="center-state">
    <EmptyState
      :title="forbidden ? '当前无法访问此功能' : '未找到可访问的内容'"
      :description="
        forbidden
          ? '请由企业管理员核对角色与资源范围。'
          : '链接可能已变更，或当前范围内无可访问的对象。'
      "
      ><t-button theme="primary" @click="$router.push('/apps')"
        >返回应用</t-button
      ></EmptyState
    >
  </section>
</template>
<script setup lang="ts">
import { Button as TButton } from "tdesign-vue-next/es/button";
import { computed } from "vue";
import { useRoute } from "vue-router";
import EmptyState from "@/components/EmptyState.vue";
const route = useRoute(),
  forbidden = computed(() => route.path === "/forbidden");
</script>
