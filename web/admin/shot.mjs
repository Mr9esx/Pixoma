import { chromium } from 'playwright'
const b = await chromium.launch()
const p = await b.newPage({ viewport: { width: 1440, height: 900 } })
await p.goto('http://127.0.0.1:5179/login', { waitUntil: 'networkidle', timeout: 30000 }).catch(e => console.log('ERR', e.message))
await p.waitForTimeout(2000)
await p.screenshot({ path: '/tmp/login-full.png' })
// 放大关键区域（右侧点阵 + mask 聚光区域）
await p.locator('svg,img,canvas').first().screenshot({ path: '/tmp/x.png' }).catch(()=>{})
const body = await p.evaluate(() => document.body.innerHTML.length)
console.log('body chars', body)
await b.close()
