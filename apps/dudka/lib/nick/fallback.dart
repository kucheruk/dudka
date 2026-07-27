/// Nick fallbacks for first-run (DUD-CHAT-110 / P062).

const adjectives = <String>[
  'Сонный',
  'Быстрый',
  'Тихий',
  'Храбрый',
  'Уютный',
  'Рыжий',
  'Северный',
  'Домашний',
  'Весёлый',
  'Мутный',
];

const animals = <String>[
  'Барсук',
  'Ёжик',
  'Лисица',
  'Выдра',
  'Ворон',
  'Кот',
  'Ёрш',
  'Суслик',
  'Хомяк',
  'Кабан',
];

typedef NickPick = int Function(int max);

bool meaningfulHost(String host) {
  var h = host.trim();
  if (h.isEmpty) return false;
  if (h.endsWith('.')) h = h.substring(0, h.length - 1);
  switch (h.toLowerCase()) {
    case 'localhost':
    case 'localhost.localdomain':
    case 'localdomain':
    case '(none)':
    case 'none':
      return false;
  }
  // Reject pure numeric hosts (e.g. 127.0.0.1).
  final hasLetter = h.runes.any((r) {
    final ch = String.fromCharCode(r);
    return RegExp(r'[A-Za-zА-Яа-яЁё]').hasMatch(ch);
  });
  return hasLetter;
}

String generateAdjectiveAnimal({NickPick? pick}) {
  final p = pick ?? (_) => 0;
  final adj = adjectives[p(adjectives.length) % adjectives.length];
  final animal = animals[p(animals.length) % animals.length];
  return '$adj+$animal';
}

/// typed → meaningful hostname → generated adjective+animal.
String resolveNickFallback({
  required String typed,
  String? hostname,
  NickPick? pick,
}) {
  final t = typed.trim();
  if (t.isNotEmpty) return t;
  final host = (hostname ?? '').trim();
  if (meaningfulHost(host)) return host;
  return generateAdjectiveAnimal(pick: pick);
}
