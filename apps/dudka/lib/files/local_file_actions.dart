import 'dart:io';

import 'package:file_selector/file_selector.dart';
import 'package:mime/mime.dart';

import '../engine/client.dart';

/// Opens the platform-native picker and materializes every selected file.
Future<List<LocalFileBytes>> pickLocalFiles() async {
  final selected = await openFiles();
  final files = <LocalFileBytes>[];
  for (final file in selected) {
    final bytes = await file.readAsBytes();
    final name = file.name.trim().isEmpty
        ? file.path.split(Platform.pathSeparator).last
        : file.name;
    files.add(
      LocalFileBytes(
        name: name,
        mime: file.mimeType ??
            lookupMimeType(name, headerBytes: bytes.take(32).toList()) ??
            'application/octet-stream',
        bytes: bytes,
      ),
    );
  }
  return files;
}

/// Reveals a completed transfer in the native desktop file manager.
///
/// Mobile platforms still display the exact path in the chat UI, but do not
/// have a Finder/Explorer equivalent that can reveal an app-private path.
Future<void> revealDownloadedFile(String path) async {
  final target = path.trim();
  if (target.isEmpty) {
    throw EngineException('движок не вернул путь к скачанному файлу');
  }

  ProcessResult result;
  if (Platform.isMacOS) {
    result = await Process.run('open', ['-R', target]);
  } else if (Platform.isWindows) {
    result = await Process.run('explorer.exe', ['/select,$target']);
  } else if (Platform.isLinux) {
    result = await Process.run('xdg-open', [File(target).parent.path]);
  } else {
    throw EngineException('файл сохранён: $target');
  }
  if (result.exitCode != 0) {
    final detail = result.stderr.toString().trim();
    throw EngineException(
      detail.isEmpty ? 'не удалось показать файл: $target' : detail,
    );
  }
}

typedef DownloadedFileRevealer = Future<void> Function(String path);
