export interface Config {
  basic: { name: string; description: string };
  model_prompt: { model_id: string; prompt: string };
  capability_refs: {
    asset_refs: { asset_id: string }[];
    service_refs: { service_id: string; version: string }[];
    tool_names: string[];
  };
  conversation_defaults: { welcome: string; suggested_questions: string[] };
}
export interface Release {
  release_id: string;
  app_id: string;
  config_snapshot: Config;
  release_note: string;
  published_at: string;
  published_by: string;
}
export interface Application {
  app_id: string;
  agent_id: string;
  tenant_id: string;
  owner_id: string;
  draft: Config;
  draft_revision: string;
  current_release_id: string | null;
  releases: Release[];
  updated_at: string;
}
export interface Message {
  id: string;
  session_id: string;
  role: "user" | "assistant";
  content: string;
  created_at: string;
  application_release_id?: string;
  turn_id?: string;
  execution_id?: string;
  result?: ServiceResult;
  is_completed: boolean;
}
export interface ServiceResult {
  service_execution_id: string;
  status: string;
  result: {
    summary: string;
    type: string;
    columns: { name: string; data_type: string }[];
    rows: (string | number)[][];
  };
  step_executions: { execution_id: string }[];
}
export interface Principal {
  tenant_id: string;
  actor_user_id: string;
  name: string;
  role: "user" | "engineer" | "admin";
  scope: string;
}
export interface Asset {
  asset_id: string;
  asset_type: string;
  name: string;
  summary: string;
  effective_version: string;
}
export interface Service {
  service_id: string;
  service_version: string;
  name: string;
  input_schema: object;
  output_schema: object;
}
export interface Grant {
  grant_id: string;
  subject_type: string;
  subject_id: string;
  resource_type: string;
  resource_id: string;
  action: string;
  scope: string;
  active: boolean;
}
