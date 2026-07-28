import 'package:dudka/engine/client.dart';
import 'package:dudka/screens/chat_screen.dart';
import 'package:dudka/theme/dudka_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

void main() {
  test('formatStatusStrip and formatFeedLine are Russian (P072)', () {
    final ok = ChatSnapshot(
      me: const MeInfo(peerId: 'p1', name: 'Аня'),
      peers: const [PeerInfo(peerId: 'p2', displayName: 'Боря')],
      network: 'ok',
      protoMajor: 1,
      protoMinor: 0,
      messages: const [],
    );
    expect(ok.onlineCount, 2);
    expect(ok.onlineParticipants.first.displayName, 'Аня');
    expect(ok.remotePeerCount, 1);
    expect(formatStatusStrip(ok), contains('онлайн 2'));
    expect(formatStatusStrip(ok), contains('ок'));
    expect(formatStatusStrip(ok), contains('прото 1.0'));
    expect(formatStatusStrip(ok), isNot(contains('online')));
    expect(formatStatusStrip(ok), isNot(contains(' proto ')));

    final alone = ChatSnapshot(
      me: const MeInfo(peerId: 'p1', name: 'Аня'),
      peers: const [],
      network: 'ok',
      protoMajor: 0,
      protoMinor: 0,
      messages: const [],
    );
    expect(alone.onlineCount, 1);
    expect(alone.remotePeerCount, 0);
    expect(formatStatusStrip(alone), contains('один'));
    expect(formatStatusStrip(alone), contains('онлайн 1'));
    expect(formatStatusStrip(alone), isNot(contains('alone')));

    final noNet = ChatSnapshot(
      me: const MeInfo(peerId: 'p1', name: 'Аня'),
      peers: const [],
      network: 'no_network',
      protoMajor: 0,
      protoMinor: 0,
      messages: const [],
    );
    expect(formatStatusStrip(noNet), contains('нет сети'));
    expect(formatStatusStrip(noNet), isNot(contains('no_network')));

    final fileLine = formatFeedLine(
      ChatMessage(
        msgId: 'm1',
        peerId: 'p2',
        displayNameAtSend: 'Боря',
        ts: DateTime.utc(2026, 7, 27, 12, 34),
        text: '',
        type: 'file_announce',
        fileId: 'fid',
        fileName: 'doc.bin',
        size: 10,
      ),
    );
    expect(fileLine, contains('ФАЙЛ doc.bin'));
    expect(fileLine, isNot(contains('FILE ')));
  });

  testWidgets('chat labels use Russian silkscreen (P072)', (tester) async {
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
              '{"peers":[{"peer_id":"p2","display_name":"Boris"},{"peer_id":"p3","display_name":"Anna"}]}',
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
          default:
            return http.Response('nope', 404);
        }
      }),
    );
    await tester.pumpWidget(
      MaterialApp(
        theme: buildDudkaTheme(),
        home: ChatScreen(client: client, pollInterval: const Duration(days: 1)),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));
    await tester.pump(const Duration(milliseconds: 50));
    expect(find.text('СОСЕДИ'), findsWidgets);
    expect(find.text('PEERS'), findsNothing);
    expect(find.textContaining('онлайн 3'), findsOneWidget);
    expect(find.text('Katya · ВЫ'), findsOneWidget);
    expect(find.textContaining('online '), findsNothing);
    client.close();
  });
}
