import 'dart:convert';
import 'dart:io';

/// Persists whether the Flutter shell completed first-run nick (P062).
///
/// Uses synchronous IO: Flutter widget tests run under fake-async, and
/// `await File.writeAsString` never completes there (deadlock).
class FirstRunStore {
  FirstRunStore({required this.file});

  final File file;

  static FirstRunStore inDir(Directory dir) {
    return FirstRunStore(file: File('${dir.path}/flutter_first_run.json'));
  }

  bool isNickConfirmed() {
    if (!file.existsSync()) return false;
    try {
      final map = jsonDecode(file.readAsStringSync()) as Map<String, dynamic>;
      return map['nick_confirmed'] == true;
    } catch (_) {
      return false;
    }
  }

  String? confirmedNick() {
    if (!file.existsSync()) return null;
    try {
      final map = jsonDecode(file.readAsStringSync()) as Map<String, dynamic>;
      if (map['nick_confirmed'] != true) return null;
      final nick = (map['nick'] as String?)?.trim() ?? '';
      return nick.isEmpty ? null : nick;
    } catch (_) {
      return null;
    }
  }

  Future<void> markNickConfirmed([String? nick]) async {
    file.parent.createSync(recursive: true);
    final normalized = nick?.trim() ?? '';
    file.writeAsStringSync(
      jsonEncode({
        'nick_confirmed': true,
        if (normalized.isNotEmpty) 'nick': normalized,
      }),
    );
  }
}
