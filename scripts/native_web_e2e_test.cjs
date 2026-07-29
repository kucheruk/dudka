#!/usr/bin/env node
"use strict";

const assert = require("node:assert/strict");
const { execFileSync, spawn } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { chromium } = require("playwright");

const repo = path.resolve(__dirname, "..");
const origin = "http://127.0.0.1:55252";
const engineOrigin = "http://127.0.0.1:17889";
const temp = fs.mkdtempSync(path.join(os.tmpdir(), "dudka-native-web-"));
const signalBin = path.join(temp, "dudka-signal");
const engineBin = path.join(temp, "dudkad");
execFileSync("go", ["build", "-o", signalBin, "./cmd/dudka-signal"], { cwd: repo });
execFileSync("go", ["build", "-o", engineBin, "./cmd/dudkad"], { cwd: repo });

const signal = spawn(signalBin, [
  "-listen", "127.0.0.1:55252", "-origin", origin, "-web-dir", "web",
], { cwd: repo });
const engine = spawn(engineBin, [
  "-listen", "127.0.0.1:17889",
  "-data-dir", path.join(temp, "engine"),
  "-name", "Приложение",
  "-signal-url", "ws://127.0.0.1:55252/dudka/signal",
  "-signal-origin", origin,
  "-stun-url", "stun:127.0.0.1:9",
], { cwd: repo });
let signalLog = "";
let engineLog = "";
signal.stdout.on("data", (chunk) => { signalLog += chunk; });
signal.stderr.on("data", (chunk) => { signalLog += chunk; });
engine.stdout.on("data", (chunk) => { engineLog += chunk; });
engine.stderr.on("data", (chunk) => { engineLog += chunk; });

async function waitFor(url, condition = (response) => response.ok, timeout = 15000) {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(url);
      if (await condition(response)) return response;
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
  throw new Error(`timeout: ${url}`);
}

async function main() {
  await waitFor(`${origin}/health`);
  await waitFor(`${engineOrigin}/health`);
  let response = await fetch(`${engineOrigin}/internet-consent`, { method: "POST" });
  assert(response.ok, await response.text());

  const browser = await chromium.launch({
    headless: true,
    args: ["--disable-features=WebRtcHideLocalIpsWithMdns"],
  });
  try {
    const page = await browser.newPage();
    await page.goto(origin);
    await page.fill("#display-name", "Браузер");
    await page.click("#consent-accept");
    try {
      await page.waitForFunction(
        () => document.querySelector("#online-count").textContent === "ОНЛАЙН 2",
        null,
        { timeout: 15000 },
      );
    } catch (error) {
      const diagnostic = await page.evaluate(() => buildDiagnostic());
      throw new Error(
        `${error.message}\nBROWSER\n${diagnostic}\nENGINE\n${engineLog}\nSIGNAL\n${signalLog}`,
      );
    }
    await waitFor(`${engineOrigin}/peers`, async (peerResponse) => {
      const payload = await peerResponse.json();
      return payload.peers?.some((peer) => peer.display_name === "Браузер");
    });

    response = await fetch(`${engineOrigin}/send`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ text: "из приложения" }),
    });
    assert(response.ok, await response.text());
    await page.waitForFunction(
      () => document.querySelector("#feed").textContent.includes("из приложения"),
    );

    await page.fill("#message-input", "из браузера");
    await page.press("#message-input", "Meta+Enter");
    await waitFor(`${engineOrigin}/messages`, async (messageResponse) => {
      const payload = await messageResponse.json();
      return payload.messages?.some((message) => message.text === "из браузера");
    });

    response = await fetch(`${engineOrigin}/files/announce`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        name: "native.gif",
        size: 6,
        mime: "image/gif",
        hash: "sha256:610f5ae4d76e332636a17bd357fd6ce99029316a99d320280d4d77a746bf29e8",
        content_b64: Buffer.from("GIF89a").toString("base64"),
      }),
    });
    assert(response.ok, await response.text());
    await page.waitForFunction(
      () => document.querySelector(".file-message a")?.download === "native.gif",
    );

    await page.setInputFiles("#file-input", {
      name: "browser.txt", mimeType: "text/plain", buffer: Buffer.from("hello"),
    });
    await page.click(".send-control");
    await waitFor(`${engineOrigin}/messages`, async (messageResponse) => {
      const payload = await messageResponse.json();
      return payload.messages?.some((message) => message.name === "browser.txt");
    });
    console.log("native_web_e2e OK peers=2 text=bidirectional files=bidirectional");
  } finally {
    await browser.close();
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
}).finally(() => {
  signal.kill("SIGTERM");
  engine.kill("SIGTERM");
  fs.rmSync(temp, { recursive: true, force: true });
});
