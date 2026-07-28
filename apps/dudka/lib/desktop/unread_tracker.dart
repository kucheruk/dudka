import '../engine/client.dart';

class UnreadTracker {
  final Set<String> _known = {};
  bool _hasBaseline = false;
  int _count = 0;

  int get count => _count;

  int observe({
    required List<ChatMessage> messages,
    required String selfPeerId,
    required bool active,
  }) {
    final current = messages.map(_messageKey).toSet();
    if (!_hasBaseline) {
      _known
        ..clear()
        ..addAll(current);
      _hasBaseline = true;
      return _count;
    }

    if (!active) {
      for (final message in messages) {
        final key = _messageKey(message);
        if (!_known.contains(key) && message.peerId != selfPeerId) {
          _count++;
        }
      }
    } else {
      _count = 0;
    }
    _known
      ..clear()
      ..addAll(current);
    return _count;
  }

  int clear() {
    _count = 0;
    return _count;
  }

  String _messageKey(ChatMessage message) {
    if (message.msgId.isNotEmpty) return 'id:${message.msgId}';
    return [
      message.peerId,
      message.ts.toUtc().toIso8601String(),
      message.type,
      message.fileId,
      message.text,
    ].join('|');
  }
}
