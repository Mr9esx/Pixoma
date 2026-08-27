import { chromium } from 'playwright'
const b = await chromium.launch()
const p = await b.newPage({ viewport: { width: 1440, height: 900 } })
await p.goto('http://localhost:5179/login', { waitUntil: 'networkidle', timeout: 30000 }).catch(async e => { console.log('goto login err', e.message); await p.goto('http://localhost:5179/', { waitUntil: 'networkidle', timeout: 30000 }) })
await p.waitForTimeout(1500)
await p.screenshot({ path: '/tmp/login-full.png' })
const info = await p.evaluate(() => {
  const nodes = Array.from(document.querySelectorAll('[style*="radial-gradient"]'))
  return nodes.map(n => {
    const s = getComputedStyle(n)
    return {
      cls: n.className,
      bgImage: s.backgroundImage,
      maskImage: s.maskImage,
      webkitMask: s.webkitMaskImage,
      hasBg: s.backgroundImage && s.backgroundImage.includes('radial-gradient'),
    }
  })
})
console.log(JSON.stringify(info, null, 2))
await b.close()
