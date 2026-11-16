import test from 'node:test';
import assert from 'node:assert/strict';
import { formatDateTime, supportedTimeZone, browserTimeZone } from './dateTime';
test('Saved timezones display the same instant on opposite calendar days',()=>{
 const instant='2026-11-16T00:30:00Z';
 assert.match(formatDateTime(instant,'Asia/Shanghai'),/(?=.*2026\/11\/16)(?=.*08:30:00)(?=.*GMT\+8)/);
 assert.match(formatDateTime(instant,'America/New_York'),/(?=.*2026\/11\/15)(?=.*19:30:00)(?=.*GMT-5)/);
});
test('DST offsets follow the event instant, not the current date',()=>{
 assert.match(formatDateTime('2026-07-01T12:00:00Z','America/New_York'),/(?=.*08:00:00)(?=.*GMT-4)/);
 assert.match(formatDateTime('2026-12-01T12:00:00Z','America/New_York'),/(?=.*07:00:00)(?=.*GMT-5)/);
});
test('Offset representations and UTC display the same instant',()=>{
 assert.equal(formatDateTime('2026-11-16T08:30:00+08:00','UTC'),formatDateTime('2026-11-16T00:30:00Z','UTC'));
 assert.match(formatDateTime('2026-11-16T00:30:00Z','Asia/Kolkata'),/(?=.*06:00:00)(?=.*GMT\+5:30)/);
});
test('Time-only chat labels still identify the offset',()=>{
 const value=formatDateTime('2026-11-16T00:30:00Z','Asia/Shanghai',true);
 assert.match(value,/08:30:00/);assert.match(value,/GMT\+8/);assert.doesNotMatch(value,/2026/);
});
test('Missing, unsupported and invalid timestamps have explicit fallback behavior',()=>{
 for(const value of [undefined,null,'','not-a-date'])assert.equal(formatDateTime(value,'UTC'),'—');
 for(const value of [undefined,'','Local','Mars/Base'])assert.equal(supportedTimeZone(value),'');
 assert.equal(formatDateTime('2026-11-16T00:30:00Z','Mars/Base'),formatDateTime('2026-11-16T00:30:00Z'));
 assert.ok(browserTimeZone());
});
