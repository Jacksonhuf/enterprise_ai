import { test } from "node:test";
import assert from "node:assert/strict";
import { compareConfig, modelLabel } from "./configReview";
import type { Config } from "./types";
const config = (): Config => ({ basic: { name:"补货助手", description:"业务描述" }, model_prompt:{ model_id:"validation-only", prompt:"原提示词" }, capability_refs:{asset_refs:[{asset_id:"a"},{asset_id:"b"}],service_refs:[{service_id:"gap",version:"v1"}],tool_names:["discover","execute"]},conversation_defaults:{welcome:"欢迎",suggested_questions:["问题一","问题二"]} });
test("Config comparison treats resource order as a set but preserves service versions and question order", () => {
  const before = config(), after = config();
  after.capability_refs.asset_refs.reverse(); after.capability_refs.tool_names.reverse();
  assert.deepEqual(compareConfig(before, after), []);
  after.capability_refs.service_refs[0].version = "v2";
  after.conversation_defaults.suggested_questions.reverse();
  const changes = compareConfig(before, after);
  assert.deepEqual(changes.map(x => x.label), ["认知服务版本", "示例问题"]);
  assert.equal(changes[0].before,"gap · v1"); assert.equal(changes[0].after,"gap · v2");
  assert.equal(before.capability_refs.service_refs[0].version,"v1");
});
test("Config comparison includes removals, model changes and multiline prompts without HTML interpretation", () => {
  const before=config(), after=config();
  after.model_prompt={model_id:"business",prompt:"第一行\n<script>文本</script>"};
  after.capability_refs.asset_refs=[]; after.conversation_defaults.welcome="";
  const changes=compareConfig(before,after);
  assert.deepEqual(changes.map(x=>x.label),["主模型","提示词","认知资产","欢迎语"]);
  assert.equal(changes.find(x=>x.label==="认知资产")?.after,"");
  assert.equal(changes.find(x=>x.label==="提示词")?.after,after.model_prompt.prompt);
  assert.equal(modelLabel("business",[{id:"business",name:"企业模型"}]),"企业模型");
  assert.equal(modelLabel("unknown",[]),"unknown");
});
