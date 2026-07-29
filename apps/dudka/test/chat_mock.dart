import 'dart:convert';

import 'package:http/http.dart' as http;

/// Shared MockClient branches for chat snapshot endpoints (P063+).
http.Response? chatSnapshotResponse(http.Request req, {String meName = 'Me'}) {
  switch (req.url.path) {
    case '/me':
      return http.Response(
        '{"peer_id":"p1","name":"$meName"}',
        200,
        headers: {'content-type': 'application/json; charset=utf-8'},
      );
    case '/internet-consent':
      return http.Response(
        '{"enabled":true}',
        200,
        headers: {'content-type': 'application/json; charset=utf-8'},
      );
    case '/peers':
      return http.Response(
        '{"peers":[]}',
        200,
        headers: {'content-type': 'application/json; charset=utf-8'},
      );
    case '/status':
      return http.Response(
        '{"proto_major":1,"proto_minor":0,"network":"ok"}',
        200,
        headers: {'content-type': 'application/json; charset=utf-8'},
      );
    case '/messages':
      return http.Response(
        jsonEncode({'messages': <Object>[]}),
        200,
        headers: {'content-type': 'application/json; charset=utf-8'},
      );
    case '/files/transfers':
      return http.Response(
        '{"transfers":[]}',
        200,
        headers: {'content-type': 'application/json; charset=utf-8'},
      );
    default:
      return null;
  }
}
