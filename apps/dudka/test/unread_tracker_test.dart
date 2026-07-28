import 'package:dudka/desktop/unread_tracker.dart';
import 'package:dudka/engine/client.dart';
import 'package:flutter_test/flutter_test.dart';

ChatMessage message(String id, String peer) {
  return ChatMessage(
    msgId: id,
    peerId: peer,
    displayNameAtSend: peer,
    ts: DateTime.utc(2026, 7, 28, 12, 0, id.hashCode.abs() % 59),
    text: id,
    type: 'chat',
  );
}

void main() {
  test('initial tail is baseline and does not create unread messages', () {
    final tracker = UnreadTracker();

    expect(
      tracker.observe(
        messages: [message('old', 'peer')],
        selfPeerId: 'me',
        active: false,
      ),
      0,
    );
  });

  test('counts only new messages from other peers while inactive', () {
    final tracker = UnreadTracker();
    final old = message('old', 'peer');
    tracker.observe(messages: [old], selfPeerId: 'me', active: true);

    expect(
      tracker.observe(
        messages: [
          old,
          message('remote-1', 'peer'),
          message('mine', 'me'),
          message('remote-2', 'peer'),
        ],
        selfPeerId: 'me',
        active: false,
      ),
      2,
    );
  });

  test('does not count the same message twice and clears when active', () {
    final tracker = UnreadTracker();
    tracker.observe(messages: const [], selfPeerId: 'me', active: true);
    final remote = message('remote', 'peer');

    expect(
      tracker.observe(
        messages: [remote],
        selfPeerId: 'me',
        active: false,
      ),
      1,
    );
    expect(
      tracker.observe(
        messages: [remote],
        selfPeerId: 'me',
        active: false,
      ),
      1,
    );
    expect(
      tracker.observe(
        messages: [remote],
        selfPeerId: 'me',
        active: true,
      ),
      0,
    );
  });
}
