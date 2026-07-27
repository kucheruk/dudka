import 'package:dudka/nick/fallback.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('meaningfulHost rejects localhost-like names', () {
    expect(meaningfulHost('MacBook-Pro'), isTrue);
    expect(meaningfulHost('localhost'), isFalse);
    expect(meaningfulHost('127.0.0.1'), isFalse);
    expect(meaningfulHost(''), isFalse);
  });

  test('generateAdjectiveAnimal uses dictionary form', () {
    final n = generateAdjectiveAnimal(pick: (max) => 0);
    expect(n, 'Сонный+Барсук');
    expect(n.contains('+'), isTrue);
  });

  test('resolveNickFallback prefers typed then host then generated', () {
    expect(resolveNickFallback(typed: '  Вася '), 'Вася');
    expect(
      resolveNickFallback(typed: '', hostname: 'Kitchen-Mac'),
      'Kitchen-Mac',
    );
    expect(
      resolveNickFallback(typed: '', hostname: 'localhost', pick: (max) => 1),
      'Быстрый+Ёжик',
    );
  });
}
