import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { api, prefix } from "../api";
import type { Principal } from "../types";
export const useCognaraAuth = defineStore("cognara-auth", () => {
  const user = ref<Principal | null>(null);
  const modelsConfigured = ref(false);
  const canBuild = computed(() =>
    ["engineer", "admin"].includes(user.value?.role || ""),
  );
  const isAdmin = computed(() => user.value?.role === "admin");
  async function load() {
    user.value = await api<Principal>(prefix + "/me");
    modelsConfigured.value = (
      await api(prefix + "/runtime-status")
    ).models_configured;
  }
  async function login(id: string) {
    localStorage.setItem("cognara_demo_identity", id);
    try {
      await load();
    } catch (e) {
      logout();
      throw e;
    }
  }
  function logout() {
    localStorage.removeItem("cognara_demo_identity");
    user.value = null;
  }
  return { user, modelsConfigured, canBuild, isAdmin, load, login, logout };
});
