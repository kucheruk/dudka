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
