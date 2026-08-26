import { chromium } from 'playwright'
const exe = '/Users/mr9esx/Library/Caches/ms-playwright/chromium-1228/chrome-mac-arm64/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing'
const browser = await chromium.launch({ executablePath: exe, headless: true })
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
const errs = []
page.on('pageerror', e => errs.push('PAGEERR: ' + e.message))
page.on('console', m => { if (m.type()==='error') errs.push('CONSOLE: ' + m.text()) })
const url = process.argv[2] || 'http://127.0.0.1:5174/cases/new'
await page.goto(url, { waitUntil: 'networkidle', timeout: 30000 }).catch(()=>{})
await page.waitForTimeout(2500)
await page.screenshot({ path: '/tmp/table1.png', fullPage: false })
console.log('JS errors:', errs.slice(0,10).join('\n'))
await browser.close()
