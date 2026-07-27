import 'dart:convert';

import 'package:dudka/engine/client.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

void main() {
  test('fetchSnapshot loads me, peers, status, messages', () async {
    final client = EngineClient(
      baseUrl: 'http://127.0.0.1:9',
      httpClient: MockClient((req) async {
        switch (req.url.path) {
          case '/me':
            return http.Response(
              '{"peer_id":"me1","name":"Anya"}',
              200,
              headers: {'content-type': 'application/json; charset=utf-8'},
            );
          case '/peers':
            return http.Response(
              jsonEncode({
                'peers': [
                  {'peer_id': 'p2', 'display_name': 'Boris'},
                ],
              }),
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
              jsonEncode({
                'messages': [
                  {
                    'type': 'chat',
                    'msg_id': 'm1',
                    'peer_id': 'p2',
                    'display_name_at_send': 'Boris',
                    'ts': '2026-07-27T12:34:00Z',
                    'text': 'hello',
                  },
                ],
              }),
              200,
              headers: {'content-type': 'application/json; charset=utf-8'},
            );
          default:
            return http.Response('nope', 404);
        }
      }),
    );

    final snap = await client.fetchSnapshot();
    expect(snap.me.name, 'Anya');
    expect(snap.me.peerId, 'me1');
    expect(snap.peers, hasLength(1));
    expect(snap.peers.single.displayName, 'Boris');
    expect(snap.network, 'ok');
    expect(snap.protoMajor, 1);
    expect(snap.messages, hasLength(1));
    expect(snap.messages.single.text, 'hello');
    expect(snap.messages.single.displayNameAtSend, 'Boris');
    expect(snap.messages.single.ts.toUtc().hour, 12);
    expect(snap.messages.single.ts.toUtc().minute, 34);
    client.close();
  });

  test('chatState alone when peers empty and network ok', () {
    expect(chatNetworkState(network: 'ok', peerCount: 0), 'alone');
    expect(chatNetworkState(network: 'no_network', peerCount: 0), 'no_network');
    expect(chatNetworkState(network: 'ok', peerCount: 2), 'ok');
  });

  test('formatFeedLine renders time nick text', () {
    final line = formatFeedLine(
      ChatMessage(
        msgId: 'm1',
        peerId: 'p2',
        displayNameAtSend: 'Boris',
        ts: DateTime.utc(2026, 7, 27, 12, 34),
        text: 'hello',
        type: 'chat',
      ),
    );
    expect(line, '12:34 · Boris · hello');
  });
}
