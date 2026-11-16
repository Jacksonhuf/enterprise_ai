export interface GovernanceFilters { query: string; status: string; outcome: string; unit: string; range: (string | number | Date)[] | null }
export const pagedGovernance = (section: string) => ["audit","exceptions","usage","grants"].includes(section);
export function governanceParams(section: string, filters: GovernanceFilters, cursor: string | null) {
  const params = new URLSearchParams({query: filters.query});
  if (!pagedGovernance(section)) return params;
  params.set("limit","20");
  if (section === "audit" && filters.outcome) params.set("outcome", filters.outcome);
  if (["exceptions","grants"].includes(section) && filters.status) params.set("status",filters.status);
  if (section === "usage" && filters.unit) params.set("unit", filters.unit);
  const range = filters.range || [];
  if (section !== "grants" && range.length) {
    if (range.length !== 2) throw new Error("请选择完整时间范围");
    const dates = range.map(value => new Date(value));
    if (dates.some(date => !Number.isFinite(date.getTime())) || dates[0] > dates[1]) throw new Error("时间范围无效");
    params.set("from_time",dates[0].toISOString()); params.set("to_time",dates[1].toISOString());
  }
  if (cursor !== null) params.set("cursor",cursor);
  return params;
}
