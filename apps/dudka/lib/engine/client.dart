import 'dart:convert';
import 'dart:typed_data';

import 'package:crypto/crypto.dart';
import 'package:http/http.dart' as http;

/// Thin loopback client for dudkad (P060–P067 / docs/design/flutter-bind.md).
class EngineClient {
  EngineClient({
    required this.baseUrl,
    http.Client? httpClient,
  }) : _http = httpClient ?? http.Client();

  /// e.g. http://127.0.0.1:17880
  final String baseUrl;
  final http.Client _http;

  Uri _uri(String path) {
    final base = baseUrl.endsWith('/') ? baseUrl.substring(0, baseUrl.length - 1) : baseUrl;
    return Uri.parse('$base$path');
  }

  /// GET /me → peer identity.
  Future<MeInfo> fetchMe() async {
    final res = await _http.get(_uri('/me'));
    if (res.statusCode != 200) {
      throw EngineException('GET /me → ${res.statusCode}: ${res.body}');
    }
    return _parseMe(res.body);
  }

  /// POST /nick — set display name (P016 / P062 first-run).
  Future<MeInfo> setNick(String name) async {
    final n = name.trim();
    if (n.isEmpty) {
      throw EngineException('nick is required');
    }
    final res = await _http.post(
      _uri('/nick'),
      headers: {'content-type': 'application/json; charset=utf-8'},
      body: jsonEncode({'name': n}),
    );
    if (res.statusCode < 200 || res.statusCode > 299) {
      throw EngineException('POST /nick → ${res.statusCode}: ${res.body}');
    }
    return _parseMe(res.body, fallbackName: n);
  }

  /// POST /send — publish chat text (P064 / DUD-CHAT-100).
  Future<SendResult> sendText(String text) async {
    final t = text.trim();
    if (t.isEmpty) {
      throw EngineException('text is required');
    }
    final res = await _http.post(
      _uri('/send'),
      headers: {'content-type': 'application/json; charset=utf-8'},
      body: jsonEncode({'text': t}),
    );
    if (res.statusCode < 200 || res.statusCode > 299) {
      throw EngineException('POST /send → ${res.statusCode}: ${res.body}');
    }
    final map = jsonDecode(res.body) as Map<String, dynamic>;
    final status = (map['status'] as String?)?.trim() ?? '';
    final msgRaw = map['message'];
    ChatMessage? msg;
    if (msgRaw is Map) {
      msg = ChatMessage.fromJson(Map<String, dynamic>.from(msgRaw));
    }
    return SendResult(
      status: status.isEmpty ? 'accepted' : status,
      queued: (map['queued'] as num?)?.toInt() ?? 0,
      message: msg,
      text: msg?.text ?? t,
    );
  }

  /// POST /scan — subnet/host probe when alone (P065 / DUD-UI-120).
  Future<ScanResult> startScan({List<String>? hosts, int? port}) async {
    final payload = <String, Object?>{};
    if (hosts != null && hosts.isNotEmpty) payload['hosts'] = hosts;
    if (port != null && port > 0) payload['port'] = port;
    final res = await _http.post(
      _uri('/scan'),
      headers: {'content-type': 'application/json; charset=utf-8'},
      body: jsonEncode(payload),
    );
    if (res.statusCode < 200 || res.statusCode > 299) {
      throw EngineException('POST /scan → ${res.statusCode}: ${res.body}');
    }
    final map = jsonDecode(res.body) as Map<String, dynamic>;
    final peersRaw = map['peers'];
    final peers = <PeerInfo>[];
    if (peersRaw is List) {
      for (final item in peersRaw) {
        if (item is! Map) continue;
        final m = Map<String, dynamic>.from(item);
        final id = (m['peer_id'] as String?)?.trim() ?? '';
        final name = (m['display_name'] as String?)?.trim() ?? '';
        if (id.isEmpty && name.isEmpty) continue;
        peers.add(PeerInfo(peerId: id, displayName: name.isEmpty ? id : name));
      }
    }
    return ScanResult(
      probed: (map['probed'] as num?)?.toInt() ?? 0,
      found: (map['found'] as num?)?.toInt() ?? peers.length,
      peers: peers,
    );
  }

  /// POST /files/announce — publish file into feed (P067 / DUD-FILE-101).
  Future<ChatMessage> announceFile({
    required String name,
    required String mime,
    required List<int> content,
    String? hash,
  }) async {
    final n = name.trim();
    final m = mime.trim();
    if (n.isEmpty) throw EngineException('file name required');
    if (m.isEmpty) throw EngineException('file mime required');
    final digest = hash?.trim().isNotEmpty == true
        ? hash!.trim()
        : 'sha256:${sha256.convert(content)}';
    final res = await _http.post(
      _uri('/files/announce'),
      headers: {'content-type': 'application/json; charset=utf-8'},
      body: jsonEncode({
        'name': n,
        'size': content.length,
        'mime': m,
        'hash': digest,
        'content_b64': base64Encode(content),
      }),
    );
    if (res.statusCode < 200 || res.statusCode > 299) {
      throw EngineException('POST /files/announce → ${res.statusCode}: ${res.body}');
    }
    final map = jsonDecode(res.body) as Map<String, dynamic>;
    final msgRaw = map['message'];
    if (msgRaw is! Map) {
      throw EngineException('POST /files/announce missing message');
    }
    return ChatMessage.fromJson(Map<String, dynamic>.from(msgRaw));
  }

  /// POST /files/fetch with wait:false — start async download (P067 / P052).
  Future<TransferInfo> startFetch(String fileId) async {
    final id = fileId.trim();
    if (id.isEmpty) throw EngineException('file_id required');
    final res = await _http.post(
      _uri('/files/fetch'),
      headers: {'content-type': 'application/json; charset=utf-8'},
      body: jsonEncode({'file_id': id, 'wait': false}),
    );
    if (res.statusCode < 200 || res.statusCode > 299) {
      throw EngineException('POST /files/fetch → ${res.statusCode}: ${res.body}');
    }
    return TransferInfo.fromJson(jsonDecode(res.body) as Map<String, dynamic>);
  }

  /// POST /files/cancel (P067 / P053).
  Future<TransferInfo> cancelFetch(String fileId) async {
    final id = fileId.trim();
    if (id.isEmpty) throw EngineException('file_id required');
    final res = await _http.post(
      _uri('/files/cancel'),
      headers: {'content-type': 'application/json; charset=utf-8'},
      body: jsonEncode({'file_id': id}),
    );
    if (res.statusCode < 200 || res.statusCode > 299) {
      throw EngineException('POST /files/cancel → ${res.statusCode}: ${res.body}');
    }
    return TransferInfo.fromJson(jsonDecode(res.body) as Map<String, dynamic>);
  }

  /// GET /files/transfers (P067 / P052).
  Future<List<TransferInfo>> fetchTransfers() async {
    final res = await _http.get(_uri('/files/transfers'));
    if (res.statusCode != 200) {
      throw EngineException('GET /files/transfers → ${res.statusCode}: ${res.body}');
    }
    final map = jsonDecode(res.body) as Map<String, dynamic>;
    final raw = map['transfers'];
    final out = <TransferInfo>[];
    if (raw is List) {
      for (final item in raw) {
        if (item is! Map) continue;
        out.add(TransferInfo.fromJson(Map<String, dynamic>.from(item)));
      }
    }
    return out;
  }

  /// One frame: /me + /peers + /status + /messages + /files/transfers (P063/P067).
  Future<ChatSnapshot> fetchSnapshot() async {
    final me = await fetchMe();

    final peersRes = await _http.get(_uri('/peers'));
    if (peersRes.statusCode != 200) {
      throw EngineException('GET /peers → ${peersRes.statusCode}: ${peersRes.body}');
    }
    final peersMap = jsonDecode(peersRes.body) as Map<String, dynamic>;
    final peersRaw = peersMap['peers'];
    final peers = <PeerInfo>[];
    if (peersRaw is List) {
      for (final item in peersRaw) {
        if (item is! Map) continue;
        final m = Map<String, dynamic>.from(item);
        final id = (m['peer_id'] as String?)?.trim() ?? '';
        final name = (m['display_name'] as String?)?.trim() ?? '';
        if (id.isEmpty && name.isEmpty) continue;
        peers.add(PeerInfo(peerId: id, displayName: name.isEmpty ? id : name));
      }
    }

    final statusRes = await _http.get(_uri('/status'));
    if (statusRes.statusCode != 200) {
      throw EngineException('GET /status → ${statusRes.statusCode}: ${statusRes.body}');
    }
    final st = jsonDecode(statusRes.body) as Map<String, dynamic>;
    final network = (st['network'] as String?)?.trim();
    final protoMajor = (st['proto_major'] as num?)?.toInt() ?? 0;
    final protoMinor = (st['proto_minor'] as num?)?.toInt() ?? 0;

    final msgsRes = await _http.get(_uri('/messages'));
    if (msgsRes.statusCode != 200) {
      throw EngineException('GET /messages → ${msgsRes.statusCode}: ${msgsRes.body}');
    }
    final msgsMap = jsonDecode(msgsRes.body) as Map<String, dynamic>;
    final msgsRaw = msgsMap['messages'];
    final messages = <ChatMessage>[];
    if (msgsRaw is List) {
      for (final item in msgsRaw) {
        if (item is! Map) continue;
        final m = Map<String, dynamic>.from(item);
        messages.add(ChatMessage.fromJson(m));
      }
    }

    final transfers = await fetchTransfers();

    return ChatSnapshot(
      me: me,
      peers: peers,
      network: (network == null || network.isEmpty) ? 'ok' : network,
      protoMajor: protoMajor,
      protoMinor: protoMinor,
      messages: messages,
      transfers: transfers,
    );
  }

  MeInfo _parseMe(String body, {String? fallbackName}) {
    final map = jsonDecode(body) as Map<String, dynamic>;
    final peerId = (map['peer_id'] as String?)?.trim() ?? '';
    final name = (map['name'] as String?)?.trim() ?? '';
    if (peerId.isEmpty) {
      throw EngineException('GET /me missing peer_id');
    }
    final out = name.isEmpty ? (fallbackName ?? '—') : name;
    return MeInfo(peerId: peerId, name: out.isEmpty ? '—' : out);
  }

  void close() => _http.close();
}

class MeInfo {
  const MeInfo({required this.peerId, required this.name});

  final String peerId;
  final String name;
}

class PeerInfo {
  const PeerInfo({required this.peerId, required this.displayName});

  final String peerId;
  final String displayName;
}

class ChatMessage {
  const ChatMessage({
    required this.msgId,
    required this.peerId,
    required this.displayNameAtSend,
    required this.ts,
    required this.text,
    required this.type,
    this.fileId = '',
    this.fileName = '',
    this.size = 0,
    this.mime = '',
    this.hash = '',
    this.thumbB64 = '',
    this.thumbPath = '',
  });

  final String msgId;
  final String peerId;
  final String displayNameAtSend;
  final DateTime ts;
  final String text;
  final String type;
  final String fileId;
  final String fileName;
  final int size;
  final String mime;
  final String hash;
  final String thumbB64;
  final String thumbPath;

  bool get isFileAnnounce =>
      type == 'file_announce' || (fileId.isNotEmpty && fileName.isNotEmpty);

  factory ChatMessage.fromJson(Map<String, dynamic> m) {
    final tsRaw = m['ts'];
    DateTime ts;
    if (tsRaw is String && tsRaw.isNotEmpty) {
      ts = DateTime.tryParse(tsRaw)?.toUtc() ?? DateTime.fromMillisecondsSinceEpoch(0, isUtc: true);
    } else {
      ts = DateTime.fromMillisecondsSinceEpoch(0, isUtc: true);
    }
    final typ = (m['type'] as String?)?.trim();
    return ChatMessage(
      msgId: (m['msg_id'] as String?)?.trim() ?? '',
      peerId: (m['peer_id'] as String?)?.trim() ?? '',
      displayNameAtSend: (m['display_name_at_send'] as String?)?.trim() ?? '',
      ts: ts,
      text: (m['text'] as String?)?.trim() ?? '',
      type: (typ == null || typ.isEmpty) ? 'chat' : typ,
      fileId: (m['file_id'] as String?)?.trim() ?? '',
      fileName: (m['name'] as String?)?.trim() ?? '',
      size: (m['size'] as num?)?.toInt() ?? 0,
      mime: (m['mime'] as String?)?.trim() ?? '',
      hash: (m['hash'] as String?)?.trim() ?? '',
      thumbB64: (m['thumb_b64'] as String?)?.trim() ?? '',
      thumbPath: (m['thumb_path'] as String?)?.trim() ?? '',
    );
  }
}

enum FeedThumbKind { none, image, heicMark }

bool isImageMime(String mime) {
  switch (mime.trim().toLowerCase()) {
    case 'image/jpeg':
    case 'image/jpg':
    case 'image/png':
    case 'image/webp':
      return true;
    default:
      return false;
  }
}

bool isHeicMime(String mime) {
  switch (mime.trim().toLowerCase()) {
    case 'image/heic':
    case 'image/heif':
      return true;
    default:
      return false;
  }
}

Uint8List? decodeThumbBytes(ChatMessage m) {
  final b64 = m.thumbB64.trim();
  if (b64.isEmpty) return null;
  try {
    return Uint8List.fromList(base64Decode(b64));
  } catch (_) {
    return null;
  }
}

FeedThumbKind feedThumbKind(ChatMessage m) {
  if (!m.isFileAnnounce) return FeedThumbKind.none;
  if (decodeThumbBytes(m) != null) return FeedThumbKind.image;
  if (isHeicMime(m.mime)) return FeedThumbKind.heicMark;
  return FeedThumbKind.none;
}

class TransferInfo {
  const TransferInfo({
    required this.fileId,
    required this.name,
    required this.percent,
    required this.status,
    this.received = 0,
    this.total = 0,
    this.path = '',
  });

  final String fileId;
  final String name;
  final int percent;
  final String status;
  final int received;
  final int total;
  final String path;

  factory TransferInfo.fromJson(Map<String, dynamic> m) {
    return TransferInfo(
      fileId: (m['file_id'] as String?)?.trim() ?? '',
      name: (m['name'] as String?)?.trim() ?? '',
      percent: (m['percent'] as num?)?.toInt() ?? 0,
      status: (m['status'] as String?)?.trim() ?? '',
      received: (m['received'] as num?)?.toInt() ?? 0,
      total: (m['total'] as num?)?.toInt() ?? 0,
      path: (m['path'] as String?)?.trim() ?? '',
    );
  }
}

class LocalFileBytes {
  const LocalFileBytes({
    required this.name,
    required this.mime,
    required this.bytes,
  });

  final String name;
  final String mime;
  final List<int> bytes;
}

typedef LocalFilePicker = Future<LocalFileBytes?> Function();

class ChatSnapshot {
  const ChatSnapshot({
    required this.me,
    required this.peers,
    required this.network,
    required this.protoMajor,
    required this.protoMinor,
    required this.messages,
    this.transfers = const [],
  });

  final MeInfo me;
  final List<PeerInfo> peers;
  final String network;
  final int protoMajor;
  final int protoMinor;
  final List<ChatMessage> messages;
  final List<TransferInfo> transfers;

  TransferInfo? transferFor(String fileId) {
    for (final t in transfers) {
      if (t.fileId == fileId) return t;
    }
    return null;
  }
}

/// UI network state for status strip (mirrors TUI / DUD-NET-140).
String chatNetworkState({required String network, required int peerCount}) {
  if (network == 'no_network') return 'no_network';
  if (peerCount == 0) return 'alone';
  return 'ok';
}

String formatStatusStrip(ChatSnapshot snap) {
  final me = snap.me.name.trim().isEmpty ? '—' : snap.me.name.trim();
  final n = snap.peers.length;
  final state = chatNetworkState(network: snap.network, peerCount: n);
  final buf = StringBuffer('ДУДКА · $me · online $n · $state');
  if (snap.protoMajor > 0) {
    buf.write(' · proto ${snap.protoMajor}.${snap.protoMinor}');
  }
  return buf.toString();
}

String formatFeedLine(ChatMessage m) {
  final name = m.displayNameAtSend.trim().isEmpty ? '—' : m.displayNameAtSend.trim();
  String body;
  if (m.isFileAnnounce) {
    final fn = m.fileName.trim().isEmpty ? 'file' : m.fileName.trim();
    body = 'FILE $fn';
    if (m.size > 0) body = '$body ${m.size}';
  } else {
    body = m.text.trim();
  }
  if (m.ts.millisecondsSinceEpoch == 0) {
    return '· $name · $body';
  }
  final hh = m.ts.toUtc().hour.toString().padLeft(2, '0');
  final mm = m.ts.toUtc().minute.toString().padLeft(2, '0');
  return '$hh:$mm · $name · $body';
}

class SendResult {
  const SendResult({
    required this.status,
    required this.queued,
    required this.text,
    this.message,
  });

  final String status;
  final int queued;
  final String text;
  final ChatMessage? message;
}

class ScanResult {
  const ScanResult({
    required this.probed,
    required this.found,
    required this.peers,
  });

  final int probed;
  final int found;
  final List<PeerInfo> peers;
}

class EngineException implements Exception {
  EngineException(this.message);
  final String message;

  @override
  String toString() => message;
}
