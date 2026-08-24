import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: '.',
  timeout: 30000,
  use: { baseURL: process.env.DASHBOARD_URL || 'http://127.0.0.1:28371' },
  reporter: 'list',
})
