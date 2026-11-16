import {APIError,formatAPIError} from "./api";
import {configCopy} from "./configSnapshot";
import {compareConfig} from "./configReview";
import type {Application} from "./types";
export interface CreateRequest {creation_mode:"BLANK"|"TEMPLATE";name:string;description:string;template_id?:string;expected_template_revision?:string}
interface Transport {list:()=>Promise<Application[]>;read:(id:string)=>Promise<Application>;create:(body:CreateRequest)=>Promise<Application>}
export class ApplicationCreation {
 visible=false;mode:"BLANK"|"TEMPLATE"="BLANK";name="";description="";templateId="";template:Application|null=null;
 options:Application[]=[];templatesLoading=false;previewLoading=false;creating=false;blocked=false;error="";
 private generation=0;private listEpoch=0;private selectionEpoch=0;
 constructor(private transport:Transport){}
 open(){if(this.creating||this.blocked)return;this.generation++;this.listEpoch++;this.selectionEpoch++;this.visible=true;this.mode="BLANK";this.name="";this.description="";this.templateId="";this.template=null;this.options=[];this.templatesLoading=false;this.previewLoading=false;this.blocked=false;this.error=""}
 close(){if(this.creating)return;this.invalidate()}
 invalidate(){this.generation++;this.listEpoch++;this.selectionEpoch++;this.visible=false;this.template=null}
 get canCreate(){return this.visible&&!this.creating&&!this.blocked&&!!this.name.trim()&&Array.from(this.name).length<=80&&(this.mode==="BLANK"||!!this.template&&!this.previewLoading)}
 async loadTemplates(){
  const generation=this.generation,epoch=++this.listEpoch;this.templatesLoading=true;this.options=[];this.error="";
  try{const options=await this.transport.list();if(generation!==this.generation||epoch!==this.listEpoch||this.mode!=="TEMPLATE"||this.creating)return;this.options=options}
  catch(e){if(generation===this.generation&&epoch===this.listEpoch&&this.mode==="TEMPLATE"&&!this.creating)this.error="模板目录读取失败："+formatAPIError(e)}
  finally{if(generation===this.generation&&epoch===this.listEpoch)this.templatesLoading=false}
 }
 async selectTemplate(id:string){
  if(this.creating||this.blocked)return;
  const generation=this.generation,epoch=++this.selectionEpoch;this.templateId=id;this.template=null;this.error="";this.previewLoading=!!id;
  if(!id)return;
  try{
   const value=await this.transport.read(id);
   if(generation!==this.generation||epoch!==this.selectionEpoch||this.mode!=="TEMPLATE")return;
   if(value?.app_id!==id||!value.draft_revision)throw new Error("模板响应缺少有效版本");
   const accepted=JSON.parse(JSON.stringify(value)) as Application;accepted.draft=configCopy(accepted.draft);this.template=accepted;
  }catch(e){if(generation===this.generation&&epoch===this.selectionEpoch)this.error="模板预览读取失败："+formatAPIError(e)}
  finally{if(generation===this.generation&&epoch===this.selectionEpoch)this.previewLoading=false}
 }
 async create():Promise<Application|null>{
  if(!this.canCreate)return null;
  const generation=this.generation,template=this.mode==="TEMPLATE"?this.template:null;
  const body:CreateRequest={creation_mode:this.mode,name:this.name,description:this.description};
  const expected=template?configCopy(template.draft):null;
  if(expected)expected.basic={name:body.name,description:body.description};
  if(template){body.template_id=template.app_id;body.expected_template_revision=template.draft_revision}
  this.creating=true;this.error="";
  try{
   const result=await this.transport.create(body);if(generation!==this.generation)return null;
   if(typeof result?.app_id!=="string"||!result.app_id||typeof result.draft_revision!=="string"||!result.draft_revision||result.current_release_id!==null||!Array.isArray(result.releases)||result.releases.length)
    throw new Error("创建响应缺少有效的新草稿信息");
   const config=configCopy(result.draft);
   if(config.basic.name!==body.name||config.basic.description!==body.description||(expected&&compareConfig(expected,config).length))
    throw new Error("创建响应与已审阅的配置不匹配");
   this.visible=false;return result;
  }catch(e){
   if(generation===this.generation){
    if(e instanceof APIError&&e.status===409&&e.code==="VERSION_INCOMPATIBLE"){
     this.template=null;this.error="模板已更新，请刷新模板预览并核对后再创建："+formatAPIError(e);
    }else if(!(e instanceof APIError)||e.status>=500){
     this.blocked=true;this.error="创建结果未确认，请刷新应用列表核对，勿直接重复提交："+formatAPIError(e);
    }else this.error=formatAPIError(e);
   }
   return null;
  }finally{if(generation===this.generation)this.creating=false}
 }
}
