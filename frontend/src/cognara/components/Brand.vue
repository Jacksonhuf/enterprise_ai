<template>
  <span class="cognara-brand" aria-label="Cognara 企业认知平台"
    ><img src="/brand/cognara-mark.svg" alt="" class="brand-symbol" /><img
      v-if="!compact"
      src="/brand/cognara-word.svg"
      alt="Cognara"
      class="brand-word"
  /></span>
</template>
<script setup lang="ts">
defineProps<{ compact?: boolean }>();
</script>
