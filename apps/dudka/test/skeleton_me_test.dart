import 'package:dudka/app.dart';
import 'package:dudka/engine/client.dart';
import 'package:dudka/screens/me_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

void main() {
  testWidgets('skeleton MeScreen shows /me from engine', (tester) async {
    final client = EngineClient(
      baseUrl: 'http://127.0.0.1:9',
      httpClient: MockClient((req) async {
        expect(req.url.path, '/me');
        return http.Response(
          '{"peer_id":"peer-skel","name":"Skeleton"}',
          200,
          headers: {'content-type': 'application/json; charset=utf-8'},
        );
      }),
    );
    await tester.pumpWidget(
      DudkaApp(
        engineBase: 'http://127.0.0.1:9',
        client: client,
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byType(MeScreen), findsOneWidget);
    expect(find.byKey(const Key('me-name')), findsOneWidget);
    expect(find.text('name: Skeleton'), findsOneWidget);
    expect(find.text('peer_id: peer-skel'), findsOneWidget);
    // Not the P060 spike chrome.
    expect(find.textContaining('spike'), findsNothing);
    client.close();
  });
}
