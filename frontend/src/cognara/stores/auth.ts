import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { api, prefix } from "../api";
import type { Principal } from "../types";
export const useCognaraAuth = defineStore("cognara-auth", () => {
  const user = ref<Principal | null>(null);
  const canBuild = computed(() =>
    ["engineer", "admin"].includes(user.value?.role || ""),
  );
  const isAdmin = computed(() => user.value?.role === "admin");
  async function load() {
    user.value = await api<Principal>(prefix + "/me");
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
  return { user, canBuild, isAdmin, load, login, logout };
});
