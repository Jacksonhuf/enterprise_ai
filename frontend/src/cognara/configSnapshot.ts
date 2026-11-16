import type {Config} from "./types";
const copy=<T>(value:T):T=>JSON.parse(JSON.stringify(value));
export const configCopy=(value:Config):Config=>{
 if(!value||typeof value!=="object")throw new Error("应用响应缺少完整配置，请重新加载");
 const config=copy(value);
 if(!config?.basic||!config.model_prompt||!config.capability_refs||!config.conversation_defaults)throw new Error("应用响应缺少完整配置，请重新加载");
 config.capability_refs.asset_refs??=[];config.capability_refs.service_refs??=[];config.capability_refs.tool_names??=[];config.conversation_defaults.suggested_questions??=[];
 if(!Array.isArray(config.capability_refs.asset_refs)||!Array.isArray(config.capability_refs.service_refs)||!Array.isArray(config.capability_refs.tool_names)||!Array.isArray(config.conversation_defaults.suggested_questions))throw new Error("应用配置列表格式无效，请重新加载");
 return config;
};
