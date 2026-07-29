"use strict";

const MAX_HISTORY = 200;
const MAX_TEXT = 4000;
const FILE_CHUNK = 16 * 1024;
const BUFFER_HIGH = 1024 * 1024;
const HISTORY_KEY = "dudka.web.history.v1";
const IDENTITY_KEY = "dudka.web.identity.v1";

const els = {
  consent: document.querySelector("#consent-screen"),
  accept: document.querySelector("#consent-accept"),
  decline: document.querySelector("#consent-decline"),
  consentNote: document.querySelector("#consent-note"),
  name: document.querySelector("#display-name"),
  app: document.querySelector("#chat-app"),
  connectionLed: document.querySelector("#connection-led"),
  connectionState: document.querySelector("#connection-state"),
  onlineCount: document.querySelector("#online-count"),
  peerList: document.querySelector("#peer-list"),
  feed: document.querySelector("#feed"),
  emptyFeed: document.querySelector("#empty-feed"),
  form: document.querySelector("#compose-form"),
  input: document.querySelector("#message-input"),
  fileInput: document.querySelector("#file-input"),
  drafts: document.querySelector("#draft-files"),
  status: document.querySelector("#status-message"),
  statusLine: document.querySelector(".status-line"),
  copyDiagnostic: document.querySelector("#copy-diagnostic"),
};

const identity = loadIdentity();
let signalQueue = Promise.resolve();
const state = {
  consented: false,
  displayName: "Дудка браузер",
  signalID: "",
  socket: null,
  peers: new Map(),
  messages: loadHistory(),
  messageIDs: new Set(),
  drafts: [],
  lastError: "",
  signalClose: "не было",
  reconnectAttempt: 0,
  reconnectTimer: null,
  startedAt: "",
};
for (const message of state.messages) state.messageIDs.add(message.id);

els.name.value = identity.lastName || "Дудка браузер";

els.accept.addEventListener("click", () => {
  const name = cleanName(els.name.value);
  if (!name) {
    els.consentNote.textContent = "Введите имя: его увидят только соседние вкладки.";
    els.name.focus();
    return;
  }
  state.consented = true;
  state.displayName = name;
  state.startedAt = new Date().toISOString();
  identity.lastName = name;
  localStorage.setItem(IDENTITY_KEY, JSON.stringify(identity));
  els.consent.hidden = true;
  els.app.hidden = false;
  renderMessages();
  renderPeers();
  connectSignaling();
  els.input.focus();
});

els.decline.addEventListener("click", () => {
  els.consentNote.textContent = "Сеть не запущена. Можно закрыть вкладку или разрешить подключение позже.";
  els.accept.focus();
});

els.form.addEventListener("submit", (event) => {
  event.preventDefault();
  sendDraft();
});

els.input.addEventListener("keydown", (event) => {
  if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
    event.preventDefault();
    sendDraft();
  }
});

els.fileInput.addEventListener("change", () => {
  for (const file of els.fileInput.files) addDraft(file);
  els.fileInput.value = "";
});

els.copyDiagnostic.addEventListener("click", async () => {
  const report = buildDiagnostic();
  try {
    await navigator.clipboard.writeText(report);
    setStatus("ДИАГНОСТИКА СКОПИРОВАНА");
  } catch {
    setStatus("НЕ УДАЛОСЬ СКОПИРОВАТЬ · выделите текст вручную", true);
  }
});

window.addEventListener("beforeunload", () => {
  for (const draft of state.drafts) URL.revokeObjectURL(draft.previewURL);
  if (state.socket) state.socket.close(1000, "вкладка закрыта");
  for (const peer of state.peers.values()) peer.pc.close();
});

function connectSignaling() {
  if (!state.consented || state.socket) return;
  setConnection("ЗНАКОМЛЮСЬ…");
  const scheme = location.protocol === "https:" ? "wss:" : "ws:";
  const socket = new WebSocket(`${scheme}//${location.host}/dudka/signal`);
  state.socket = socket;

  socket.addEventListener("open", () => {
    state.reconnectAttempt = 0;
    state.signalClose = "соединение открыто";
    setConnection("ИЩУ СВОИХ…", true);
  });
  socket.addEventListener("message", (event) => {
    signalQueue = signalQueue
      .then(() => handleSignal(JSON.parse(event.data)))
      .catch((error) => showError(`Ошибка signaling: ${error.message}`));
  });
  socket.addEventListener("close", (event) => {
    state.socket = null;
    state.signalClose = `code=${event.code} clean=${event.wasClean} reason=${event.reason || "нет"}`;
    setConnection("SIGNALING НЕДОСТУПЕН", false, true);
    if ([...state.peers.values()].some((peer) => peer.open)) {
      setStatus("ПРЯМОЙ ЧАТ РАБОТАЕТ · новые вкладки пока не найдутся");
    } else {
      scheduleReconnect();
    }
  });
  socket.addEventListener("error", () => {
    showError("Не удалось открыть signaling WebSocket.");
  });
}

function scheduleReconnect() {
  if (!state.consented || state.reconnectTimer) return;
  state.reconnectAttempt += 1;
  const delay = Math.min(10000, 1000 * (2 ** state.reconnectAttempt));
  showError(`Сигнальный сервис отключился. Повтор через ${delay / 1000} с.`);
  state.reconnectTimer = window.setTimeout(() => {
    state.reconnectTimer = null;
    connectSignaling();
  }, delay);
}

async function handleSignal(message) {
  try {
    switch (message.type) {
      case "welcome":
        state.signalID = message.from;
        for (const peerID of message.peers || []) {
          const peer = createPeer(peerID, true);
          await makeOffer(peer);
        }
        setStatus(message.peers?.length ? "СОГЛАСОВЫВАЮ ПРЯМОЙ КАНАЛ…" : "ЖДУ СОСЕДНЮЮ ВКЛАДКУ");
        break;
      case "offer": {
        const peer = createPeer(message.from, false);
        await peer.pc.setRemoteDescription(message.description);
        const answer = await peer.pc.createAnswer();
        await peer.pc.setLocalDescription(answer);
        sendSignal({ type: "answer", to: message.from, description: peer.pc.localDescription });
        break;
      }
      case "answer": {
        const peer = state.peers.get(message.from);
        if (peer) await peer.pc.setRemoteDescription(message.description);
        break;
      }
      case "ice": {
        const peer = state.peers.get(message.from);
        if (peer && message.candidate) await peer.pc.addIceCandidate(message.candidate);
        break;
      }
      default:
        throw new Error(`неизвестный signaling type: ${message.type}`);
    }
  } catch (error) {
    showError(`WebRTC не согласован: ${error.message}`);
  }
}

function createPeer(peerID, initiator) {
  if (state.peers.has(peerID)) return state.peers.get(peerID);
  const pc = new RTCPeerConnection({
    iceServers: [{ urls: "stun:zamoo.team:3478" }],
  });
  const peer = { id: peerID, pc, channel: null, open: false, name: "Соседняя вкладка" };
  state.peers.set(peerID, peer);

  pc.addEventListener("icecandidate", (event) => {
    if (event.candidate) sendSignal({ type: "ice", to: peerID, candidate: event.candidate });
  });
  pc.addEventListener("connectionstatechange", () => {
    if (["failed", "closed", "disconnected"].includes(pc.connectionState)) removePeer(peerID);
  });
  pc.addEventListener("datachannel", (event) => {
    if (event.channel.label === "dudka-chat") wireChatChannel(peer, event.channel);
    if (event.channel.label.startsWith("dudka-file:")) receiveFileChannel(peer, event.channel);
  });

  if (initiator) wireChatChannel(peer, pc.createDataChannel("dudka-chat", { ordered: true }));
  renderPeers();
  return peer;
}

async function makeOffer(peer) {
  const offer = await peer.pc.createOffer();
  await peer.pc.setLocalDescription(offer);
  sendSignal({ type: "offer", to: peer.id, description: peer.pc.localDescription });
}

function sendSignal(message) {
  if (!state.socket || state.socket.readyState !== WebSocket.OPEN) {
    throw new Error("signaling WebSocket закрыт");
  }
  state.socket.send(JSON.stringify(message));
}

function wireChatChannel(peer, channel) {
  peer.channel = channel;
  channel.addEventListener("open", () => {
    peer.open = true;
    sendPeerPacket(peer, {
      type: "hello",
      peerID: identity.peerID,
      name: state.displayName,
    });
    const tail = state.messages.slice(-MAX_HISTORY);
    for (let index = 0; index < tail.length; index += 10) {
      sendPeerPacket(peer, { type: "tail", messages: tail.slice(index, index + 10) });
    }
    setConnection("ПРЯМОЙ КАНАЛ", true);
    setStatus("СОЕДИНЕНО НАПРЯМУЮ · signaling не передаёт чат");
    renderPeers();
  });
  channel.addEventListener("message", (event) => handlePeerPacket(peer, JSON.parse(event.data)));
  channel.addEventListener("close", () => removePeer(peer.id));
  channel.addEventListener("error", () => showError(`Прямой канал с ${peer.name} повреждён.`));
}

function handlePeerPacket(peer, packet) {
  switch (packet.type) {
    case "hello":
      peer.name = cleanName(packet.name) || "Соседняя вкладка";
      peer.remotePeerID = String(packet.peerID || "");
      renderPeers();
      break;
    case "chat":
      acceptMessage(packet.message);
      break;
    case "tail":
      for (const message of packet.messages || []) acceptMessage(message, false);
      saveHistory();
      renderMessages();
      break;
    default:
      showError(`Сосед прислал неизвестный пакет: ${packet.type}`);
  }
}

function sendDraft() {
  const text = els.input.value.trim();
  if (!text && state.drafts.length === 0) return;
  if (text.length > MAX_TEXT) {
    showError(`Сообщение длиннее ${MAX_TEXT} знаков.`);
    return;
  }

  if (text) {
    const message = {
      id: crypto.randomUUID(),
      senderID: identity.peerID,
      sender: state.displayName,
      text,
      sentAt: new Date().toISOString(),
    };
    acceptMessage(message);
    broadcast({ type: "chat", message });
  }

  const drafts = state.drafts;
  state.drafts = [];
  els.input.value = "";
  renderDrafts();
  for (const draft of drafts) sendFile(draft.file, draft.previewURL);
  setStatus(openPeerCount() ? "ОТПРАВЛЕНО В ПРЯМОЙ КАНАЛ" : "РЯДОМ НИКОГО · сообщение осталось в этой вкладке");
}

function acceptMessage(message, persist = true) {
  if (!validChatMessage(message) || state.messageIDs.has(message.id)) return;
  state.messageIDs.add(message.id);
  state.messages.push(message);
  state.messages.sort((a, b) => a.sentAt.localeCompare(b.sentAt));
  state.messages = state.messages.slice(-MAX_HISTORY);
  if (persist) saveHistory();
  renderMessages();
}

function validChatMessage(message) {
  return message &&
    typeof message.id === "string" &&
    typeof message.senderID === "string" &&
    typeof message.sender === "string" &&
    typeof message.text === "string" &&
    message.text.length <= MAX_TEXT &&
    typeof message.sentAt === "string";
}

function broadcast(packet) {
  for (const peer of state.peers.values()) sendPeerPacket(peer, packet);
}

function sendPeerPacket(peer, packet) {
  if (peer.open && peer.channel?.readyState === "open") {
    peer.channel.send(JSON.stringify(packet));
  }
}

function addDraft(file) {
  const previewURL = file.type.startsWith("image/") ? URL.createObjectURL(file) : "";
  state.drafts.push({ id: crypto.randomUUID(), file, previewURL });
  renderDrafts();
}

function renderDrafts() {
  els.drafts.replaceChildren();
  for (const draft of state.drafts) {
    const row = document.createElement("article");
    row.className = "draft-file";

    const preview = draft.previewURL ? document.createElement("img") : document.createElement("span");
    if (draft.previewURL) {
      preview.src = draft.previewURL;
      preview.alt = "";
    } else {
      preview.className = "draft-placeholder";
      preview.textContent = "ФАЙЛ";
    }

    const info = document.createElement("div");
    info.className = "draft-file-info";
    const name = document.createElement("strong");
    name.textContent = draft.file.name;
    const size = document.createElement("span");
    size.textContent = formatBytes(draft.file.size);
    info.append(name, size);

    const remove = document.createElement("button");
    remove.type = "button";
    remove.setAttribute("aria-label", `Убрать ${draft.file.name}`);
    remove.textContent = "×";
    remove.addEventListener("click", () => {
      if (draft.previewURL) URL.revokeObjectURL(draft.previewURL);
      state.drafts = state.drafts.filter((item) => item.id !== draft.id);
      renderDrafts();
    });
    row.append(preview, info, remove);
    els.drafts.append(row);
  }
}

function sendFile(file, previewURL) {
  const fileID = crypto.randomUUID();
  renderFileMessage({
    id: fileID,
    sender: state.displayName,
    name: file.name,
    size: file.size,
    mime: file.type || "application/octet-stream",
    sentAt: new Date().toISOString(),
    url: previewURL || URL.createObjectURL(file),
  });
  for (const peer of state.peers.values()) {
    if (peer.open) {
      sendFileToPeer(peer, fileID, file).catch((error) => {
        showError(`Не удалось отправить ${file.name}: ${error.message}`);
      });
    }
  }
}

async function sendFileToPeer(peer, fileID, file) {
  const channel = peer.pc.createDataChannel(`dudka-file:${fileID}`, { ordered: true });
  channel.binaryType = "arraybuffer";
  channel.bufferedAmountLowThreshold = BUFFER_HIGH / 2;
  await waitForOpen(channel);
  channel.send(JSON.stringify({
    type: "meta",
    id: fileID,
    sender: state.displayName,
    name: file.name,
    size: file.size,
    mime: file.type || "application/octet-stream",
    sentAt: new Date().toISOString(),
  }));
  for (let offset = 0; offset < file.size; offset += FILE_CHUNK) {
    await waitForBuffer(channel);
    channel.send(await file.slice(offset, offset + FILE_CHUNK).arrayBuffer());
  }
  channel.send(JSON.stringify({ type: "done" }));
}

function receiveFileChannel(peer, channel) {
  channel.binaryType = "arraybuffer";
  let meta = null;
  let received = 0;
  const chunks = [];
  channel.addEventListener("message", (event) => {
    if (typeof event.data === "string") {
      const packet = JSON.parse(event.data);
      if (packet.type === "meta") {
        if (!validFileMeta(packet)) {
          channel.close();
          showError(`Файл от ${peer.name} имеет неверное описание.`);
          return;
        }
        meta = packet;
        setStatus(`ПРИНИМАЮ ${packet.name} · 0%`);
      } else if (packet.type === "done" && meta) {
        if (received !== meta.size) {
          channel.close();
          showError(`Файл от ${peer.name} принят не полностью.`);
          return;
        }
        const blob = new Blob(chunks, { type: meta.mime });
        renderFileMessage({ ...meta, url: URL.createObjectURL(blob) });
        setStatus(`ФАЙЛ ПОЛУЧЕН · ${meta.name}`);
        channel.close();
      }
      return;
    }
    if (!meta || received + event.data.byteLength > meta.size) {
      channel.close();
      showError(`Файл от ${peer.name} имеет неверный размер.`);
      return;
    }
    chunks.push(event.data);
    received += event.data.byteLength;
    const percent = meta.size ? Math.floor((received / meta.size) * 100) : 100;
    setStatus(`ПРИНИМАЮ ${meta.name} · ${percent}%`);
  });
}

function validFileMeta(meta) {
  return meta &&
    typeof meta.id === "string" &&
    typeof meta.sender === "string" &&
    meta.sender.length <= 80 &&
    typeof meta.name === "string" &&
    meta.name.length > 0 &&
    meta.name.length <= 240 &&
    Number.isSafeInteger(meta.size) &&
    meta.size >= 0 &&
    typeof meta.mime === "string" &&
    meta.mime.length <= 200 &&
    typeof meta.sentAt === "string";
}

function renderMessages() {
  els.feed.querySelectorAll(".message").forEach((node) => node.remove());
  els.emptyFeed.hidden = state.messages.length > 0;
  for (const message of state.messages) {
    const row = document.createElement("article");
    row.className = "message";
    const meta = document.createElement("time");
    meta.className = "message-meta";
    meta.dateTime = message.sentAt;
    meta.textContent = formatTime(message.sentAt);
    const body = document.createElement("p");
    body.className = "message-body";
    const sender = document.createElement("span");
    sender.className = "message-sender";
    sender.textContent = message.sender;
    body.append(sender, document.createTextNode(`  ${message.text}`));
    row.append(meta, body);
    els.feed.append(row);
  }
  els.feed.scrollTop = els.feed.scrollHeight;
}

function renderFileMessage(file) {
  els.emptyFeed.hidden = true;
  const row = document.createElement("article");
  row.className = "message";
  const meta = document.createElement("time");
  meta.className = "message-meta";
  meta.dateTime = file.sentAt;
  meta.textContent = formatTime(file.sentAt);
  const content = document.createElement("div");
  content.className = "file-message";
  const sender = document.createElement("span");
  sender.className = "message-sender";
  sender.textContent = file.sender;
  content.append(sender);
  if (file.mime.startsWith("image/")) {
    const image = document.createElement("img");
    image.src = file.url;
    image.alt = file.name;
    content.append(image);
  }
  const link = document.createElement("a");
  link.href = file.url;
  link.download = safeFilename(file.name);
  link.textContent = `СКАЧАТЬ ${file.name} · ${formatBytes(file.size)}`;
  content.append(link);
  row.append(meta, content);
  els.feed.append(row);
  els.feed.scrollTop = els.feed.scrollHeight;
}

function renderPeers() {
  els.peerList.replaceChildren();
  const self = document.createElement("li");
  self.className = "self";
  self.textContent = `${state.displayName} · ВЫ`;
  els.peerList.append(self);
  for (const peer of state.peers.values()) {
    if (!peer.open) continue;
    const row = document.createElement("li");
    row.textContent = peer.name;
    els.peerList.append(row);
  }
  els.onlineCount.textContent = `ОНЛАЙН ${openPeerCount() + 1}`;
}

function removePeer(peerID) {
  const peer = state.peers.get(peerID);
  if (!peer) return;
  peer.open = false;
  peer.pc.close();
  state.peers.delete(peerID);
  renderPeers();
  if (!openPeerCount()) setConnection("ЖДУ СОСЕДЕЙ…", true);
}

function openPeerCount() {
  return [...state.peers.values()].filter((peer) => peer.open).length;
}

function setConnection(text, active = false, error = false) {
  els.connectionState.textContent = text;
  els.connectionLed.className = `led${active ? " active" : ""}${error ? " error" : ""}`;
}

function setStatus(text, error = false) {
  els.status.textContent = text;
  els.statusLine.classList.toggle("error", error);
  if (!error) {
    els.copyDiagnostic.hidden = true;
    state.lastError = "";
  }
}

function showError(text) {
  state.lastError = text;
  els.status.textContent = text;
  els.statusLine.classList.add("error");
  els.copyDiagnostic.hidden = false;
}

function buildDiagnostic() {
  return [
    "ДУДКА WEB — ДИАГНОСТИКА ДЛЯ АГЕНТА",
    `собрано: ${new Date().toISOString()}`,
    `запущено: ${state.startedAt || "нет"}`,
    `страница: ${location.origin}${location.pathname}`,
    `online: ${openPeerCount() + 1}`,
    `signaling state: ${state.socket?.readyState ?? "закрыт"}`,
    `signaling close: ${state.signalClose}`,
    `signaling reconnect: ${state.reconnectAttempt}`,
    `webrtc: ${[...state.peers.values()].map((peer) =>
      `${peer.pc.connectionState}/${peer.pc.iceConnectionState}/${peer.pc.iceGatheringState}/${peer.pc.signalingState}`).join(",") || "нет"}`,
    `браузер: ${navigator.userAgent}`,
    "",
    "ОШИБКА",
    state.lastError || "(нет)",
  ].join("\n");
}

function loadIdentity() {
  try {
    const stored = JSON.parse(localStorage.getItem(IDENTITY_KEY));
    if (stored?.peerID) return stored;
  } catch {
    // Создаём новую локальную личность.
  }
  return { peerID: crypto.randomUUID(), lastName: "" };
}

function loadHistory() {
  try {
    const stored = JSON.parse(localStorage.getItem(HISTORY_KEY));
    return Array.isArray(stored) ? stored.filter(validChatMessage).slice(-MAX_HISTORY) : [];
  } catch {
    return [];
  }
}

function saveHistory() {
  localStorage.setItem(HISTORY_KEY, JSON.stringify(state.messages.slice(-MAX_HISTORY)));
}

function cleanName(value) {
  return String(value || "").trim().replace(/\s+/g, " ").slice(0, 80);
}

function safeFilename(value) {
  return String(value || "file").replace(/[\\/\0-\x1f\x7f]/g, "_").slice(0, 240) || "file";
}

function formatTime(value) {
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? "—" : date.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
}

function formatBytes(bytes) {
  if (bytes < 1024) return `${bytes} Б`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} КиБ`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} МиБ`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} ГиБ`;
}

function waitForOpen(channel) {
  if (channel.readyState === "open") return Promise.resolve();
  return new Promise((resolve, reject) => {
    channel.addEventListener("open", resolve, { once: true });
    channel.addEventListener("error", () => reject(new Error("file channel failed")), { once: true });
  });
}

function waitForBuffer(channel) {
  if (channel.bufferedAmount <= BUFFER_HIGH) return Promise.resolve();
  return new Promise((resolve) => {
    channel.addEventListener("bufferedamountlow", resolve, { once: true });
  });
}
