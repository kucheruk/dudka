import 'package:dudka/engine/host.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('parseListenLine extracts host:port', () {
    expect(
      EngineHost.parseListenLine('listen=127.0.0.1:17880'),
      '127.0.0.1:17880',
    );
    expect(EngineHost.parseListenLine('ready peer_id=x name=y'), isNull);
    expect(EngineHost.parseListenLine('listen=::1:9'), '::1:9');
  });

  test('parseReadyLine detects ready banner', () {
    expect(EngineHost.parseReadyLine('ready peer_id=abc name=Spike'), isTrue);
    expect(EngineHost.parseReadyLine('listen=127.0.0.1:1'), isFalse);
  });

  test('baseUrlFromListen builds http URL', () {
    expect(
      EngineHost.baseUrlFromListen('127.0.0.1:17880'),
      'http://127.0.0.1:17880',
    );
  });

  test('saved engine identity is not overwritten by a placeholder name', () {
    final host = EngineHost(binaryPath: '/bin/dudkad', dataDir: '/data');
    final args = host.arguments(announcePort: 41777);
    expect(args, isNot(contains('-name')));
  });

  test('recovery nick can seed an empty engine data directory', () {
    final host = EngineHost(
      binaryPath: '/bin/dudkad',
      dataDir: '/data',
      name: 'Евгений',
    );
    final args = host.arguments(announcePort: 41777);
    final nameIndex = args.indexOf('-name');
    expect(nameIndex, greaterThanOrEqualTo(0));
    expect(args[nameIndex + 1], 'Евгений');
  });
}
