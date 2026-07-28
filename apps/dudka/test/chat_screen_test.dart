import 'dart:convert';

import 'package:dudka/engine/client.dart';
import 'package:dudka/screens/chat_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

Future<void> pumpFrames(WidgetTester tester) async {
  // Avoid pumpAndSettle — CircularProgressIndicator animates forever.
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 50));
  await tester.pump(const Duration(milliseconds: 50));
  await tester.pump(const Duration(milliseconds: 50));
}

EngineClient mockChatClient({
  required String meName,
  List<Map<String, String>> peers = const [],
  String network = 'ok',
  List<Map<String, Object?>> messages = const [],
}) {
  return EngineClient(
    baseUrl: 'http://127.0.0.1:9',
    httpClient: MockClient((req) async {
      switch (req.url.path) {
        case '/me':
          return http.Response(
            '{"peer_id":"me1","name":"$meName"}',
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
            '{"proto_major":1,"proto_minor":0,"network":"$network"}',
            200,
            headers: {'content-type': 'application/json; charset=utf-8'},
          );
        case '/messages':
          return http.Response(
            jsonEncode({'messages': messages}),
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
}

void main() {
  testWidgets('chat shows status strip, peers, and text feed', (tester) async {
    final client = mockChatClient(
      meName: 'Anya',
      peers: [
        {'peer_id': 'p2', 'display_name': 'Boris'},
        {'peer_id': 'p3', 'display_name': 'Vera'},
      ],
      messages: [
        {
          'type': 'chat',
          'msg_id': 'm1',
          'peer_id': 'p2',
          'display_name_at_send': 'Boris',
          'ts': '2026-07-27T12:34:00Z',
          'text': 'hello lane',
        },
      ],
    );

    await tester.pumpWidget(
      MaterialApp(
          home: ChatScreen(client: client, pollInterval: Duration.zero)),
    );
    await pumpFrames(tester);

    expect(find.byKey(const Key('chat-status')), findsOneWidget);
    expect(find.byKey(const Key('chat-header')), findsOneWidget);
    expect(find.text('Чат'), findsNothing);
    expect(find.textContaining('Anya'), findsWidgets);
    expect(find.textContaining('онлайн 2'), findsOneWidget);
    expect(find.textContaining('ок'), findsWidgets);

    expect(find.byKey(const Key('chat-peers')), findsOneWidget);
    expect(find.text('Boris'), findsWidgets);
    expect(find.text('Vera'), findsOneWidget);
    expect(find.textContaining('НИКОГО РЯДОМ'), findsNothing);

    expect(find.byKey(const Key('chat-feed')), findsOneWidget);
    expect(find.byType(SelectionArea), findsOneWidget);
    expect(find.textContaining('12:34 · Boris · hello lane'), findsOneWidget);

    expect(find.byIcon(Icons.attach_file), findsOneWidget);
    expect(find.byIcon(Icons.send), findsOneWidget);
    expect(find.byKey(const Key('chat-compose')), findsOneWidget);

    client.close();
  });

  testWidgets('empty peers shows НИКОГО РЯДОМ and один in status',
      (tester) async {
    final client = mockChatClient(meName: 'Katya', peers: const []);
    await tester.pumpWidget(
      MaterialApp(
          home: ChatScreen(client: client, pollInterval: Duration.zero)),
    );
    await pumpFrames(tester);

    expect(find.textContaining('один'), findsOneWidget);
    expect(find.textContaining('онлайн 0'), findsOneWidget);
    expect(find.text('НИКОГО РЯДОМ'), findsOneWidget);
    expect(find.text('ИСКАТЬ'), findsOneWidget);
    expect(find.byKey(const Key('chat-feed-empty')), findsOneWidget);
    client.close();
  });

  testWidgets('no_network shows НЕТ СЕТИ without ИСКАТЬ', (tester) async {
    final client = mockChatClient(meName: 'Katya', network: 'no_network');
    await tester.pumpWidget(
      MaterialApp(
          home: ChatScreen(client: client, pollInterval: Duration.zero)),
    );
    await pumpFrames(tester);

    expect(find.textContaining('нет сети'), findsOneWidget);
    expect(find.text('НЕТ СЕТИ'), findsOneWidget);
    expect(find.text('ИСКАТЬ'), findsNothing);
    client.close();
  });
}
