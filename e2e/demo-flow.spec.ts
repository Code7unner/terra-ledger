import { test, expect } from "@playwright/test";

test.describe("TerraLedger Demo Flow", () => {
  test("landing page loads with role selection and map", async ({ page }) => {
    await page.goto("/");

    // Hero title visible
    await expect(page.locator("h1")).toContainText("Agricultural Credit Intelligence");

    // Both role cards visible
    await expect(page.getByText("I'm a Farmer")).toBeVisible();
    await expect(page.getByText("I'm a Lender")).toBeVisible();

    // Map section renders
    await expect(page.getByText("Browse Registered Parcels")).toBeVisible();
  });

  test("lender flow: search parcel and view credit profile", async ({ page }) => {
    await page.goto("/");

    // Select lender role
    await page.getByText("I'm a Lender").click();

    // Search step should appear — fill cadastral number
    const searchInput = page.locator('input[placeholder*="cadastral" i], input[type="text"]').first();
    await expect(searchInput).toBeVisible({ timeout: 5000 });
    await searchInput.fill("KZ11-0032-001");

    // Submit search (click the button or press enter)
    const searchButton = page.getByRole("button", { name: /search|find|next/i });
    if (await searchButton.isVisible()) {
      await searchButton.click();
    } else {
      await searchInput.press("Enter");
    }

    // Should see profile data — wait for it to load
    await expect(page.getByText("KZ11-0032-001")).toBeVisible({ timeout: 10000 });
  });

  test("parcel detail page shows credit profile sections", async ({ page }) => {
    await page.goto("/parcel/KZ11-0032-001");

    // Wait for data to load
    await expect(page.getByText("Parcel KZ11-0032-001")).toBeVisible({ timeout: 10000 });

    // Parcel details section
    await expect(page.getByText("Parcel Details")).toBeVisible();

    // NDVI History section
    await expect(page.getByText("NDVI History")).toBeVisible();

    // Credit Score metric should render
    await expect(page.getByText("Credit Score")).toBeVisible();

    // Area metric
    await expect(page.getByText("Area (ha)")).toBeVisible();

    // Lien History section
    await expect(page.getByText("Lien History")).toBeVisible();
  });

  test("map page renders with navigation", async ({ page }) => {
    await page.goto("/map");

    // Map page title
    await expect(page.locator("h1")).toContainText("Parcel Map");

    // Legend visible
    await expect(page.getByText("Land Class")).toBeVisible();
    await expect(page.getByText("Class 1-2 (high)")).toBeVisible();

    // Leaflet map container present
    await expect(page.locator(".leaflet-container")).toBeVisible({ timeout: 5000 });
  });

  test("navigation between pages works", async ({ page }) => {
    // Start at landing
    await page.goto("/");
    await expect(page.getByText("Agricultural Credit Intelligence")).toBeVisible();

    // Navigate to map via TopBar link
    const mapLink = page.getByRole("link", { name: "Map" });
    if (await mapLink.isVisible()) {
      await mapLink.click();
      await expect(page.locator("h1")).toContainText("Parcel Map");
    }

    // Navigate back to home via logo
    await page.getByRole("link", { name: /terraledger/i }).click();
    await expect(page.getByText("Agricultural Credit Intelligence")).toBeVisible();
  });
});
