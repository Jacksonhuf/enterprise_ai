<template>
  <div class="login-page">
    <section class="login-card">
      <Brand />
      <h1>登录 Cognara</h1>
      <p class="muted">企业认知平台 · 本地验证环境</p>
      <t-alert
        theme="info"
        message="当前使用验证账户；未连接企业SSO，不接收真实密码。"
      /><label class="field-label">选择验证身份</label
      ><t-select
        v-model="identity"
        :options="[
          { label: '张敏 · 普通业务用户', value: 'demo-user' },
          { label: '应用开发者', value: 'demo-engineer' },
          { label: '企业与平台管理员', value: 'demo-admin' },
          { label: '隔离验证账户 · 另一企业', value: 'demo-outsider' },
        ]"
      /><t-alert v-if="error" theme="error" :message="error" /><t-button
        block
        theme="primary"
        :loading="loading"
        @click="login"
        >进入验证工作台</t-button
      >
      <p class="meta">运行服务只监听本机。请勿在此输入生产凭据。</p>
    </section>
  </div>
</template>
<script setup lang="ts">
import { Alert as TAlert } from "tdesign-vue-next/es/alert";
import { Button as TButton } from "tdesign-vue-next/es/button";
import { Select as TSelect } from "tdesign-vue-next/es/select";
import { ref } from "vue";
import { useRouter } from "vue-router";
import Brand from "../components/Brand.vue";
import { useCognaraAuth } from "../stores/auth";
const auth = useCognaraAuth(),
  router = useRouter(),
  identity = ref("demo-user"),
  loading = ref(false),
  error = ref("");
async function login() {
  loading.value = true;
  error.value = "";
  try {
    await auth.login(identity.value);
    router.push("/apps");
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}
</script>
