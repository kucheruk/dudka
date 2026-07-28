import 'package:dudka/layout/chat_layout.dart';
import 'package:dudka/screens/chat_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'chat_screen_test.dart' show mockChatClient, pumpFrames;

void main() {
  test('wide breakpoint is 700 dp (DUD-UI-140)', () {
    expect(dudkaWideBreakpoint, 700);
    expect(isWideChatLayout(699), isFalse);
    expect(isWideChatLayout(700), isTrue);
    expect(isWideChatLayout(1200), isTrue);
  });

  testWidgets('narrow width uses peer strip layout', (tester) async {
    final client = mockChatClient(
      meName: 'Anya',
      peers: [
        {'peer_id': 'p2', 'display_name': 'Boris'},
        {'peer_id': 'p3', 'display_name': 'Vera'},
      ],
    );

    await tester.binding.setSurfaceSize(const Size(390, 800));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    await tester.pumpWidget(
      MaterialApp(
          home: ChatScreen(client: client, pollInterval: Duration.zero)),
    );
    await pumpFrames(tester);

    expect(find.byKey(const Key('chat-layout-narrow')), findsOneWidget);
    expect(find.byKey(const Key('chat-layout-wide')), findsNothing);
    expect(find.byKey(const Key('chat-peers-strip')), findsOneWidget);
    expect(find.text('Boris'), findsOneWidget);
    expect(find.text('Vera'), findsOneWidget);
    expect(find.byKey(const Key('chat-compose')), findsOneWidget);
    client.close();
  });

  testWidgets('wide width uses dual-pane layout', (tester) async {
    final client = mockChatClient(
      meName: 'Anya',
      peers: [
        {'peer_id': 'p2', 'display_name': 'Boris'},
      ],
    );

    await tester.binding.setSurfaceSize(const Size(900, 700));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    await tester.pumpWidget(
      MaterialApp(
          home: ChatScreen(client: client, pollInterval: Duration.zero)),
    );
    await pumpFrames(tester);

    expect(find.byKey(const Key('chat-layout-wide')), findsOneWidget);
    expect(find.byKey(const Key('chat-layout-narrow')), findsNothing);
    expect(find.byKey(const Key('chat-peers-pane')), findsOneWidget);
    expect(find.byKey(const Key('chat-feed')), findsOneWidget);
    expect(find.byKey(const Key('chat-compose')), findsOneWidget);
    client.close();
  });

  testWidgets('resize desktop keeps compose text', (tester) async {
    final client = mockChatClient(
      meName: 'Anya',
      peers: [
        {'peer_id': 'p2', 'display_name': 'Boris'},
      ],
    );

    await tester.binding.setSurfaceSize(const Size(390, 800));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    await tester.pumpWidget(
      MaterialApp(
          home: ChatScreen(client: client, pollInterval: Duration.zero)),
    );
    await pumpFrames(tester);

    const draft = 'не теряй этот черновик';
    await tester.enterText(find.byKey(const Key('chat-compose')), draft);
    await tester.pump();
    expect(find.text(draft), findsOneWidget);

    await tester.binding.setSurfaceSize(const Size(960, 720));
    await tester.pump();
    await pumpFrames(tester);
    expect(find.byKey(const Key('chat-layout-wide')), findsOneWidget);
    expect(
        tester
            .widget<TextField>(find.byKey(const Key('chat-compose')))
            .controller!
            .text,
        draft);

    await tester.binding.setSurfaceSize(const Size(360, 640));
    await tester.pump();
    await pumpFrames(tester);
    expect(find.byKey(const Key('chat-layout-narrow')), findsOneWidget);
    expect(
        tester
            .widget<TextField>(find.byKey(const Key('chat-compose')))
            .controller!
            .text,
        draft);

    client.close();
  });
}
