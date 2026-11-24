// chrome:// URLs have no OS handler, so shell.openExternal cannot open them.
// A Chrome-based browser accepts one when it is launched with the URL as an
// argument (Edge rewrites it to edge://extensions).
const EXTENSIONS_PAGE = "chrome://extensions";

export type ChromeCommandRunner = (file: string, args: string[]) => Promise<unknown>;

// Linux has no single launcher: try the Chrome-based browsers the native host
// is registered for, in order, and use the first one that is installed.
const LINUX_BROWSER_COMMANDS = [
	"google-chrome",
	"google-chrome-stable",
	"chromium",
	"chromium-browser",
	"brave-browser",
	"brave",
	"microsoft-edge-stable",
	"microsoft-edge",
	"vivaldi-stable",
	"vivaldi",
];

export function chromeExtensionsPageCommands(platform: NodeJS.Platform): [string, string[]][] {
	if (platform === "darwin") return [["open", ["-a", "Google Chrome", EXTENSIONS_PAGE]]];
	if (platform === "win32") return [["cmd", ["/c", "start", "", "chrome", EXTENSIONS_PAGE]]];
	return LINUX_BROWSER_COMMANDS.map((command) => [command, [EXTENSIONS_PAGE]]);
}

function isMissingCommand(error: unknown): boolean {
	return (error as NodeJS.ErrnoException | undefined)?.code === "ENOENT";
}

export async function openChromeExtensionsPage(platform: NodeJS.Platform, run: ChromeCommandRunner): Promise<void> {
	const commands = chromeExtensionsPageCommands(platform);
	for (const [index, [file, args]] of commands.entries()) {
		try {
			await run(file, args);
			return;
		} catch (error) {
			// Only a missing launcher moves on; a browser that started and failed is the answer.
			if (!isMissingCommand(error) || index === commands.length - 1) throw error;
		}
	}
}
