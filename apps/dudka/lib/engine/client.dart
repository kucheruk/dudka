import 'dart:convert';

import 'package:http/http.dart' as http;

/// Thin loopback client for dudkad (P060/P061 / docs/design/flutter-bind.md).
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

  /// GET /me → peer identity for the skeleton home screen.
  Future<MeInfo> fetchMe() async {
    final res = await _http.get(_uri('/me'));
    if (res.statusCode != 200) {
      throw EngineException('GET /me → ${res.statusCode}: ${res.body}');
    }
    final map = jsonDecode(res.body) as Map<String, dynamic>;
    final peerId = (map['peer_id'] as String?)?.trim() ?? '';
    final name = (map['name'] as String?)?.trim() ?? '';
    if (peerId.isEmpty) {
      throw EngineException('GET /me missing peer_id');
    }
    return MeInfo(peerId: peerId, name: name.isEmpty ? '—' : name);
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
    final map = jsonDecode(res.body) as Map<String, dynamic>;
    final peerId = (map['peer_id'] as String?)?.trim() ?? '';
    final out = (map['name'] as String?)?.trim() ?? n;
    return MeInfo(peerId: peerId, name: out);
  }

  void close() => _http.close();
}

class MeInfo {
  const MeInfo({required this.peerId, required this.name});

  final String peerId;
  final String name;
}

class EngineException implements Exception {
  EngineException(this.message);
  final String message;

  @override
  String toString() => message;
}
