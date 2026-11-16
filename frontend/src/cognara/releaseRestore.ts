import { APIError, formatAPIError } from "./api";
import { configCopy } from "./applicationDraft";
import { compareConfig, type ConfigDifference } from "./configReview";
import type { Application, Config, Release } from "./types";

export class ReleaseRestore {
  visible=false; saving=false; blocked=false; error=""; source=""; changes:ConfigDifference[]=[];
  private plan:{appId:string;revision:string;config:Config}|null=null;
  private generation=0;
  constructor(private save:(id:string,payload:Config & {expected_draft_revision:string})=>Promise<Application>){}
  open(application:Application, release:Release) {
    if(this.saving||this.blocked)return false;
    if(!application.draft_revision||!release.release_id||release.app_id!==application.app_id) {
      this.error="发布记录与当前应用不匹配，请重新加载";return false;
    }
    const config=configCopy(release.config_snapshot);
    this.changes=compareConfig(configCopy(application.draft),config);
    if(!this.changes.length)return false;
    this.generation++;this.plan={appId:application.app_id,revision:application.draft_revision,config};
    this.source=release.release_id;this.error="";this.visible=true;return true;
  }
  close(){if(this.saving)return;this.visible=false;this.plan=null;this.generation++}
  invalidate(){this.generation++;this.visible=false;this.plan=null}
  async confirm():Promise<Application|null>{
    if(!this.visible||!this.plan||this.saving||this.blocked)return null;
    const generation=this.generation,plan=this.plan;
    this.saving=true;this.error="";
    try {
      const result=await this.save(plan.appId,{...configCopy(plan.config),expected_draft_revision:plan.revision});
      if(generation!==this.generation)return null;
      if(result?.app_id!==plan.appId||!result.draft_revision||result.draft_revision===plan.revision)
        throw new Error("恢复响应缺少有效草稿版本");
      if(compareConfig(plan.config,configCopy(result.draft)).length)
        throw new Error("恢复响应配置不匹配");
      this.visible=false;this.plan=null;return result;
    } catch(e) {
      if(generation===this.generation){
        this.blocked=true;
        this.error=(e instanceof APIError&&e.status===409
          ?"草稿已变化，请重新加载并核对后再恢复："
          :!(e instanceof APIError)||e.status>=500
          ?"恢复结果未确认，请重新加载草稿核对，勿直接重复提交："
          :"恢复未完成，请重新加载并核对：")+formatAPIError(e);
      }
      return null;
    } finally {if(generation===this.generation)this.saving=false}
  }
}
