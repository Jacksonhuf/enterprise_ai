# R18 抽页 PNG 基线（v1.05）

与 `scripts-smoke.mjs` 中 R18 spot 列表一致（v1.08：13 面）。更新基线：

```bash
go build -C backend/platform -tags=nomsgpack -o /tmp/cognara-dev ./cmd/cognara-dev
cd frontend && npm run build
COGNARA_R18_CAPTURE=./r18-baselines COGNARA_SMOKE_BINARY=/tmp/cognara-dev npm run test:cognara-browser
git add r18-baselines/*.png r18-baselines/manifest.json
```

CI 对比门：`npm run verify:r18-baseline -- /tmp/r18-spots`（默认 ≤2% 像素差）。