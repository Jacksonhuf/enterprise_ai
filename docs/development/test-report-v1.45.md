# 测试报告 v1.45（R18 +3 抽页、豁免关闭、Figma 首条 matched）

## 范围

- R18：**41** 抽页（+`services-engineer`、`governance-access-groups`、`governance-access-roles`）
- R02 matrix v1.45；shared-shell 豁免 v1.45 关闭
- Figma 字段：`enter-app-cta` → **matched**（工程彩排，非产品签）

## 本地

```bash
go build -C backend/platform -tags=nomsgpack -o /tmp/cognara-dev ./cmd/cognara-dev
cd frontend && npm run build
CHROMIUM_EXECUTABLE_PATH=/usr/bin/google-chrome \
  COGNARA_R18_CAPTURE=./r18-baselines COGNARA_SMOKE_BINARY=/tmp/cognara-dev npm run test:cognara-browser
python3 scripts/validate-r02-page-api-acceptance-matrix.py docs/development/fixtures/r02-page-api-acceptance-matrix-v1.45.json
npm run verify:r18-baseline -- /tmp/r18-spots
```

**R18 baseline gate OK**；validator 全绿。

## 结论

关闭 v1.44 三项 shell 豁免；R18 专署抽页到位。Figma 产品签字仍开放。