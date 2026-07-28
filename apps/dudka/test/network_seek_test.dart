import 'dart:convert';

import 'package:dudka/engine/client.dart';
import 'package:dudka/screens/chat_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

Future<void> pumpFrames(WidgetTester tester) async {
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 50));
  await tester.pump(const Duration(milliseconds: 50));
  await tester.pump(const Duration(milliseconds: 50));
}

void main() {
  test('startScan POSTs /scan and returns found count', () async {
    String? body;
    final client = EngineClient(
      baseUrl: 'http://127.0.0.1:9',
      httpClient: MockClient((req) async {
        expect(req.method, 'POST');
        expect(req.url.path, '/scan');
        body = req.body;
        return http.Response(
          jsonEncode({
            'probed': 3,
            'found': 1,
            'peers': [
              {'peer_id': 'p2', 'display_name': 'Boris'},
            ],
          }),
          200,
          headers: {'content-type': 'application/json; charset=utf-8'},
        );
      }),
    );
    final res = await client.startScan();
    expect(body, isNotNull);
    expect(res.found, 1);
    expect(res.probed, 3);
    expect(res.peers.single.displayName, 'Boris');
    client.close();
  });

  testWidgets('alone shows ИСКАТЬ; tap triggers scan and refreshes peers',
      (tester) async {
    var peers = <Map<String, String>>[];
    var scanned = false;
    final client = EngineClient(
      baseUrl: 'http://127.0.0.1:9',
      httpClient: MockClient((req) async {
        switch (req.url.path) {
          case '/me':
            return http.Response(
              '{"peer_id":"me1","name":"Katya"}',
              200,
              headers: {'content-type': 'application/json; charset=utf-8'},
            );
          case '/peers':
            return http.Response(
              jsonEncode({'peers': peers}),
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
              '{"messages":[]}',
              200,
              headers: {'content-type': 'application/json; charset=utf-8'},
            );
          case '/files/transfers':
            return http.Response(
              '{"transfers":[]}',
              200,
              headers: {'content-type': 'application/json; charset=utf-8'},
            );
          case '/scan':
            scanned = true;
            peers = [
              {'peer_id': 'p2', 'display_name': 'Boris'},
            ];
            return http.Response(
              jsonEncode({
                'probed': 1,
                'found': 1,
                'peers': peers,
              }),
              200,
              headers: {'content-type': 'application/json; charset=utf-8'},
            );
          default:
            return http.Response('nope', 404);
        }
      }),
    );

    await tester.pumpWidget(
      MaterialApp(
          home: ChatScreen(client: client, pollInterval: Duration.zero)),
    );
    await pumpFrames(tester);

    expect(find.textContaining('один'), findsOneWidget);
    expect(find.text('НИКОГО РЯДОМ'), findsOneWidget);
    expect(find.byKey(const Key('chat-seek')), findsOneWidget);
    expect(find.text('ИСКАТЬ'), findsOneWidget);

    await tester.tap(find.byKey(const Key('chat-seek')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));
    await tester.pump();

    expect(scanned, isTrue);
    expect(find.text('Boris'), findsOneWidget);
    expect(find.textContaining('онлайн 1'), findsOneWidget);
    expect(find.text('ИСКАТЬ'), findsNothing);
    client.close();
  });

  testWidgets('no_network never shows ИСКАТЬ', (tester) async {
    final client = EngineClient(
      baseUrl: 'http://127.0.0.1:9',
      httpClient: MockClient((req) async {
        switch (req.url.path) {
          case '/me':
            return http.Response(
              '{"peer_id":"me1","name":"Katya"}',
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
              '{"proto_major":1,"proto_minor":0,"network":"no_network"}',
              200,
              headers: {'content-type': 'application/json; charset=utf-8'},
            );
          case '/messages':
            return http.Response(
              '{"messages":[]}',
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
            return http.Response('nope', 404);
        }
      }),
    );
    await tester.pumpWidget(
      MaterialApp(
          home: ChatScreen(client: client, pollInterval: Duration.zero)),
    );
    await pumpFrames(tester);
    expect(find.text('НЕТ СЕТИ'), findsOneWidget);
    expect(find.text('ИСКАТЬ'), findsNothing);
    expect(find.byKey(const Key('chat-seek')), findsNothing);
    client.close();
  });
}
