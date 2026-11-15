import type { Config } from "./types";
export interface ConfigDifference { label: string; before: string; after: string }
const lines = (values: string[]) => [...new Set(values)].sort().join("\n");
export function compareConfig(before: Config, after: Config): ConfigDifference[] {
  const fields: [string, (c: Config) => string][] = [
    ["应用名称", c => c.basic.name],
    ["业务描述", c => c.basic.description],
    ["主模型", c => c.model_prompt.model_id],
    ["提示词", c => c.model_prompt.prompt],
    ["认知资产", c => lines(c.capability_refs.asset_refs.map(x => x.asset_id))],
    ["认知服务版本", c => lines(c.capability_refs.service_refs.map(x => `${x.service_id} · ${x.version}`))],
    ["受控能力", c => lines(c.capability_refs.tool_names)],
    ["欢迎语", c => c.conversation_defaults.welcome],
    ["示例问题", c => c.conversation_defaults.suggested_questions.join("\n")],
  ];
  return fields.map(([label, read]) => ({ label, before: read(before), after: read(after) }))
    .filter(row => row.before !== row.after);
}
export function modelLabel(id: string, models: { id: string; name: string }[]) {
  if (id === "validation-only") return "验证运行 · 未启用模型解释";
  return models.find(model => model.id === id)?.name || id || "未配置";
}
