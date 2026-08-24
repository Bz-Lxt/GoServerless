import { expect, test } from "@playwright/test";

const web = process.env.WEB_BASE || "http://frontend";

test("login deploy invoke observe", async ({ page }) => {
  await page.goto(`${web}/login`);
  await expect(page.getByRole("heading", { name: "GoServerless" })).toBeVisible();
  await page.getByLabel("用户名 *").fill("admin");
  await page.getByLabel("密码 *").fill("admin123");
  await page.getByRole("button", { name: "进入控制台" }).click();
  await expect(page.getByRole("heading", { name: "总览" })).toBeVisible();
  await page.getByRole("button", { name: "新建函数" }).click();
  const name = `e2e${Date.now().toString().slice(-6)}`;
  await page.getByPlaceholder("hello-go").fill(name);
  await page.locator("select").first().selectOption("nodejs");
  await page.getByRole("button", { name: "创建" }).click();
  await expect(page.getByRole("heading", { name })).toBeVisible();
  await page.getByRole("button", { name: "部署" }).click();
  await expect(page.getByText("READY")).toBeVisible({ timeout: 30000 });
  await page.getByRole("button", { name: "试调用" }).click();
  await expect(page.getByText("调用次数")).toBeVisible();
});
