import {APIError,formatAPIError} from "./api";
export type ExceptionStatus="OPEN"|"ACKNOWLEDGED"|"RESOLVED";
export interface ExceptionSnapshot {id:string;revision:string;status:ExceptionStatus;name?:string;request_id?:string}
const actions:Record<ExceptionStatus,{label:string;status:ExceptionStatus}[]>={
 OPEN:[{label:"确认",status:"ACKNOWLEDGED"},{label:"标记解决",status:"RESOLVED"}],
 ACKNOWLEDGED:[{label:"标记解决",status:"RESOLVED"}],
 RESOLVED:[{label:"重新打开",status:"OPEN"}]
};
export function exceptionTransitions(status:unknown) {
 return typeof status==="string"&&Object.hasOwn(actions,status)?actions[status as ExceptionStatus].map(action=>({...action})):[];
}
export class ExceptionTransition {
 visible=false;saving=false;blocked=false;error="";record:ExceptionSnapshot|null=null;targetStatus:ExceptionStatus="ACKNOWLEDGED";
 private generation=0;private plan:{record:ExceptionSnapshot;status:ExceptionStatus}|null=null;
 constructor(private write:(id:string,body:{expected_revision:string;status:ExceptionStatus})=>Promise<ExceptionSnapshot>){}
 open(input:ExceptionSnapshot,status:string) {
  if(this.saving||this.blocked)return false;
  const action=exceptionTransitions(input?.status).find(action=>action.status===status);
  if(typeof input?.id!=="string"||!input.id||typeof input.revision!=="string"||!input.revision||!action){
   this.error="异常已变化或状态迁移无效，请重新加载列表";return false;
  }
  const record:ExceptionSnapshot={id:input.id,revision:input.revision,status:input.status,
   name:typeof input.name==="string"?input.name:"未命名异常",request_id:typeof input.request_id==="string"?input.request_id:undefined};
  this.generation++;this.record={...record};this.targetStatus=action.status;this.plan={record,status:action.status};this.error="";this.visible=true;return true;
 }
 close(){if(this.saving)return;this.generation++;this.visible=false;this.plan=null;this.record=null;if(!this.blocked)this.error=""}
 invalidate(){this.generation++;this.visible=false;this.plan=null;this.record=null}
 get canConfirm(){return this.visible&&!!this.plan&&!this.saving&&!this.blocked}
 async confirm():Promise<ExceptionSnapshot|null>{
  if(!this.canConfirm)return null;
  const generation=this.generation,plan=this.plan!;this.saving=true;this.error="";
  try{
   const updated=await this.write(plan.record.id,{expected_revision:plan.record.revision,status:plan.status});
   if(generation!==this.generation)return null;
   if(updated?.id!==plan.record.id||typeof updated.revision!=="string"||!updated.revision||updated.revision===plan.record.revision||updated.status!==plan.status||
    (plan.record.request_id!==undefined&&updated.request_id!==plan.record.request_id))
    throw new Error("状态更新响应与已审阅异常不匹配");
   this.visible=false;this.plan=null;return updated;
  }catch(e){
   if(generation===this.generation){
    this.blocked=true;
    this.error=(!(e instanceof APIError)||e.status>=500?"处理结果未确认，请重新加载异常列表核对，勿直接重复提交：":
     e.status===409?"异常已更新，请重新加载并核对状态：":"状态处理未完成，请重新加载列表核对：")+formatAPIError(e);
   }
   return null;
  }finally{if(generation===this.generation)this.saving=false}
 }
}
