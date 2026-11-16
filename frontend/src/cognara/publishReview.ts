import {configCopy} from "./configSnapshot";
import {compareConfig,configEntries,type ConfigDifference} from "./configReview";
import type {Application,Config} from "./types";

export interface PublishReview {
  appId:string;
  revision:string;
  currentReleaseId:string|null;
  firstPublication:boolean;
  rows:ConfigDifference[];
}
// Capture strings and metadata, never retain live pointers into an editable draft.
export function buildPublishReview(application:Application,config:Config):PublishReview {
  if(!application.app_id||!application.draft_revision)throw new Error("缺少草稿版本，请重新加载后检查");
  const snapshot=configCopy(config),current=application.current_release_id;
  if(current===null){
    return {appId:application.app_id,revision:application.draft_revision,currentReleaseId:null,firstPublication:true,
      rows:configEntries(snapshot).map(row=>({label:row.label,before:"（尚未发布）",after:row.value}))};
  }
  const matches=application.releases?.filter(release=>release.release_id===current) || [];
  if(typeof current!=="string"||!current||matches.length!==1||matches[0].app_id!==application.app_id)
    throw new Error("无法核对当前发布配置，请重新加载应用后再检查");
  return {appId:application.app_id,revision:application.draft_revision,currentReleaseId:current,firstPublication:false,
    rows:compareConfig(configCopy(matches[0].config_snapshot),snapshot)};
}
