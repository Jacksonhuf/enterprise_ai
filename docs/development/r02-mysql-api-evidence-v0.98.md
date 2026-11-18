# R02 MySQL API 证据（v0.98）

| API | 测试 | 说明 |
|-----|------|------|
| `PUT …/settings/profile` | `TestR20Schema8DualHostSettingsProfileAndGeneralCAS` | 双主机 CAS |
| `PUT …/settings/general` | 同上 | 双主机 CAS |
| public 写 + 治理/配置读 | `TestR20Schema8MixedSmokeAfterPublicWrite` | R01 写后烟雾 |

索引：[test-report-v0.98.md](test-report-v0.98.md)