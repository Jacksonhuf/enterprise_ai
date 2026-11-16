// A bounded read-only projection, not a JSON Schema validator or reference resolver.
export interface ContractRow {
  path:string; name:string; depth:number; type:string; required:string; description:string; constraints:string;
}
export interface ContractView { rows:ContractRow[]; notices:string[]; raw:string; rawTruncated:boolean }
const record=(v:unknown):v is Record<string,unknown>=>!!v&&typeof v==="object"&&!Array.isArray(v);
const own=(v:Record<string,unknown>,key:string)=>Object.prototype.hasOwnProperty.call(v,key);
const short=(value:unknown):string=>{
  const text=JSON.stringify(value)??"";
  return text.length>500?text.slice(0,500)+"…（查看原始定义）":text;
};
const pointer=(key:string)=>key.replace(/~/g,"~0").replace(/\//g,"~1");
const labels:Record<string,string>={
  minimum:"最小值",maximum:"最大值",exclusiveMinimum:"排他下界",exclusiveMaximum:"排他上界",
  multipleOf:"步长",minLength:"最短长度",maxLength:"最长长度",pattern:"匹配模式",format:"格式",
  minItems:"最少项数",maxItems:"最多项数",uniqueItems:"元素唯一",minProperties:"最少属性数",
  maxProperties:"最多属性数",additionalProperties:"额外属性",const:"固定值",default:"默认值",
};
export function inspectContract(schema:unknown):ContractView {
  const view:ContractView={rows:[],notices:[],raw:"",rawTruncated:false};
  const notice=(text:string)=>{if(!view.notices.includes(text))view.notices.push(text)};
  if(schema===undefined||schema===null){
    notice("目录未提供此方向的契约定义。");return view;
  }
  try {
    const raw=JSON.stringify(schema,null,2);
    view.rawTruncated=raw.length>64000;view.raw=raw.slice(0,64000);
    if(view.rawTruncated)notice("原始定义过大，仅展示前 64000 字符；请通过服务目录接口获取完整定义。");
  }catch{notice("定义无法序列化，请刷新目录核对。")}
  const active=new WeakSet<object>();
  function walk(value:unknown,path:string,name:string,depth:number,required:string) {
    if(view.rows.length>=200){notice("字段超过 200 项，表格已截断；请查看原始定义。");return}
    if(depth>8){notice("嵌套超过 8 层，表格已截断；请查看原始定义。");return}
    if(value===true||value===false){
      view.rows.push({path,name,depth,required,type:value?"任意值":"不允许任何值",description:"",constraints:"布尔 Schema"});return;
    }
    if(!record(value)){
      view.rows.push({path,name,depth,required,type:"无效定义",description:"",constraints:"请核对原始定义"});return;
    }
    if(depth===0 && value.type==="object" && !own(value,"properties") && !own(value,"$ref") && !["allOf","anyOf","oneOf"].some(key=>own(value,key)))
      notice("目录未提供具名字段明细，请查看原始定义。");
    const constraints:string[]=[];
    for(const [key,label] of Object.entries(labels))if(own(value,key))constraints.push(label+"："+short(value[key]));
    if(Array.isArray(value.enum)){
      constraints.push("可选值："+short(value.enum.slice(0,12))+(value.enum.length>12?"（共 "+value.enum.length+" 项，查看原始定义）":""));
    }
    if(typeof value.$ref==="string"){
      constraints.push("引用："+value.$ref);notice("引用保持原样，不自动展开或请求外部地址；请查看原始定义。");
    }
    for(const key of ["allOf","anyOf","oneOf","not","if","then","else","dependentSchemas","dependentRequired","unevaluatedProperties","contains"])
      if(own(value,key)){constraints.push(key+"：查看原始定义");notice("组合与条件等约束未在表格展开，完整契约以原始定义为准。")}
    if(Array.isArray(value.required)&&value.required.length)constraints.push("必填属性："+short(value.required));
    view.rows.push({path,name,depth,required,
      type:typeof value.type==="string"?value.type:Array.isArray(value.type)?value.type.map(String).join(" | "):"类型未声明",
      description:typeof value.description==="string"?value.description:"",
      constraints:constraints.join("\n")||"未声明"});
    if(active.has(value)){notice("检测到循环结构，已停止展开。");return}
    active.add(value);
    if(record(value.properties)){
      for(const [key,child] of Object.entries(value.properties)){
        if(view.rows.length>=200){notice("字段超过 200 项，表格已截断；请查看原始定义。");break}
        walk(child,path+"/properties/"+pointer(key),key,depth+1,Array.isArray(value.required)&&value.required.includes(key)?"是":"否");
      }
    }
    if(Array.isArray(value.prefixItems)){
      notice("数组位置项按 prefixItems 展示，剩余项仍以原始定义为准。");
      for(let i=0;i<Math.min(value.prefixItems.length,200);i++)walk(value.prefixItems[i],path+"/prefixItems/"+i,"位置项 "+i,depth+1,"—");
      if(value.prefixItems.length>200)notice("数组位置项超过 200 项，表格已截断；请查看原始定义。");
    }
    if(own(value,"items")){
      if(Array.isArray(value.items))notice("元组式 items 未展开，请查看原始定义。");
      else walk(value.items,path+"/items","数组元素",depth+1,"—");
    }
    active.delete(value);
  }
  walk(schema,"#","根对象",0,"—");
  return view;
}
