const requestPattern=/^[A-Za-z0-9_-]{1,128}$/;
export function auditRequestId(value:unknown):string {
 return typeof value==="string"&&requestPattern.test(value)?value:"";
}
export function auditTraceLocation(value:unknown) {
 const id=auditRequestId(value);
 return id?{path:"/governance/audit",query:{request_id:id}}:null;
}
