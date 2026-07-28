import 'dart:io';

import 'package:dudka/app.dart';
import 'package:dudka/engine/client.dart';
import 'package:dudka/screens/chat_screen.dart';
import 'package:dudka/session/first_run_store.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

import 'chat_mock.dart';

Future<void> pumpFrames(WidgetTester tester) async {
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 50));
  await tester.pump(const Duration(milliseconds: 50));
}

void main() {
  testWidgets('skeleton opens chat with /me nick after first-run done',
      (tester) async {
    final tmp = Directory.systemTemp.createTempSync('dudka-skel-');
    addTearDown(() {
      if (tmp.existsSync()) tmp.deleteSync(recursive: true);
    });
    final store = FirstRunStore(file: File('${tmp.path}/first_run.json'));
    await store.markNickConfirmed();

    final client = EngineClient(
      baseUrl: 'http://127.0.0.1:9',
      httpClient: MockClient((req) async {
        return chatSnapshotResponse(req, meName: 'Skeleton') ??
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
    expect(find.textContaining('ДУДКА · Skeleton'), findsOneWidget);
    expect(find.byKey(const Key('chat-status')), findsOneWidget);
    expect(find.textContaining('spike'), findsNothing);
    client.close();
  });
}
