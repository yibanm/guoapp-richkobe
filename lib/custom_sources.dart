import 'dart:convert';

/// 自定义源配置：Discuz 论坛空间页作目录、帖内对象存储直链作分集。
/// Cookie 仅存本机 custom_sources.json，界面回读时由原生层掩码。
class CustomSource {
  const CustomSource({
    required this.id,
    required this.name,
    required this.base,
    required this.spaceUid,
    this.cookie = '',
  });

  final String id;
  final String name;
  final String base;
  final String spaceUid;
  final String cookie;

  /// 源级 id，剧集 id 为 `custom:<slug>:<tid>`
  String get sourceId => id;

  Map<String, dynamic> toJson() => {
    'id': id,
    'name': name,
    'base': base,
    'spaceUid': spaceUid,
    'cookie': cookie,
  };

  static CustomSource fromJson(Map<String, dynamic> json) => CustomSource(
    id: json['id'] as String? ?? '',
    name: json['name'] as String? ?? '',
    base: json['base'] as String? ?? '',
    spaceUid: json['spaceUid'] as String? ?? '',
    cookie: json['cookie'] as String? ?? '',
  );

  CustomSource copyWith({
    String? name,
    String? base,
    String? spaceUid,
    String? cookie,
  }) => CustomSource(
    id: id,
    name: name ?? this.name,
    base: base ?? this.base,
    spaceUid: spaceUid ?? this.spaceUid,
    cookie: cookie ?? this.cookie,
  );
}

List<CustomSource> customSourcesFromJson(String raw) {
  final decoded = jsonDecode(raw);
  if (decoded is! Map<String, dynamic>) return const [];
  final profiles = decoded['profiles'];
  if (profiles is! List) return const [];
  return [
    for (final item in profiles)
      if (item is Map<String, dynamic>) CustomSource.fromJson(item),
  ];
}

String customSourcesToJson(List<CustomSource> sources) => jsonEncode({
  'profiles': [for (final source in sources) source.toJson()],
});
