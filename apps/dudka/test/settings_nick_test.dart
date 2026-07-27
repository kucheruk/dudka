import 'dart:convert';

import 'package:dudka/engine/client.dart';
import 'package:dudka/screens/chat_screen.dart';
import 'package:dudka/screens/settings_nick_screen.dart';
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

EngineClient mockClient({required String Function() meName, String? Function(String)? onNick}) {
  return EngineClient(
    baseUrl: 'http://127.0.0.1:9',
    httpClient: MockClient((req) async {
      switch (req.url.path) {
        case '/me':
          return http.Response(
            '{"peer_id":"me1","name":"${meName()}"}',
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
        case '/nick':
          final map = jsonDecode(req.body) as Map<String, dynamic>;
          final name = (map['name'] as String?) ?? '';
          final out = onNick?.call(name) ?? name;
          return http.Response(
            '{"peer_id":"me1","name":"$out"}',
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
  testWidgets('settings screen is nick-only (no avatar/email/phone)', (tester) async {
    final client = mockClient(meName: () => 'OldNick');
    await tester.pumpWidget(
      MaterialApp(
        home: SettingsNickScreen(client: client, initialNick: 'OldNick'),
      ),
    );
    await pumpFrames(tester);

    expect(find.textContaining('Ник'), findsWidgets);
    expect(find.byKey(const Key('settings-nick-field')), findsOneWidget);
    expect(find.byKey(const Key('settings-nick-save')), findsOneWidget);
    expect(find.textContaining('email'), findsNothing);
    expect(find.textContaining('телефон'), findsNothing);
    expect(find.textContaining('аватар'), findsNothing);
    expect(find.textContaining('пароль'), findsNothing);
    client.close();
  });

  testWidgets('chat opens settings; save nick updates chat strip', (tester) async {
    var name = 'OldNick';
    final client = mockClient(
      meName: () => name,
      onNick: (n) {
        name = n;
        return n;
      },
    );

    await tester.pumpWidget(
      MaterialApp(home: ChatScreen(client: client, pollInterval: Duration.zero)),
    );
    await pumpFrames(tester);

    expect(find.text('вы: OldNick'), findsOneWidget);
    expect(find.byKey(const Key('chat-settings')), findsOneWidget);

    await tester.tap(find.byKey(const Key('chat-settings')));
    await pumpFrames(tester);

    expect(find.byType(SettingsNickScreen), findsOneWidget);
    await tester.enterText(find.byKey(const Key('settings-nick-field')), 'NewNick');
    await tester.tap(find.byKey(const Key('settings-nick-save')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));
    await tester.pump();
    await pumpFrames(tester);

    expect(find.byType(SettingsNickScreen), findsNothing);
    expect(find.byType(ChatScreen), findsOneWidget);
    expect(find.text('вы: NewNick'), findsOneWidget);
    expect(find.textContaining('NewNick'), findsWidgets);
    client.close();
  });
}
