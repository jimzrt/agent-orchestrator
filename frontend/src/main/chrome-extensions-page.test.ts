import { describe, expect, it, vi } from "vitest";
import { chromeExtensionsPageCommands, openChromeExtensionsPage } from "./chrome-extensions-page";

function missing(): Error {
	return Object.assign(new Error("spawn ENOENT"), { code: "ENOENT" });
}

describe("chrome extensions page", () => {
	it("launches Chrome itself with the page on each platform", () => {
		expect(chromeExtensionsPageCommands("darwin")).toEqual([["open", ["-a", "Google Chrome", "chrome://extensions"]]]);
		expect(chromeExtensionsPageCommands("win32")).toEqual([["cmd", ["/c", "start", "", "chrome", "chrome://extensions"]]]);
		expect(chromeExtensionsPageCommands("linux")[0]).toEqual(["google-chrome", ["chrome://extensions"]]);
	});

	it("tries Chromium, Brave, Edge and Vivaldi launchers on Linux", () => {
		const files = chromeExtensionsPageCommands("linux").map(([file]) => file);
		expect(files).toEqual(
			expect.arrayContaining(["chromium", "brave-browser", "microsoft-edge-stable", "vivaldi-stable"]),
		);
	});

	it("falls through missing launchers to the installed browser", async () => {
		const run = vi.fn(async (file: string) => {
			if (file !== "chromium") throw missing();
		});
		await openChromeExtensionsPage("linux", run);
		expect(run.mock.calls.map(([file]) => file)).toEqual(["google-chrome", "google-chrome-stable", "chromium"]);
		expect(run).toHaveBeenLastCalledWith("chromium", ["chrome://extensions"]);
	});

	it("fails when no launcher is installed", async () => {
		const run = vi.fn(async () => {
			throw missing();
		});
		await expect(openChromeExtensionsPage("linux", run)).rejects.toThrow("ENOENT");
		expect(run).toHaveBeenCalledTimes(chromeExtensionsPageCommands("linux").length);
	});

	it("does not try another browser after one started and failed", async () => {
		const run = vi.fn(async () => {
			throw new Error("exit 1");
		});
		await expect(openChromeExtensionsPage("linux", run)).rejects.toThrow("exit 1");
		expect(run).toHaveBeenCalledTimes(1);
	});

	it("reports a failed launch", async () => {
		const run = vi.fn(async () => {
			throw new Error("Chrome is not installed");
		});
		await expect(openChromeExtensionsPage("darwin", run)).rejects.toThrow("Chrome is not installed");
	});
});
