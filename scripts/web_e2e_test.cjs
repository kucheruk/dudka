#!/usr/bin/env node
"use strict";

const assert = require("node:assert/strict");
const { execFileSync, spawn } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { chromium } = require("playwright");

const repo = path.resolve(__dirname, "..");
const origin = "http://127.0.0.1:55251";
const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "dudka-web-e2e-"));
const serverBinary = path.join(tempDir, "dudka-signal");
execFileSync("go", ["build", "-o", serverBinary, "./cmd/dudka-signal"], {
  cwd: repo,
  stdio: "inherit",
});
const server = spawn(
  serverBinary,
  [
    "-listen",
    "127.0.0.1:55251",
    "-origin",
    origin,
    "-web-dir",
    "web",
  ],
  { cwd: repo, stdio: ["ignore", "pipe", "pipe"] },
);

let serverOutput = "";
server.stdout.on("data", (chunk) => { serverOutput += chunk; });
server.stderr.on("data", (chunk) => { serverOutput += chunk; });

async function waitForServer() {
  const deadline = Date.now() + 15000;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(`${origin}/health`);
      if (response.ok) return;
    } catch {
      // Go build or listener startup is still in progress.
    }
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  throw new Error(`dudka-signal did not start:\n${serverOutput}`);
}

function instrument(page) {
  return page.addInitScript(() => {
    const NativeWebSocket = window.WebSocket;
    const NativePeerConnection = window.RTCPeerConnection;
    // Safari may serialize a native RTCIceCandidate as an empty object.
    // The application must copy candidate fields explicitly.
    Object.defineProperty(window.RTCIceCandidate.prototype, "toJSON", {
      configurable: true,
      value: () => ({}),
    });
    window.__dudkaWebSockets = 0;
    window.__dudkaPeerConnections = 0;
    window.WebSocket = class extends NativeWebSocket {
      constructor(...args) {
        super(...args);
        window.__dudkaWebSockets += 1;
      }
    };
    window.RTCPeerConnection = class extends NativePeerConnection {
      constructor(...args) {
        super(...args);
        window.__dudkaPeerConnections += 1;
      }
    };
  });
}

async function main() {
  await waitForServer();
  const browser = await chromium.launch({
    headless: true,
    // Headless Chromium has no mDNS resolver. Real browsers resolve these
    // host candidates in LAN; the test exposes the same local ICE addresses.
    args: ["--disable-features=WebRtcHideLocalIpsWithMdns"],
  });
  try {
    const firstContext = await browser.newContext();
    const secondContext = await browser.newContext();
    const first = await firstContext.newPage();
    const second = await secondContext.newPage();
    await instrument(first);
    await instrument(second);

    await first.goto(origin);
    assert.equal(await first.evaluate(() => window.__dudkaWebSockets), 0);
    assert.equal(await first.evaluate(() => window.__dudkaPeerConnections), 0);
    await first.click("#consent-decline");
    assert.equal(await first.evaluate(() => window.__dudkaWebSockets), 0);

    await first.fill("#display-name", "Евгений");
    await first.click("#consent-accept");
    await second.goto(origin);
    assert.equal(await second.evaluate(() => window.__dudkaWebSockets), 0);
    await second.fill("#display-name", "Жена");
    await second.click("#consent-accept");

    const onlineTwo = () =>
      document.querySelector("#online-count").textContent === "ОНЛАЙН 2";
    await first.waitForFunction(onlineTwo, null, { timeout: 15000 });
    await second.waitForFunction(onlineTwo, null, { timeout: 15000 });

    await first.evaluate(() => restartSignaling("e2e resume"));
    await first.waitForFunction(() =>
      window.__dudkaWebSockets === 2 &&
      document.querySelector("#online-count").textContent === "ОНЛАЙН 2",
    null, { timeout: 15000 });
    await second.waitForFunction(onlineTwo, null, { timeout: 15000 });

    await first.fill("#message-input", "Привет через прямой WebRTC");
    await first.press("#message-input", "Meta+Enter");
    await second.waitForFunction(() =>
      document.querySelector("#feed").textContent.includes("Привет через прямой WebRTC"));
    await second.fill("#message-input", "Ответ напрямую");
    await second.press("#message-input", "Meta+Enter");
    await first.waitForFunction(() =>
      document.querySelector("#feed").textContent.includes("Ответ напрямую"));

    await first.setInputFiles("#file-input", {
      name: "проверка.gif",
      mimeType: "image/gif",
      buffer: Buffer.from("GIF89a"),
    });
    assert.equal(await second.locator(".file-message").count(), 0);
    assert.match(await first.locator("#draft-files").textContent(), /проверка\.gif/);
    await first.click(".send-control");
    await second.waitForFunction(() =>
      document.querySelector(".file-message a")?.getAttribute("download") === "проверка.gif");

    console.log("OK consent=offline peers=2 text=bidirectional file=проверка.gif safari-ice=json");
  } finally {
    await browser.close();
  }
}

main()
  .catch((error) => {
    console.error(error);
    process.exitCode = 1;
  })
  .finally(() => {
    server.kill("SIGTERM");
    fs.rmSync(tempDir, { recursive: true, force: true });
  });
