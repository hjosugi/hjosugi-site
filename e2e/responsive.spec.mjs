import { test, expect } from "@playwright/test";
import { mkdir } from "node:fs/promises";
import { join } from "node:path";

// Pages of the exported static site to verify.
const PAGES = [
  { name: "about", path: "/", ready: "h1" },
  { name: "radar", path: "/radar/", ready: ".radar-card" },
];

const SHOT_DIR = join("e2e", "screenshots");

// Returns selectors of elements that stick out past the viewport width — the
// failure mode behind the original "radar isn't responsive" report.
async function overflowingElements(page) {
  return page.evaluate(() => {
    const docWidth = document.documentElement.clientWidth;
    const offenders = [];
    for (const node of document.body.querySelectorAll("*")) {
      const rect = node.getBoundingClientRect();
      // 1px tolerance for sub-pixel rounding.
      if (rect.width > 0 && rect.right > docWidth + 1) {
        const id = node.id ? `#${node.id}` : "";
        const cls = node.className && typeof node.className === "string"
          ? "." + node.className.trim().split(/\s+/).join(".")
          : "";
        offenders.push(`${node.tagName.toLowerCase()}${id}${cls} (right=${Math.round(rect.right)} > ${docWidth})`);
      }
    }
    return offenders.slice(0, 12);
  });
}

for (const pageDef of PAGES) {
  test(`${pageDef.name} has no horizontal overflow`, async ({ page }, testInfo) => {
    await page.goto(pageDef.path, { waitUntil: "networkidle" });
    await page.waitForSelector(pageDef.ready, { timeout: 10_000 });

    // The document must not scroll horizontally.
    const scrollWidth = await page.evaluate(() => document.documentElement.scrollWidth);
    const clientWidth = await page.evaluate(() => document.documentElement.clientWidth);
    const offenders = await overflowingElements(page);

    // Capture the rendered design for review (artifact + report attachment).
    await mkdir(join(SHOT_DIR, testInfo.project.name), { recursive: true });
    const shot = join(SHOT_DIR, testInfo.project.name, `${pageDef.name}.png`);
    await page.screenshot({ path: shot, fullPage: true });
    await testInfo.attach(`${pageDef.name}-${testInfo.project.name}`, {
      path: shot,
      contentType: "image/png",
    });

    expect(
      offenders,
      `elements overflow the viewport on ${testInfo.project.name}:\n${offenders.join("\n")}`,
    ).toEqual([]);
    expect(scrollWidth, "document scrolls horizontally").toBeLessThanOrEqual(clientWidth + 1);
  });
}

test("radar search input stays inside the viewport", async ({ page }) => {
  await page.goto("/radar/", { waitUntil: "networkidle" });
  const input = page.locator("#radar-search");
  await expect(input).toBeVisible();
  const box = await input.boundingBox();
  const clientWidth = await page.evaluate(() => document.documentElement.clientWidth);
  expect(box, "search input has a box").not.toBeNull();
  expect(box.x + box.width, "search input overflows").toBeLessThanOrEqual(clientWidth + 1);
});
