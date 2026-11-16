import {APIError,formatAPIError} from "./api";
import type {Grant} from "./types";
export type GrantInput=Pick<Grant,"subject_type"|"subject_id"|"resource_type"|"resource_id"|"action"|"scope">;
export interface GrantReviewRow {label:string;value:string}
interface GrantTransport {create:(body:GrantInput)=>Promise<Grant>;revoke:(id:string)=>Promise<{success:boolean}>}
type Plan={kind:"CREATE";body:GrantInput}|{kind:"REVOKE";id:string};
const keys:(keyof GrantInput)[]=["subject_type","subject_id","resource_type","resource_id","action","scope"];
export class GrantMutation {
 visible=false;saving=false;blocked=false;error="";kind:"CREATE"|"REVOKE"="CREATE";rows:GrantReviewRow[]=[];
 private generation=0;private plan:Plan|null=null;
 constructor(private transport:GrantTransport){}
 openCreate(input:GrantInput,rows:GrantReviewRow[]){
  if(this.saving||this.blocked)return false;
  const body=Object.fromEntries(keys.map(key=>[key,input[key]])) as GrantInput;
  if(keys.some(key=>typeof body[key]!=="string"||!body[key].trim())){this.error="请完整选择授权主体、资源、操作和范围";return false}
  this.generation++;this.kind="CREATE";this.plan={kind:"CREATE",body};this.rows=rows.map(row=>({...row}));this.error="";this.visible=true;return true;
 }
 openRevoke(grant:Grant,rows:GrantReviewRow[]){
  if(this.saving||this.blocked)return false;
  if(!grant?.grant_id||grant.active!==true){this.error="授权已变化，请重新加载列表";return false}
  this.generation++;this.kind="REVOKE";this.plan={kind:"REVOKE",id:grant.grant_id};this.rows=rows.map(row=>({...row}));this.error="";this.visible=true;return true;
 }
 close(){if(this.saving)return;this.generation++;this.visible=false;this.plan=null;if(!this.blocked)this.error=""}
 invalidate(){this.generation++;this.visible=false;this.plan=null}
 async confirm():Promise<Grant|{success:boolean}|null>{
  if(!this.visible||!this.plan||this.saving||this.blocked)return null;
  const generation=this.generation,plan=this.plan;this.saving=true;this.error="";
  try{
   const result=plan.kind==="CREATE"?await this.transport.create({...plan.body}):await this.transport.revoke(plan.id);
   if(generation!==this.generation)return null;
   if(plan.kind==="CREATE"){
    const grant=result as Grant;
    if(typeof grant?.grant_id!=="string"||!grant.grant_id||grant.active!==true||keys.some(key=>grant[key]!==plan.body[key]))
     throw new Error("授权响应与已审阅内容不匹配");
   }else if((result as {success:boolean})?.success!==true)throw new Error("撤销响应未确认成功");
   this.visible=false;this.plan=null;return result;
  }catch(e){
   if(generation===this.generation){
    const uncertain=!(e instanceof APIError)||e.status>=500;
    this.blocked=uncertain||(e instanceof APIError&&[401,403,404,409].includes(e.status));
    this.error=(uncertain?"操作结果未确认，请重新加载授权列表核对，勿直接重复提交：":this.blocked?"授权或权限已变化，请重新加载列表核对：":"")+formatAPIError(e);
   }
   return null;
  }finally{if(generation===this.generation)this.saving=false}
 }
}
