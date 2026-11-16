import { APIError, formatAPIError } from "./api";
export interface SessionSummary { session_id:string; app_id:string; title:string; revision:string }
export class SessionRename {
  visible=false; title=""; error=""; saving=false; conflict=false;
  private target:SessionSummary|null=null;
  private generation=0;
  constructor(private write:(id:string,payload:{title:string;expected_revision:string})=>Promise<SessionSummary>){}
  open(target:SessionSummary) { this.generation++;this.target={...target};this.title=target.title;this.error="";this.conflict=false;this.saving=false;this.visible=true; }
  close(){this.generation++;this.visible=false;this.saving=false;this.target=null;}
  async save(){
    if(this.saving||this.conflict||!this.target)return false;
    const title=this.title.trim();
    if(!title||[...title].length>80){this.error="请输入1到80个字符的对话标题";return false;}
    const generation=this.generation,target=this.target;
    this.saving=true;this.error="";
    try{
      const saved=await this.write(target.session_id,{title,expected_revision:target.revision});
      if(generation!==this.generation)return false;
      if(!saved.revision||saved.session_id!==target.session_id)throw new Error("保存结果未确认，请重新加载历史核对");
      this.visible=false;return true;
    }catch(e){if(generation===this.generation){this.error=formatAPIError(e);this.conflict=e instanceof APIError&&e.status===409;}return false;}
    finally{if(generation===this.generation)this.saving=false;}
  }
}
