// Renders icon.svg to the PNG sizes the game, the .exe and the installer need.
// Needs Playwright (and Python with Pillow for the .ico and .bmp files).
const { chromium } = require('playwright');
const fs = require('fs'), path = require('path');
const root = path.join(__dirname, '..', '..');
(async () => {
  const svg = fs.readFileSync(path.join(__dirname, 'icon.svg'), 'utf8');
  const b = await chromium.launch();
  const shot = async (size, out) => {
    const p = await b.newPage({ viewport: { width: size, height: size } });
    await p.setContent(`<html><body style="margin:0;background:transparent">${svg.replace('<svg ', `<svg width="${size}" height="${size}" `)}</body></html>`);
    await p.screenshot({ path: out, omitBackground: true });
    await p.close();
  };
  await shot(1024, path.join(root, 'icon-1024.png'));
  for (const n of [16, 32, 48, 64, 256]) await shot(n, path.join(root, 'assets', `icon-${n}.png`));
  // installer side panel art
  const wiz = async (w, h, out) => {
    const p = await b.newPage({ viewport: { width: w, height: h } });
    await p.setContent(`<html><body style="margin:0;width:${w}px;height:${h}px;overflow:hidden;background:linear-gradient(#0b0e27,#3a2560);display:flex;align-items:center;justify-content:center">${svg.replace('<svg ', `<svg width="${w * 0.9}" height="${w * 0.9}" `)}</body></html>`);
    await p.screenshot({ path: out });
    await p.close();
  };
  for (const [k, w, h] of [[100, 164, 314], [150, 246, 471], [200, 328, 628]]) await wiz(w, h, `/tmp/wiz-${k}.png`);
  for (const [k, n] of [[100, 55], [150, 82], [200, 110]]) await shot(n, `/tmp/small-${k}.png`);
  await b.close();
})();
