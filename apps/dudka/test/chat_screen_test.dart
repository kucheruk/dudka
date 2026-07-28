import 'dart:convert';

import 'package:dudka/engine/client.dart';
import 'package:dudka/screens/chat_screen.dart';
import 'package:dudka/update/update_manager.dart';
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

class FakeUpdateController extends ChangeNotifier implements UpdateController {
  FakeUpdateController(this._snapshot);

  UpdateSnapshot _snapshot;
  int startCalls = 0;
  int activateCalls = 0;

  @override
  UpdateSnapshot get snapshot => _snapshot;

  @override
  void start() => startCalls++;

  @override
  Future<void> activate() async => activateCalls++;

  void setSnapshot(UpdateSnapshot value) {
    _snapshot = value;
    notifyListeners();
  }
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
        home: ChatScreen(client: client, pollInterval: Duration.zero),
      ),
    );
    await pumpFrames(tester);

    expect(find.byKey(const Key('chat-status')), findsOneWidget);
    expect(find.byKey(const Key('chat-header')), findsOneWidget);
    expect(find.text('Чат'), findsNothing);
    expect(find.textContaining('Anya'), findsWidgets);
    expect(find.textContaining('онлайн 3'), findsOneWidget);
    expect(find.textContaining('ок'), findsWidgets);

    expect(find.byKey(const Key('chat-peers')), findsOneWidget);
    expect(find.text('Anya · ВЫ'), findsOneWidget);
    expect(find.text('Boris'), findsWidgets);
    expect(find.text('Vera'), findsOneWidget);
    expect(find.textContaining('БОЛЬШЕ НИКОГО РЯДОМ'), findsNothing);

    expect(find.byKey(const Key('chat-feed')), findsOneWidget);
    expect(find.byType(SelectionArea), findsOneWidget);
    expect(find.textContaining('12:34 · Boris · hello lane'), findsOneWidget);

    expect(find.byIcon(Icons.attach_file), findsOneWidget);
    expect(find.byIcon(Icons.send), findsOneWidget);
    expect(find.byKey(const Key('chat-compose')), findsOneWidget);

    client.close();
  });

  testWidgets('empty remote peers still shows self and онлайн 1', (
    tester,
  ) async {
    final client = mockChatClient(meName: 'Katya', peers: const []);
    await tester.pumpWidget(
      MaterialApp(
        home: ChatScreen(client: client, pollInterval: Duration.zero),
      ),
    );
    await pumpFrames(tester);

    expect(find.textContaining('один'), findsOneWidget);
    expect(find.textContaining('онлайн 1'), findsOneWidget);
    expect(find.text('Katya · ВЫ'), findsOneWidget);
    expect(find.text('БОЛЬШЕ НИКОГО РЯДОМ'), findsOneWidget);
    expect(find.text('ИСКАТЬ'), findsOneWidget);
    expect(find.byKey(const Key('chat-feed-empty')), findsOneWidget);
    client.close();
  });

  testWidgets('no_network shows НЕТ СЕТИ without ИСКАТЬ', (tester) async {
    final client = mockChatClient(meName: 'Katya', network: 'no_network');
    await tester.pumpWidget(
      MaterialApp(
        home: ChatScreen(client: client, pollInterval: Duration.zero),
      ),
    );
    await pumpFrames(tester);

    expect(find.textContaining('нет сети'), findsOneWidget);
    expect(find.textContaining('онлайн 1'), findsOneWidget);
    expect(find.text('Katya · ВЫ'), findsOneWidget);
    expect(find.text('НЕТ СЕТИ'), findsOneWidget);
    expect(find.text('ИСКАТЬ'), findsNothing);
    client.close();
  });

  testWidgets('verified update shows button and activates once', (
    tester,
  ) async {
    final client = mockChatClient(meName: 'Katya');
    final updater = FakeUpdateController(
      const UpdateSnapshot(phase: UpdatePhase.ready, version: '0.3.1'),
    );
    await tester.pumpWidget(
      MaterialApp(
        home: ChatScreen(
          client: client,
          pollInterval: Duration.zero,
          updater: updater,
        ),
      ),
    );
    await pumpFrames(tester);

    expect(updater.startCalls, 1);
    expect(find.byKey(const Key('update-ready')), findsOneWidget);
    expect(find.text('АПДЕЙТ 0.3.1'), findsOneWidget);

    await tester.tap(find.byKey(const Key('update-ready')));
    await tester.pump();
    expect(updater.activateCalls, 1);

    client.close();
    updater.dispose();
  });

  testWidgets('update action stays hidden until package is verified', (
    tester,
  ) async {
    final client = mockChatClient(meName: 'Katya');
    final updater = FakeUpdateController(
      const UpdateSnapshot(phase: UpdatePhase.downloading, version: '0.3.1'),
    );
    await tester.pumpWidget(
      MaterialApp(
        home: ChatScreen(
          client: client,
          pollInterval: Duration.zero,
          updater: updater,
        ),
      ),
    );
    await pumpFrames(tester);

    expect(find.byKey(const Key('update-ready')), findsNothing);
    updater.setSnapshot(
      const UpdateSnapshot(phase: UpdatePhase.ready, version: '0.3.1'),
    );
    await tester.pump();
    expect(find.byKey(const Key('update-ready')), findsOneWidget);

    client.close();
    updater.dispose();
  });
}
