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
  test('sendText POSTs /send and returns accepted', () async {
    String? body;
    final client = EngineClient(
      baseUrl: 'http://127.0.0.1:9',
      httpClient: MockClient((req) async {
        expect(req.method, 'POST');
        expect(req.url.path, '/send');
        body = req.body;
        return http.Response(
          '{"status":"accepted","queued":0,"message":{"type":"chat","msg_id":"m1","peer_id":"me1","display_name_at_send":"Anya","ts":"2026-07-27T12:00:00Z","text":"yo"}}',
          200,
          headers: {'content-type': 'application/json; charset=utf-8'},
        );
      }),
    );
    final res = await client.sendText('yo');
    expect(body, contains('"text":"yo"'));
    expect(res.status, 'accepted');
    expect(res.text, 'yo');
    client.close();
  });

  testWidgets('ДУНУТЬ sends text and shows it in feed', (tester) async {
    final messages = <Map<String, Object?>>[];
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
              jsonEncode({'messages': messages}),
              200,
              headers: {'content-type': 'application/json; charset=utf-8'},
            );
          case '/send':
            final map = jsonDecode(req.body) as Map<String, dynamic>;
            final text = map['text'] as String;
            messages.add({
              'type': 'chat',
              'msg_id': 'm-${messages.length + 1}',
              'peer_id': 'me1',
              'display_name_at_send': 'Anya',
              'ts': '2026-07-27T15:01:00Z',
              'text': text,
            });
            return http.Response(
              jsonEncode({
                'status': 'accepted',
                'queued': 0,
                'message': messages.last,
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
      MaterialApp(home: ChatScreen(client: client, pollInterval: Duration.zero)),
    );
    await pumpFrames(tester);

    expect(find.byKey(const Key('chat-compose')), findsOneWidget);
    expect(find.byKey(const Key('chat-blow')), findsOneWidget);
    expect(find.text('ДУНУТЬ'), findsOneWidget);

    await tester.enterText(find.byKey(const Key('chat-compose')), 'hello from flutter');
    await tester.tap(find.byKey(const Key('chat-blow')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));
    await tester.pump();

    expect(find.textContaining('hello from flutter'), findsOneWidget);
    expect(find.textContaining('Anya'), findsWidgets);
    client.close();
  });
}
