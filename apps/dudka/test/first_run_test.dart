import 'dart:io';

import 'package:dudka/app.dart';
import 'package:dudka/engine/client.dart';
import 'package:dudka/screens/chat_screen.dart';
import 'package:dudka/screens/first_run_nick_screen.dart';
import 'package:dudka/session/first_run_store.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

import 'chat_mock.dart';

Future<void> pumpFrames(WidgetTester tester) async {
  // Avoid pumpAndSettle — CircularProgressIndicator animates forever.
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 50));
  await tester.pump(const Duration(milliseconds: 50));
  await tester.pump(const Duration(milliseconds: 50));
}

void main() {
  late Directory tmp;
  late FirstRunStore store;

  setUp(() {
    tmp = Directory.systemTemp.createTempSync('dudka-firstrun-');
    store = FirstRunStore(file: File('${tmp.path}/first_run.json'));
  });

  tearDown(() {
    if (tmp.existsSync()) tmp.deleteSync(recursive: true);
  });

  testWidgets('first run shows RU nick screen then chat', (tester) async {
    String? saved;
    final client = EngineClient(
      baseUrl: 'http://127.0.0.1:9',
      httpClient: MockClient((req) async {
        if (req.method == 'POST' && req.url.path == '/nick') {
          saved = 'Vasya';
          return http.Response(
            '{"peer_id":"p1","name":"Vasya"}',
            200,
            headers: {'content-type': 'application/json; charset=utf-8'},
          );
        }
        return chatSnapshotResponse(req, meName: saved ?? 'MacBook') ??
            http.Response('nope', 404);
      }),
    );

    await tester.pumpWidget(
      DudkaApp(
        engineBase: 'http://127.0.0.1:9',
        client: client,
        firstRunStore: store,
        chatPollInterval: Duration.zero,
      ),
    );
    await pumpFrames(tester);

    expect(find.byType(FirstRunNickScreen), findsOneWidget);
    expect(find.textContaining('Как вас зовут'), findsOneWidget);
    expect(find.text('Продолжить'), findsOneWidget);
    expect(find.text('Пропустить'), findsOneWidget);
    expect(find.textContaining('email'), findsNothing);
    expect(find.textContaining('телефон'), findsNothing);
    expect(find.textContaining('аватар'), findsNothing);

    await tester.enterText(find.byKey(const Key('nick-field')), 'Vasya');
    await tester.tap(find.byKey(const Key('nick-continue')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));
    await tester.pump();

    expect(store.isNickConfirmed(), isTrue);
    expect(find.byType(FirstRunNickScreen), findsNothing);
    expect(find.byType(ChatScreen), findsOneWidget);
    client.close();
  });

  testWidgets('skip uses fallback and opens chat', (tester) async {
    String? postedBody;
    final client = EngineClient(
      baseUrl: 'http://127.0.0.1:9',
      httpClient: MockClient((req) async {
        if (req.method == 'POST' && req.url.path == '/nick') {
          postedBody = req.body;
          return http.Response(
            '{"peer_id":"p1","name":"Sonny"}',
            200,
            headers: {'content-type': 'application/json; charset=utf-8'},
          );
        }
        return chatSnapshotResponse(req, meName: 'localhost') ??
            http.Response('nope', 404);
      }),
    );

    await tester.pumpWidget(
      DudkaApp(
        engineBase: 'http://127.0.0.1:9',
        client: client,
        firstRunStore: store,
        hostnameForFallback: () => 'localhost',
        nickPick: (max) => 0,
        chatPollInterval: Duration.zero,
      ),
    );
    await pumpFrames(tester);
    await tester.tap(find.byKey(const Key('nick-skip')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));
    await tester.pump();

    expect(store.isNickConfirmed(), isTrue);
    expect(postedBody, contains('Сонный+Барсук'));
    expect(find.byType(ChatScreen), findsOneWidget);
    client.close();
  });

  testWidgets('confirmed nick skips first-run straight to chat',
      (tester) async {
    await store.markNickConfirmed();
    final client = EngineClient(
      baseUrl: 'http://127.0.0.1:9',
      httpClient: MockClient((req) async {
        return chatSnapshotResponse(req, meName: 'Katya') ??
            http.Response('nope', 404);
      }),
    );
    await tester.pumpWidget(
      DudkaApp(
        engineBase: 'http://127.0.0.1:9',
        client: client,
        firstRunStore: store,
        chatPollInterval: Duration.zero,
      ),
    );
    await pumpFrames(tester);
    expect(find.byType(ChatScreen), findsOneWidget);
    expect(find.byType(FirstRunNickScreen), findsNothing);
    expect(find.textContaining('ДУДКА · Katya'), findsOneWidget);
    expect(find.byKey(const Key('chat-status')), findsOneWidget);
    client.close();
  });
}
