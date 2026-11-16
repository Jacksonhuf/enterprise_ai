<template>
  <section>
    <header class="page-header"><div><h1>{{ isServices ? "认知服务" : "认知资产目录" }}</h1><p class="muted">在应用中引用已发布的业务能力；资产编制由业务服务负责。</p></div></header>
    <t-alert theme="info" message="当前目录来自供应链验证集，尚未连接真实业务服务。" />
    <div class="list-toolbar"><t-input v-model="query" placeholder="搜索名称" clearable /><t-button :loading="collection.loading" @click="refresh">刷新</t-button></div>
    <t-alert v-if="collection.error" theme="error" :message="collection.error" />
    <t-button v-if="collection.error" variant="text" @click="retry">重试加载</t-button>
    <t-loading :loading="collection.loading">
      <div class="card-grid"><article v-for="item in collection.items" :key="key(item)" class="resource-card">
        <div class="card-top"><h2>{{ item.name }}</h2><span class="badge success">已发布</span></div>
        <p class="muted">{{ item.summary || item.description || "经过授权筛选的业务能力引用" }}</p>
        <div class="card-footer"><span class="meta">{{ kindName(item) }} · {{ version(item) }}</span><t-button variant="text" @click="selected = item">详情</t-button></div>
      </article></div>
      <EmptyState v-if="!collection.loading && !collection.items.length && !collection.error" title="暂无可引用内容" description="请检查授权范围或搜索条件。" />
    </t-loading>
    <t-button v-if="collection.nextCursor !== null" class="spaced" :loading="collection.loading" @click="collection.next()">加载更多</t-button>
    <t-drawer :visible="!!selected" header="引用详情" size="760px" :footer="false" @close="selected = null">
      <template v-if="selected"><div class="detail-list">
        <div><span>名称</span><strong>{{ selected.name }}</strong></div>
        <div><span>类型</span><strong>{{ kindName(selected) }}</strong></div>
        <div><span>版本</span><strong>{{ version(selected) }}</strong></div>
        <div><span>业务标识</span><strong>{{ selected.asset_id || selected.service_id }}</strong></div>
        <div><span>说明</span><p>{{ selected.summary || selected.description || "目录未提供业务说明" }}</p></div>
      </div>
      <template v-if="isServices">
        <div class="detail-list"><div><span>契约修订</span><strong>{{ selected.contract_revision || "目录未提供" }}</strong></div></div>
        <t-tabs v-model="contractTab"><t-tab-panel value="input" label="输入契约" /><t-tab-panel value="output" label="输出契约" /></t-tabs>
        <ServiceContract :key="contractTab" :schema="contractTab === 'input' ? selected.input_schema : selected.output_schema" />
        <t-alert class="spaced" theme="info" message="此页供契约阅读；实际调用仍需参数校验、发布引用及执行授权。" />
      </template>
      <t-alert class="spaced" theme="info" message="目录可发现不等于已获执行授权。引用时保留服务版本，发布与运行仍重新检查权限。" /></template>
    </t-drawer>
  </section>
</template>
<script setup lang="ts">
import { Alert as TAlert } from "tdesign-vue-next/es/alert";
import { Button as TButton } from "tdesign-vue-next/es/button";
import { Drawer as TDrawer } from "tdesign-vue-next/es/drawer";
import { Input as TInput } from "tdesign-vue-next/es/input";
import { Loading as TLoading } from "tdesign-vue-next/es/loading";
import { TabPanel as TTabPanel, Tabs as TTabs } from "tdesign-vue-next/es/tabs";
import ServiceContract from "../components/ServiceContract.vue";
import { ref, computed, reactive, watch } from "vue";
import { useRoute } from "vue-router";
import { api, prefix } from "../api";
import { PagedCollection } from "../pagedCollection";
import { serviceKey } from "../capabilitySelection";
import EmptyState from "@/components/EmptyState.vue";
const route=useRoute(), isServices=computed(()=>route.path==="/services"), query=ref(""), selected=ref<any>(null), contractTab=ref("input");
watch(selected,()=>{contractTab.value="input"});
const key=(item:any)=>isServices.value?serviceKey({service_id:item.service_id,version:item.service_version}):item.asset_id;
const collection=reactive(new PagedCollection<any>(async(query,cursor)=>{
  const params=new URLSearchParams({query});if(cursor!==null)params.set("cursor",cursor);
  return api(prefix+"/selectors/"+(isServices.value?"services":"assets")+"?"+params);
},key));
function refresh(){selected.value=null;void collection.reset(query.value)}
function retry(){if(collection.items.length&&collection.nextCursor!==null)void collection.next();else refresh()}
watch([query,isServices],refresh,{immediate:true});
watch(()=>collection.error, error=>{if(error)selected.value=null});
function kindName(item:any){return ({KNOWLEDGE:"知识库",METRIC:"业务指标",SCENARIO:"场景服务",RULE:"业务规则"} as Record<string,string>)[item.asset_type||item.service_kind] || (isServices.value?"认知服务":"认知资产")}
function version(item:any){return item.service_version||item.effective_version||"目录未提供版本"}
</script>
