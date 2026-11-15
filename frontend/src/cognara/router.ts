import { createRouter, createWebHistory } from "vue-router";
import { useCognaraAuth } from "./stores/auth";
export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/login",
      component: () => import("./views/Login.vue"),
      meta: { public: true },
    },
    { path: "/", redirect: "/apps" },
    { path: "/apps", component: () => import("./views/Applications.vue") },
    {
      path: "/apps/:appId/chat/:sessionId",
      component: () => import("./views/Chat.vue"),
    },
    {
      path: "/build/apps",
      component: () => import("./views/Applications.vue"),
      meta: { build: true },
    },
    {
      path: "/build/apps/:appId/edit",
      component: () => import("./views/Editor.vue"),
      meta: { build: true },
    },
    {
      path: "/build/apps/:appId",
      component: () => import("./views/ApplicationDetail.vue"),
      meta: { build: true },
    },
    {
      path: "/governance/access",
      component: () => import("./views/Governance.vue"),
      meta: { admin: true },
    },
    {
      path: "/governance/audit",
      component: () => import("./views/Governance.vue"),
      meta: { admin: true },
    },
    {
      path: "/governance/operations",
      component: () => import("./views/Governance.vue"),
      meta: { admin: true },
    },
    {
      path: "/settings/:section",
      component: () => import("./views/Settings.vue"),
    },
    {
      path: "/assets",
      component: () => import("./views/References.vue"),
      meta: { build: true },
    },
    {
      path: "/services",
      component: () => import("./views/References.vue"),
      meta: { build: true },
    },
    { path: "/forbidden", component: () => import("./views/State.vue") },
    { path: "/:pathMatch(.*)*", component: () => import("./views/State.vue") },
  ],
});
router.beforeEach(async (to) => {
  const auth = useCognaraAuth();
  if (to.meta.public) return true;
  if (!auth.user) {
    try {
      await auth.load();
    } catch {
      return "/login";
    }
  }
  if ((to.meta.admin && !auth.isAdmin) || (to.meta.build && !auth.canBuild))
    return "/forbidden";
  if (
    to.path.startsWith("/settings/") &&
    !["profile", "general"].includes(String(to.params.section)) &&
    !auth.isAdmin
  )
    return "/forbidden";
  return true;
});
