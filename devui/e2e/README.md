# DevUI E2E

Run Playwright checks against a running DevUI server:

```bash
DEVUI_BASE_URL=http://127.0.0.1:7070 npx playwright test framework/devui/e2e --config framework/devui/e2e/playwright.config.ts
```

Scenarios covered:
- failure -> diagnosis -> intervention visibility
- runtime DLQ/requeue controls presence
