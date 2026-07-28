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
  test('announceFile / startFetch / cancelFetch / transfers', () async {
    final client = EngineClient(
      baseUrl: 'http://127.0.0.1:9',
      httpClient: MockClient((req) async {
        switch (req.url.path) {
          case '/files/announce':
            expect(req.method, 'POST');
            final body = jsonDecode(req.body) as Map<String, dynamic>;
            expect(body['name'], 'note.txt');
            expect(body['content_b64'], isNotNull);
            return http.Response(
              jsonEncode({
                'status': 'accepted',
                'queued': 0,
                'message': {
                  'type': 'file_announce',
                  'msg_id': 'm1',
                  'peer_id': 'me1',
                  'display_name_at_send': 'Anya',
                  'ts': '2026-07-27T12:00:00Z',
                  'file_id': 'f1',
                  'name': 'note.txt',
                  'size': 5,
                  'mime': 'text/plain',
                  'hash': 'sha256:abc',
                },
              }),
              200,
              headers: {'content-type': 'application/json; charset=utf-8'},
            );
          case '/files/fetch':
            expect(jsonDecode(req.body)['wait'], isFalse);
            return http.Response(
              '{"file_id":"f1","name":"note.txt","percent":10,"status":"downloading"}',
              200,
              headers: {'content-type': 'application/json; charset=utf-8'},
            );
          case '/files/transfers':
            return http.Response(
              '{"transfers":[{"file_id":"f1","name":"note.txt","percent":55,"status":"downloading","received":3,"total":5}]}',
              200,
              headers: {'content-type': 'application/json; charset=utf-8'},
            );
          case '/files/cancel':
            expect(jsonDecode(req.body)['file_id'], 'f1');
            return http.Response(
              '{"file_id":"f1","name":"note.txt","percent":55,"status":"cancelled"}',
              200,
              headers: {'content-type': 'application/json; charset=utf-8'},
            );
          default:
            return http.Response('nope', 404);
        }
      }),
    );

    final announced = await client.announceFile(
      name: 'note.txt',
      mime: 'text/plain',
      content: utf8.encode('hello'),
    );
    expect(announced.fileId, 'f1');
    expect(announced.fileName, 'note.txt');

    final started = await client.startFetch('f1');
    expect(started.status, 'downloading');
    expect(started.percent, 10);

    final xfers = await client.fetchTransfers();
    expect(xfers.single.percent, 55);

    final cancelled = await client.cancelFetch('f1');
    expect(cancelled.status, 'cancelled');
    client.close();
  });

  testWidgets('feed shows file actions: download progress cancel',
      (tester) async {
    final messages = <Map<String, Object?>>[
      {
        'type': 'file_announce',
        'msg_id': 'm1',
        'peer_id': 'p2',
        'display_name_at_send': 'Boris',
        'ts': '2026-07-27T12:34:00Z',
        'file_id': 'f1',
        'name': 'doc.bin',
        'size': 100,
        'mime': 'application/octet-stream',
        'hash': 'sha256:x',
      },
    ];
    var transferStatus = '';
    var transferPercent = 0;
    var fetchCalled = false;
    var cancelCalled = false;

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
          case '/files/transfers':
            if (transferStatus.isEmpty) {
              return http.Response(
                '{"transfers":[]}',
                200,
                headers: {'content-type': 'application/json; charset=utf-8'},
              );
            }
            return http.Response(
              jsonEncode({
                'transfers': [
                  {
                    'file_id': 'f1',
                    'name': 'doc.bin',
                    'percent': transferPercent,
                    'status': transferStatus,
                    'received': transferPercent,
                    'total': 100,
                  },
                ],
              }),
              200,
              headers: {'content-type': 'application/json; charset=utf-8'},
            );
          case '/files/fetch':
            fetchCalled = true;
            transferStatus = 'downloading';
            transferPercent = 40;
            return http.Response(
              '{"file_id":"f1","name":"doc.bin","percent":40,"status":"downloading"}',
              200,
              headers: {'content-type': 'application/json; charset=utf-8'},
            );
          case '/files/cancel':
            cancelCalled = true;
            transferStatus = 'cancelled';
            return http.Response(
              '{"file_id":"f1","name":"doc.bin","percent":40,"status":"cancelled"}',
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
        home: ChatScreen(
          client: client,
          pollInterval: Duration.zero,
          pickFiles: () async => const [], // unused in this test
        ),
      ),
    );
    await pumpFrames(tester);

    expect(find.textContaining('ФАЙЛ doc.bin'), findsOneWidget);
    expect(find.byKey(const Key('file-fetch-f1')), findsOneWidget);
    expect(find.text('СКАЧАТЬ'), findsOneWidget);

    await tester.tap(find.byKey(const Key('file-fetch-f1')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));
    await pumpFrames(tester);

    expect(fetchCalled, isTrue);
    expect(find.textContaining('40%'), findsOneWidget);
    expect(find.byKey(const Key('file-cancel-f1')), findsOneWidget);

    await tester.tap(find.byKey(const Key('file-cancel-f1')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));
    await pumpFrames(tester);

    expect(cancelCalled, isTrue);
    expect(find.textContaining('отменено'), findsOneWidget);
    client.close();
  });

  testWidgets('picker queues file; send icon announces it', (tester) async {
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
          case '/files/announce':
            final body = jsonDecode(req.body) as Map<String, dynamic>;
            messages.add({
              'type': 'file_announce',
              'msg_id': 'm-ann',
              'peer_id': 'me1',
              'display_name_at_send': 'Anya',
              'ts': '2026-07-27T12:00:00Z',
              'file_id': 'f-ann',
              'name': body['name'],
              'size': body['size'],
              'mime': body['mime'],
              'hash': body['hash'],
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
      MaterialApp(
        home: ChatScreen(
          client: client,
          pollInterval: Duration.zero,
          pickFiles: () async => [
            LocalFileBytes(
              name: 'hi.txt',
              mime: 'text/plain',
              bytes: utf8.encode('hi'),
            ),
          ],
        ),
      ),
    );
    await pumpFrames(tester);

    expect(find.byKey(const Key('chat-file')), findsOneWidget);
    await tester.tap(find.byKey(const Key('chat-file')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));
    await pumpFrames(tester);

    expect(find.byKey(const Key('chat-pending-files')), findsOneWidget);
    expect(find.text('hi.txt'), findsOneWidget);
    expect(find.textContaining('ФАЙЛ hi.txt'), findsNothing);

    await tester.tap(find.byKey(const Key('chat-blow')));
    await tester.pump(const Duration(milliseconds: 200));
    await pumpFrames(tester);

    expect(find.textContaining('ФАЙЛ hi.txt'), findsOneWidget);
    expect(find.byKey(const Key('chat-pending-files')), findsNothing);
    client.close();
  });

  testWidgets('completed download shows path and reveals it', (tester) async {
    const path = '/tmp/dudka/inbox/f1/doc.bin';
    var revealed = '';
    final client = EngineClient(
      baseUrl: 'http://127.0.0.1:9',
      httpClient: MockClient((req) async {
        switch (req.url.path) {
          case '/me':
            return http.Response('{"peer_id":"me1","name":"Anya"}', 200);
          case '/peers':
            return http.Response(
              '{"peers":[{"peer_id":"p2","display_name":"Boris"}]}',
              200,
            );
          case '/status':
            return http.Response(
              '{"proto_major":1,"proto_minor":0,"network":"ok"}',
              200,
            );
          case '/messages':
            return http.Response(
              jsonEncode({
                'messages': [
                  {
                    'type': 'file_announce',
                    'msg_id': 'm1',
                    'peer_id': 'p2',
                    'display_name_at_send': 'Boris',
                    'ts': '2026-07-28T10:00:00Z',
                    'file_id': 'f1',
                    'name': 'doc.bin',
                    'size': 100,
                    'mime': 'application/octet-stream',
                    'hash': 'sha256:x',
                  },
                ],
              }),
              200,
            );
          case '/files/transfers':
            return http.Response(
              jsonEncode({
                'transfers': [
                  {
                    'file_id': 'f1',
                    'name': 'doc.bin',
                    'percent': 100,
                    'status': 'done',
                    'path': path,
                  },
                ],
              }),
              200,
            );
          default:
            return http.Response('nope', 404);
        }
      }),
    );

    await tester.pumpWidget(
      MaterialApp(
        home: ChatScreen(
          client: client,
          pollInterval: Duration.zero,
          revealFile: (value) async => revealed = value,
        ),
      ),
    );
    await pumpFrames(tester);

    expect(find.textContaining(path), findsOneWidget);
    expect(find.byKey(const Key('file-reveal-f1')), findsOneWidget);
    await tester.tap(find.byKey(const Key('file-reveal-f1')));
    await tester.pump();
    expect(revealed, path);
    client.close();
  });
}
