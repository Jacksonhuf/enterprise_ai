// The server stores instants; timezone preferences affect presentation only.
export function supportedTimeZone(value: unknown): string {
  if (typeof value !== 'string' || !value || value === 'Local') return '';
  try { return new Intl.DateTimeFormat('zh-CN', {timeZone:value}).resolvedOptions().timeZone; }
  catch { return ''; }
}
export function formatDateTime(value: unknown, timeZone = '', timeOnly = false): string {
  if (typeof value !== 'string' || !value.trim()) return '—';
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return '—';
  const zone = supportedTimeZone(timeZone);
  return new Intl.DateTimeFormat('zh-CN', {
    ...(zone ? {timeZone:zone} : {}),
    ...(timeOnly ? {} : {year:'numeric',month:'2-digit',day:'2-digit'}),
    hour:'2-digit',minute:'2-digit',second:'2-digit',hourCycle:'h23',timeZoneName:'shortOffset',
  }).format(date);
}
export function browserTimeZone(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
}
