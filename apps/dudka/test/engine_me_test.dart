import 'package:dudka/engine/client.dart';
import 'package:dudka/screens/me_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

void main() {
  test('EngineClient.fetchMe parses peer_id and name', () async {
    final client = EngineClient(
      baseUrl: 'http://127.0.0.1:9',
      httpClient: MockClient((req) async {
        expect(req.url.path, '/me');
        return http.Response(
          '{"peer_id":"peer-a","name":"Vasya"}',
          200,
          headers: {'content-type': 'application/json; charset=utf-8'},
        );
      }),
    );
    final me = await client.fetchMe();
    expect(me.peerId, 'peer-a');
    expect(me.name, 'Vasya');
    client.close();
  });

  testWidgets('MeScreen shows GET /me fields', (tester) async {
    final client = EngineClient(
      baseUrl: 'http://127.0.0.1:9',
      httpClient: MockClient((req) async {
        return http.Response(
          '{"peer_id":"peer-spike","name":"Anya"}',
          200,
          headers: {'content-type': 'application/json; charset=utf-8'},
        );
      }),
    );
    await tester.pumpWidget(
      MaterialApp(
        home: MeScreen(engineBase: 'http://127.0.0.1:9', client: client),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('me-name')), findsOneWidget);
    expect(find.text('name: Anya'), findsOneWidget);
    expect(find.text('peer_id: peer-spike'), findsOneWidget);
    client.close();
  });
}
