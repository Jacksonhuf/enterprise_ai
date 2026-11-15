export interface ServiceReference { service_id: string; version: string }
export const serviceKey = (ref: ServiceReference) => JSON.stringify([ref.service_id, ref.version]);
export function toggleServiceVersion(selected: string[], key: string, checked: boolean, refs: ServiceReference[]) {
  const target = refs.find(ref => serviceKey(ref) === key);
  if (!target) throw new Error("服务版本不在当前引用或目录中");
  if (!checked) return selected.filter(value => value !== key);
  const sameService = new Set(refs.filter(ref => ref.service_id === target.service_id).map(serviceKey));
  return [...selected.filter(value => !sameService.has(value)), key];
}
export function resolveServiceSelection(keys: string[], refs: ServiceReference[]) {
  const byKey = new Map(refs.map(ref => [serviceKey(ref), ref]));
  const result = keys.map(key => { const ref=byKey.get(key); if (!ref) throw new Error("服务引用已变化，请重新打开选择器"); return {...ref}; });
  if (new Set(result.map(ref => ref.service_id)).size !== result.length) throw new Error("同一服务只能引用一个版本");
  return result;
}
