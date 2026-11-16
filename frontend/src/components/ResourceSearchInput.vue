<template>
  <t-input :model-value="modelValue" class="resource-search-input" :placeholder="placeholder" :aria-label="placeholder" clearable
    @update:model-value="(value: unknown) => $emit('update:modelValue', String(value ?? ''))">
    <template #prefix-icon><SearchIcon size="16px" /></template>
  </t-input>
</template>
<script setup lang="ts">
import { Input as TInput } from 'tdesign-vue-next/es/input'
import { SearchIcon } from 'tdesign-icons-vue-next'
defineProps<{ modelValue: string; placeholder: string }>()
defineEmits<{ 'update:modelValue': [value: string] }>()
</script>
<style scoped>
.resource-search-input { width: 260px; max-width: 100%; margin-left: auto; }
.resource-search-input :deep(.t-input) { border-radius: var(--app-radius-sm); }
@media (max-width: 720px) { .resource-search-input { width: 100%; } }
</style>
