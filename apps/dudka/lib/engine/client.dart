import 'dart:convert';

import 'package:http/http.dart' as http;

/// Thin loopback client for dudkad (P060–P063 / docs/design/flutter-bind.md).
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

  /// One frame for the chat wireframe: /me + /peers + /status + /messages (P063).
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

    return ChatSnapshot(
      me: me,
      peers: peers,
      network: (network == null || network.isEmpty) ? 'ok' : network,
      protoMajor: protoMajor,
      protoMinor: protoMinor,
      messages: messages,
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
  });

  final String msgId;
  final String peerId;
  final String displayNameAtSend;
  final DateTime ts;
  final String text;
  final String type;
  final String fileId;
  final String fileName;

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
    );
  }
}

class ChatSnapshot {
  const ChatSnapshot({
    required this.me,
    required this.peers,
    required this.network,
    required this.protoMajor,
    required this.protoMinor,
    required this.messages,
  });

  final MeInfo me;
  final List<PeerInfo> peers;
  final String network;
  final int protoMajor;
  final int protoMinor;
  final List<ChatMessage> messages;
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
  if (m.type == 'file_announce' || (m.fileId.isNotEmpty && m.fileName.isNotEmpty)) {
    final fn = m.fileName.trim().isEmpty ? 'file' : m.fileName.trim();
    body = 'FILE $fn';
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

class EngineException implements Exception {
  EngineException(this.message);
  final String message;

  @override
  String toString() => message;
}
