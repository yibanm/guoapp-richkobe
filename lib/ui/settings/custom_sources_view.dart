import 'package:flutter/material.dart';

import '../../core_bridge.dart';
import '../../custom_sources.dart';
import '../../local_store.dart';
import '../app_scope.dart';
import '../widgets/focus_ring.dart';

/// 自定义源管理：Discuz 论坛空间页作目录、帖内对象存储直链作分集。
/// Cookie 仅存本机，界面回读为掩码；连通测试直接打原生层探测。
class CustomSourcesView extends StatefulWidget {
  const CustomSourcesView({super.key, required this.store});
  final LocalStore store;

  @override
  State<CustomSourcesView> createState() => _CustomSourcesViewState();
}

class _CustomSourcesViewState extends State<CustomSourcesView> {
  List<CustomSource> _sources = const [];
  String? _error;
  bool _loading = true;
  String? _probing;
  String? _probeResult;

  AppRepository get _repository => AppScope.of(context).repository;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final sources = await _repository.customSources();
      if (!mounted) return;
      setState(() {
        _sources = sources;
        _loading = false;
      });
    } catch (failure) {
      if (!mounted) return;
      setState(() {
        _error = '$failure';
        _loading = false;
      });
    }
  }

  Future<void> _save(List<CustomSource> sources) async {
    try {
      await _repository.saveCustomSources(sources);
      widget.store.applyCustomSources(sources);
      await _load();
    } catch (failure) {
      if (mounted) setState(() => _error = '$failure');
    }
  }

  Future<void> _probe(CustomSource source) async {
    setState(() {
      _probing = source.id;
      _probeResult = null;
    });
    try {
      final message = await _repository.probeCustomSource(source.id);
      if (mounted) setState(() => _probeResult = '${source.name}：$message');
    } catch (failure) {
      if (mounted) setState(() => _probeResult = '${source.name}：$failure');
    } finally {
      if (mounted) setState(() => _probing = null);
    }
  }

  Future<void> _edit(CustomSource? existing) async {
    final name = TextEditingController(text: existing?.name ?? '');
    final base = TextEditingController(text: existing?.base ?? '');
    final uid = TextEditingController(text: existing?.spaceUid ?? '');
    final cookie = TextEditingController();
    final slug = TextEditingController(
      text: existing == null ? '' : existing.id.replaceFirst('custom:', ''),
    );
    final saved = await showDialog<bool>(
      context: context,
      builder: (dialog) => AlertDialog(
        title: Text(existing == null ? '新增自定义源' : '编辑自定义源'),
        content: SizedBox(
          width: 420,
          child: ListView(
            shrinkWrap: true,
            children: [
              TextField(
                controller: name,
                decoration: const InputDecoration(labelText: '显示名称'),
              ),
              TextField(
                controller: slug,
                enabled: existing == null,
                decoration: const InputDecoration(
                  labelText: '编号（英文小写，创建后不可改）',
                ),
              ),
              TextField(
                controller: base,
                decoration: const InputDecoration(
                  labelText: '论坛地址（如 https://www.example.com/main）',
                ),
              ),
              TextField(
                controller: uid,
                decoration: const InputDecoration(
                  labelText: '空间 UID（空间页地址里的 uid 数字）',
                ),
              ),
              TextField(
                controller: cookie,
                maxLines: 3,
                decoration: InputDecoration(
                  labelText: existing == null
                      ? 'Cookie（登录态，必需）'
                      : 'Cookie（留空保持原值，粘贴则覆盖）',
                ),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dialog, false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(dialog, true),
            child: const Text('保存'),
          ),
        ],
      ),
    );
    if (saved != true || !mounted) return;
    final trimmedSlug = slug.text.trim().toLowerCase();
    if (trimmedSlug.isEmpty || name.text.trim().isEmpty || base.text.trim().isEmpty || uid.text.trim().isEmpty) {
      setState(() => _error = '名称、编号、论坛地址与 UID 都不能为空');
      return;
    }
    final id = 'custom:$trimmedSlug';
    final next = [
      for (final source in _sources)
        if (source.id != id) source,
    ];
    final cookieValue = cookie.text.trim();
    next.add(
      CustomSource(
        id: id,
        name: name.text.trim(),
        base: base.text.trim(),
        spaceUid: uid.text.trim(),
        cookie: cookieValue.isEmpty
            ? (_sources.firstWhere(
                    (source) => source.id == id,
                    orElse: () => CustomSource(
                      id: id,
                      name: '',
                      base: '',
                      spaceUid: '',
                    ),
                  ).cookie)
            : cookieValue,
      ),
    );
    await _save(next);
  }

  @override
  Widget build(BuildContext context) {
    final admin = widget.store.profile.admin;
    return Scaffold(
      appBar: AppBar(title: const Text('自定义源')),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
        children: [
          if (!admin)
            const Padding(
              padding: EdgeInsets.only(bottom: 12),
              child: Text('需要管理员才能管理自定义源'),
            ),
          if (_error != null)
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: Text(_error!, style: const TextStyle(color: Colors.red)),
            ),
          if (_probeResult != null)
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: Text(_probeResult!),
            ),
          if (_loading)
            const Padding(
              padding: EdgeInsets.all(24),
              child: Center(child: CircularProgressIndicator()),
            ),
          for (final source in _sources)
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: FocusRing(
                radius: 16,
                child: Card(
                  child: ListTile(
                    title: Text(source.name),
                    subtitle: Text(
                      '${source.base}\nUID ${source.spaceUid} · '
                      'Cookie ${source.cookie.isEmpty ? '未设置' : '已设置'}',
                    ),
                    isThreeLine: true,
                    trailing: admin
                        ? Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              IconButton(
                                tooltip: '连通测试',
                                icon: _probing == source.id
                                    ? const SizedBox(
                                        width: 18,
                                        height: 18,
                                        child: CircularProgressIndicator(
                                          strokeWidth: 2,
                                        ),
                                      )
                                    : const Icon(Icons.network_check),
                                onPressed: _probing == source.id
                                    ? null
                                    : () => _probe(source),
                              ),
                              IconButton(
                                tooltip: '编辑',
                                icon: const Icon(Icons.edit_outlined),
                                onPressed: () => _edit(source),
                              ),
                              IconButton(
                                tooltip: '删除',
                                icon: const Icon(Icons.delete_outline),
                                onPressed: () => _save([
                                  for (final item in _sources)
                                    if (item.id != source.id) item,
                                ]),
                              ),
                            ],
                          )
                        : null,
                  ),
                ),
              ),
            ),
          if (admin)
            FilledButton.icon(
              onPressed: () => _edit(null),
              icon: const Icon(Icons.add),
              label: const Text('新增自定义源'),
            ),
          const Padding(
            padding: EdgeInsets.only(top: 16),
            child: Text(
              '说明：自定义源读取论坛空间页的帖子列表作目录，帖子正文里的视频直链作分集；'
              '需要登录态 Cookie 才能看到帖子内容。Cookie 与源配置只保存在本机，'
              '不会上传或写入仓库。论坛若限流或改 Cookie，用连通测试排查。',
            ),
          ),
        ],
      ),
    );
  }
}
