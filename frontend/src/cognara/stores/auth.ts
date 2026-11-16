import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { api, prefix, credential, formatAPIError } from "../api";
import type { Principal } from "../types";
export const useCognaraAuth = defineStore("cognara-auth", () => {
  const user = ref<Principal | null>(null);
  const modelsConfigured = ref(false);
  const density = ref<"comfortable" | "compact">("comfortable");
  const preferencesError = ref("");
  let preferenceRequest = 0;
  const canBuild = computed(() =>
    ["engineer", "admin"].includes(user.value?.role || ""),
  );
  const isAdmin = computed(() => user.value?.role === "admin");
  let generation = 0;
  function clearState() { user.value = null; modelsConfigured.value = false; density.value = "comfortable"; preferencesError.value = ""; preferenceRequest++; }
  function active(version: number, identity: string) {
    return generation === version && credential() === identity;
  }
  async function load() {
    const version = ++generation, identity = credential();
    clearState();
    const latest = await api<Principal>(prefix + "/me");
    if (!active(version, identity)) throw new Error("账户已变化，请重新登录");
    const runtime = await api(prefix + "/runtime-status");
    if (!active(version, identity)) throw new Error("账户已变化，请重新登录");
    // Publish both results together; partial initialization must not enable navigation.
    user.value = latest;
    modelsConfigured.value = runtime.models_configured === true;
    await refreshPreferences();
    if (!active(version, identity)) throw new Error("账户已变化，请重新登录");
  }
  async function refreshPreferences() {
    const current = user.value, version = generation, identity = credential(), request = ++preferenceRequest;
    if (!current) return;
    const isCurrent = () => active(version, identity) && request === preferenceRequest && user.value?.actor_user_id === current.actor_user_id && user.value?.tenant_id === current.tenant_id;
    try {
      const snapshot = await api(prefix + "/settings/general", { signal: AbortSignal.timeout(5000) });
      if (!isCurrent()) return;
      if (!snapshot.revision || !snapshot.values || typeof snapshot.values !== "object" || Array.isArray(snapshot.values))
        throw new Error("偏好响应无效，请重试");
      density.value = snapshot.values.density === "compact" ? "compact" : "comfortable";
      preferencesError.value = "";
    } catch (e) {
      if (isCurrent()) preferencesError.value = formatAPIError(e);
    }
  }
  async function refreshProfile() {
    const current = user.value, version = generation, identity = credential();
    if (!current) return;
    const latest = await api<Principal>(prefix + "/me");
    if (active(version, identity) && latest.actor_user_id === current.actor_user_id && latest.tenant_id === current.tenant_id && user.value?.actor_user_id === current.actor_user_id && user.value?.tenant_id === current.tenant_id)
      user.value = latest;
  }
  async function login(id: string) {
    localStorage.setItem("cognara_demo_identity", id);
    const pending = load(), version = generation;
    try { await pending; }
    catch (e) {
      // A failure from an earlier login must not log out the newer account.
      if (active(version, id)) logout();
      throw e;
    }
  }
  function logout() {
    generation++;
    localStorage.removeItem("cognara_demo_identity");
    clearState();
  }
  return { user, modelsConfigured, density, preferencesError, canBuild, isAdmin, load, refreshPreferences, refreshProfile, login, logout };
});
