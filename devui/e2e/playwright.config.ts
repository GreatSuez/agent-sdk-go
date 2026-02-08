import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  timeout: 60_000,
  use: {
    baseURL: process.env.DEVUI_BASE_URL || 'http://127.0.0.1:7070',
    headless: true,
  },
  reporter: [['list']],
});
