import { APIError, formatAPIError } from "./api";
import type { Application,Config,Release } from "./types";
export interface ReleaseCheck { status:string; check_id?:string; draft_revision?:string; valid_until?:string; issues?:string[] }
export const emptyConfig=():Config=>({basic:{name:"",description:""},model_prompt:{model_id:"validation-only",prompt:""},capability_refs:{asset_refs:[],service_refs:[],tool_names:[]},conversation_defaults:{welcome:"",suggested_questions:[]}});
const copy=<T>(value:T):T=>JSON.parse(JSON.stringify(value));
export const configCopy=(value:Config):Config=>{
 if(!value||typeof value!=="object")throw new Error("应用响应缺少完整配置，请重新加载");
 const config=copy(value);
 if(!config?.basic||!config.model_prompt||!config.capability_refs||!config.conversation_defaults)throw new Error("应用响应缺少完整配置，请重新加载");
 config.capability_refs.asset_refs??=[];config.capability_refs.service_refs??=[];config.capability_refs.tool_names??=[];config.conversation_defaults.suggested_questions??=[];
 if(!Array.isArray(config.capability_refs.asset_refs)||!Array.isArray(config.capability_refs.service_refs)||!Array.isArray(config.capability_refs.tool_names)||!Array.isArray(config.conversation_defaults.suggested_questions))throw new Error("应用配置列表格式无效，请重新加载");
 return config;
};
const appCopy=(value:Application):Application=>{
 if(!value?.app_id||!value.draft_revision)throw new Error("应用响应缺少有效版本，请重新加载");
 const application=copy(value);application.releases??=[];application.current_release_id??=null;
 if(!Array.isArray(application.releases))throw new Error("发布记录格式无效，请重新加载");
 return application;
};
export interface DraftTransport {
 read:()=>Promise<Application>;
 save:(payload:Config & {expected_draft_revision:string})=>Promise<Application>;
 check:(payload:{expected_draft_revision:string})=>Promise<ReleaseCheck>;
 publish:(payload:{check_id:string;expected_draft_revision:string;expected_current_release_id:string|null;release_note:string})=>Promise<Release>;
}
export class ApplicationDraft {
 application:Application|null=null;config=emptyConfig();loading=false;saving=false;checking=false;publishing=false;conflict=false;error="";saved=false;checkResult:ReleaseCheck|null=null;
 private baseline=JSON.stringify(this.config);private checkedConfig="";private checkedCurrent:string|null=null;private generation=0;
 constructor(private transport:DraftTransport){}
 invalidate(){this.generation++}
 get dirty(){return JSON.stringify(this.config)!==this.baseline}
 get busy(){return this.loading||this.saving||this.checking||this.publishing}
 get canSave(){return !!this.application&&!this.busy&&!this.conflict&&this.dirty}
 get canCheck(){return !!this.application&&!this.busy&&!this.conflict}
 checkCurrentAt(now:number){
  const check=this.checkResult,expiry=Date.parse(check?.valid_until||"");
  return !!check&&check.status==="PASS"&&!!check.check_id&&check.draft_revision===this.application?.draft_revision&&this.application?.current_release_id===this.checkedCurrent&&!this.dirty&&this.checkedConfig===JSON.stringify(this.config)&&Number.isFinite(expiry)&&now<expiry;
 }
 canPublishAt(now:number){return !this.busy&&!this.conflict&&this.checkCurrentAt(now)}
 async load(){
  const generation=++this.generation;this.loading=true;this.saving=false;this.checking=false;this.publishing=false;this.application=null;this.config=emptyConfig();this.baseline=JSON.stringify(this.config);this.conflict=false;this.error="";this.saved=false;this.checkResult=null;
  try{const application=await this.transport.read();if(generation!==this.generation)return;const accepted=appCopy(application),config=configCopy(accepted.draft);this.application=accepted;this.config=config;this.baseline=JSON.stringify(config)}
  catch(e){if(generation===this.generation)this.error=formatAPIError(e)}finally{if(generation===this.generation)this.loading=false}
 }
 async save(){
  if(!this.canSave)return false;
  const generation=this.generation,snapshot=copy(this.config),sent=JSON.stringify(snapshot),revision=this.application!.draft_revision;
  this.saving=true;this.error="";this.checkResult=null;
  try{
   const saved=await this.transport.save({...snapshot,expected_draft_revision:revision});if(generation!==this.generation)return false;
   const metadata=appCopy(saved),accepted=configCopy(metadata.draft);if(metadata.app_id!==this.application!.app_id)throw new Error("保存响应与当前应用不匹配，请重新加载核对");this.application=metadata;this.baseline=JSON.stringify(accepted);if(JSON.stringify(this.config)===sent)this.config=accepted;this.saved=true;return true;
  }catch(e){if(generation===this.generation){this.error=formatAPIError(e);this.conflict=e instanceof APIError&&e.status===409&&e.code==="VERSION_INCOMPATIBLE"}return false}finally{if(generation===this.generation)this.saving=false}
 }
 async checkPublish(){
  if(!this.canCheck)return false;
  if(this.dirty&&!(await this.save()))return false;
  if(this.dirty){this.error="保存期间产生了新的修改，请先保存当前草稿再检查";return false}
  const generation=this.generation,revision=this.application!.draft_revision,snapshot=JSON.stringify(this.config),current=this.application!.current_release_id;
  this.checking=true;this.error="";this.checkResult=null;
  try{
   const checked=await this.transport.check({expected_draft_revision:revision});if(generation!==this.generation)return false;
   if(this.application?.draft_revision!==revision||JSON.stringify(this.config)!==snapshot){this.error="检查期间草稿已变化，请重新检查";return false}
   this.checkResult=copy(checked);this.checkedConfig=snapshot;this.checkedCurrent=current;return true;
  }catch(e){if(generation===this.generation){this.error=formatAPIError(e);this.conflict=e instanceof APIError&&e.status===409&&e.code==="VERSION_INCOMPATIBLE"}return false}finally{if(generation===this.generation)this.checking=false}
 }
 async publish(note:string):Promise<Release|null>{
  if(!this.canPublishAt(Date.now())){this.error="发布检查已过期或草稿有变更，请重新检查";return null}
  const generation=this.generation,checked=this.checkResult!;this.publishing=true;this.error="";
  try{
   const release=await this.transport.publish({check_id:checked.check_id!,expected_draft_revision:checked.draft_revision!,expected_current_release_id:this.checkedCurrent,release_note:note});
   if(generation!==this.generation)return null;
   if(!release?.release_id||release.app_id!==this.application?.app_id)throw new Error("发布响应与当前应用不匹配，请核对发布历史");
   this.application!.current_release_id=release.release_id;this.application!.releases.push(copy(release));this.baseline=JSON.stringify(configCopy(release.config_snapshot));return release;
  }catch(e){if(generation===this.generation){this.error=(!(e instanceof APIError)||e.status>=500?"发布结果未确认，请先核对发布历史再重新检查：":"")+formatAPIError(e);this.conflict=e instanceof APIError&&e.status===409&&e.code==="VERSION_INCOMPATIBLE"}return null}finally{if(generation===this.generation){this.publishing=false;this.checkResult=null}}
 }
}
