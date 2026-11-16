import { APIError, formatAPIError } from "./api";
export interface SettingsSnapshot { values: Record<string,string>; revision: string }
const fields:Record<string,string[]>={profile:["display_name"],general:["language","timezone","density"],public:["notice"]};
export function settingsValues(section:string,values:Record<string,unknown>):Record<string,string>{
  return Object.fromEntries((fields[section]||[]).map(key=>[key,typeof values[key]==="string"?values[key]:""]));
}
export class SettingsDraft {
  values:Record<string,string>={}; revision=""; loaded=false; loading=false; saving=false; conflict=false; error="";
  private baseline="{}";
  private generation=0;
  constructor(private section:string,private read:()=>Promise<SettingsSnapshot>,private write:(payload:{values:Record<string,string>;expected_revision:string})=>Promise<SettingsSnapshot>){}
  get dirty(){return JSON.stringify(this.values)!==this.baseline}
  get canSave(){return this.loaded&&!this.loading&&!this.saving&&!this.conflict&&this.dirty}
  async load(){
    const generation=++this.generation;
    this.loading=true;this.saving=false;this.baseline="{}";this.loaded=false;this.conflict=false;this.error="";this.values={};this.revision="";
    try {
      const snapshot=await this.read();if(generation!==this.generation)return;
      if(!snapshot.revision||!snapshot.values||typeof snapshot.values!=="object")throw new Error("配置响应缺少有效版本，请重新加载");
      this.values=settingsValues(this.section,snapshot.values);this.revision=snapshot.revision;this.baseline=JSON.stringify(this.values);this.loaded=true;
    }catch(e){if(generation===this.generation)this.error=formatAPIError(e)}
    finally{if(generation===this.generation)this.loading=false}
  }
  async save(){
    if(!this.canSave)return false;
    const generation=this.generation,snapshot={...this.values};this.saving=true;this.error="";
    try{
      const saved=await this.write({values:snapshot,expected_revision:this.revision});if(generation!==this.generation)return false;
      if(!saved.revision||!saved.values||typeof saved.values!=="object")throw new Error("保存结果缺少有效版本，请重新加载核对");
      const normalized=settingsValues(this.section,saved.values);
      if(JSON.stringify(this.values)===JSON.stringify(snapshot))this.values=normalized;
      this.baseline=JSON.stringify(normalized);this.revision=saved.revision;return true;
    }catch(e){if(generation===this.generation){this.error=formatAPIError(e);this.conflict=e instanceof APIError&&e.status===409}return false}
    finally{if(generation===this.generation)this.saving=false}
  }
}
