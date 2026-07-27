import 'dart:convert';

import 'package:dudka/engine/client.dart';
import 'package:dudka/screens/chat_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

/// Tiny JFIF payload used only for widget decoding (P068).
const tinyJpegB64 =
    '/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/wAALCAABAAEBAREA/8QAHwAAAQUBAQEBAQEAAAAAAAAAAAECAwQFBgcICQoL/8QAtRAAAgEDAwIEAwUFBAQAAAF9AQIDAAQRBRIhMUEGE1FhByJxFDKBkaEII0KxwRVS0fAkM2JyggkKFhcYGRolJicoKSo0NTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqDhIWGh4iJipKTlJWWl5iZmqKjpKWmp6ipqrKztLW2t7i5usLDxMXGx8jJytLT1NXW19jZ2uHi4+Tl5ufo6erx8vP09fb3+Pn6/9oACAEBAAA/APvV2yCo8UUA/9k=';

Future<void> pumpFrames(WidgetTester tester) async {
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 50));
  await tester.pump(const Duration(milliseconds: 50));
  await tester.pump(const Duration(milliseconds: 50));
}

EngineClient feedClient(List<Map<String, Object?>> messages) {
  return EngineClient(
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
            '{"peers":[{"peer_id":"p2","display_name":"Boris"}]}',
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
  test('ChatMessage parses thumb_b64; helpers classify mime', () {
    final m = ChatMessage.fromJson({
      'type': 'file_announce',
      'msg_id': 'm1',
      'peer_id': 'p2',
      'display_name_at_send': 'Boris',
      'ts': '2026-07-27T12:00:00Z',
      'file_id': 'f1',
      'name': 'pic.jpg',
      'size': 10,
      'mime': 'image/jpeg',
      'hash': 'sha256:x',
      'thumb_b64': tinyJpegB64,
    });
    expect(m.thumbB64, tinyJpegB64);
    expect(decodeThumbBytes(m), isNotNull);
    expect(decodeThumbBytes(m)!.length, greaterThan(10));
    expect(isImageMime('image/png'), isTrue);
    expect(isImageMime('image/webp'), isTrue);
    expect(isHeicMime('image/heic'), isTrue);
    expect(isImageMime('application/pdf'), isFalse);
    expect(feedThumbKind(m), FeedThumbKind.image);
    expect(
      feedThumbKind(
        ChatMessage.fromJson({
          'type': 'file_announce',
          'msg_id': 'm2',
          'peer_id': 'p2',
          'display_name_at_send': 'Boris',
          'ts': '2026-07-27T12:00:00Z',
          'file_id': 'f2',
          'name': 'x.heic',
          'size': 1,
          'mime': 'image/heic',
          'hash': 'sha256:y',
        }),
      ),
      FeedThumbKind.heicMark,
    );
    expect(
      feedThumbKind(
        ChatMessage.fromJson({
          'type': 'file_announce',
          'msg_id': 'm3',
          'peer_id': 'p2',
          'display_name_at_send': 'Boris',
          'ts': '2026-07-27T12:00:00Z',
          'file_id': 'f3',
          'name': 'a.bin',
          'size': 1,
          'mime': 'application/octet-stream',
          'hash': 'sha256:z',
        }),
      ),
      FeedThumbKind.none,
    );
  });

  testWidgets('jpeg announce shows Image thumb in feed', (tester) async {
    final client = feedClient([
      {
        'type': 'file_announce',
        'msg_id': 'm1',
        'peer_id': 'p2',
        'display_name_at_send': 'Boris',
        'ts': '2026-07-27T12:00:00Z',
        'file_id': 'f1',
        'name': 'pic.jpg',
        'size': 100,
        'mime': 'image/jpeg',
        'hash': 'sha256:x',
        'thumb_b64': tinyJpegB64,
      },
    ]);
    await tester.pumpWidget(
      MaterialApp(home: ChatScreen(client: client, pollInterval: Duration.zero)),
    );
    await pumpFrames(tester);
    expect(find.byKey(const Key('file-thumb-f1')), findsOneWidget);
    expect(find.byType(Image), findsOneWidget);
    expect(find.text('HEIC'), findsNothing);
    client.close();
  });

  testWidgets('heic without thumb shows honest HEIC mark', (tester) async {
    final client = feedClient([
      {
        'type': 'file_announce',
        'msg_id': 'm2',
        'peer_id': 'p2',
        'display_name_at_send': 'Boris',
        'ts': '2026-07-27T12:00:00Z',
        'file_id': 'f2',
        'name': 'shot.heic',
        'size': 100,
        'mime': 'image/heic',
        'hash': 'sha256:y',
      },
    ]);
    await tester.pumpWidget(
      MaterialApp(home: ChatScreen(client: client, pollInterval: Duration.zero)),
    );
    await pumpFrames(tester);
    expect(find.byKey(const Key('file-thumb-f2')), findsNothing);
    expect(find.byKey(const Key('file-heic-f2')), findsOneWidget);
    expect(find.text('HEIC'), findsOneWidget);
    client.close();
  });

  testWidgets('binary announce has no fake preview', (tester) async {
    final client = feedClient([
      {
        'type': 'file_announce',
        'msg_id': 'm3',
        'peer_id': 'p2',
        'display_name_at_send': 'Boris',
        'ts': '2026-07-27T12:00:00Z',
        'file_id': 'f3',
        'name': 'doc.bin',
        'size': 100,
        'mime': 'application/octet-stream',
        'hash': 'sha256:z',
      },
    ]);
    await tester.pumpWidget(
      MaterialApp(home: ChatScreen(client: client, pollInterval: Duration.zero)),
    );
    await pumpFrames(tester);
    expect(find.byType(Image), findsNothing);
    expect(find.text('HEIC'), findsNothing);
    expect(find.textContaining('FILE doc.bin'), findsOneWidget);
    client.close();
  });
}
