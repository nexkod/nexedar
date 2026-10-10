import 'dart:convert';
import 'dart:io';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:package_info_plus/package_info_plus.dart';
import 'package:open_filex/open_filex.dart';
import 'package:path_provider/path_provider.dart';

class NexEdarUpdateInfo {
  final String projectId;
  final String versionName;
  final int versionCode;
  final String downloadUrl;
  final int fileSize;
  final String sha256;
  final String changelog;
  final bool isMandatory;

  NexEdarUpdateInfo({
    required this.projectId,
    required this.versionName,
    required this.versionCode,
    required this.downloadUrl,
    required this.fileSize,
    required this.sha256,
    required this.changelog,
    required this.isMandatory,
  });

  factory NexEdarUpdateInfo.fromJson(Map<String, dynamic> json) {
    return NexEdarUpdateInfo(
      projectId: json['project_id'] as String? ?? '',
      versionName: json['version_name'] as String? ?? '',
      versionCode: json['version_code'] as int? ?? 0,
      downloadUrl: json['download_url'] as String? ?? '',
      fileSize: json['file_size'] as int? ?? 0,
      sha256: json['sha256'] as String? ?? '',
      changelog: json['changelog'] as String? ?? '',
      isMandatory: json['is_mandatory'] as bool? ?? false,
    );
  }
}

class NexEdarUpdater {
  final String serverUrl;
  final String projectId;

  NexEdarUpdater({
    required this.serverUrl,
    required this.projectId,
  });

  /// Check if an update is available on NexEdar
  Future<NexEdarUpdateInfo?> checkForUpdate() async {
    if (!Platform.isAndroid) {
      // Direct sideload OTA updates are Android-specific
      return null;
    }

    try {
      final info = await PackageInfo.fromPlatform();
      final currentBuildNumber = int.tryParse(info.buildNumber) ?? 0;

      final url = Uri.parse('$serverUrl/api/v1/apps/$projectId/latest');
      final res = await http.get(url).timeout(const Duration(seconds: 10));

      if (res.statusCode == 200) {
        final data = jsonDecode(res.body) as Map<String, dynamic>;
        final update = NexEdarUpdateInfo.fromJson(data);

        if (update.versionCode > currentBuildNumber) {
          return update;
        }
      }
    } catch (e) {
      debugPrint('[NexEdarUpdater] Failed to check for updates: $e');
    }
    return null;
  }

  /// Automatically prompt user and perform update if available
  Future<void> checkAndPrompt(BuildContext context) async {
    final update = await checkForUpdate();
    if (update != null && context.mounted) {
      showDialog(
        context: context,
        barrierDismissible: !update.isMandatory,
        builder: (ctx) => _UpdateDialog(update: update),
      );
    }
  }
}

class _UpdateDialog extends StatefulWidget {
  final NexEdarUpdateInfo update;

  const _UpdateDialog({required this.update});

  @override
  State<_UpdateDialog> createState() => _UpdateDialogState();
}

class _UpdateDialogState extends State<_UpdateDialog> {
  bool _downloading = false;
  double _progress = 0.0;
  String _statusText = '';

  Future<void> _startDownloadAndInstall() async {
    setState(() {
      _downloading = true;
      _statusText = 'Downloading update...';
    });

    try {
      final client = http.Client();
      final request = http.Request('GET', Uri.parse(widget.update.downloadUrl));
      final response = await client.send(request);

      final contentLength = response.contentLength ?? widget.update.fileSize;
      final tempDir = await getTemporaryDirectory();
      final filePath = '${tempDir.path}/update_${widget.update.versionName}.apk';
      final file = File(filePath);
      final sink = file.openWrite();

      var bytesReceived = 0;
      await response.stream.listen((chunk) {
        sink.add(chunk);
        bytesReceived += chunk.length;
        if (contentLength > 0) {
          setState(() {
            _progress = bytesReceived / contentLength;
          });
        }
      }).asFuture();

      await sink.flush();
      await sink.close();

      setState(() {
        _statusText = 'Opening installer...';
      });

      // Launch native Android installer
      await OpenFilex.open(filePath, type: 'application/vnd.android.package-archive');
    } catch (e) {
      setState(() {
        _statusText = 'Download failed: $e';
        _downloading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final update = widget.update;

    return AlertDialog(
      title: Text('Update Available (v${update.versionName})'),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (update.changelog.isNotEmpty) ...[
              const Text('Changelog:', style: TextStyle(fontWeight: FontWeight.bold)),
              const SizedBox(height: 4),
              Text(update.changelog),
              const SizedBox(height: 12),
            ],
            if (_downloading) ...[
              LinearProgressIndicator(value: _progress > 0 ? _progress : null),
              const SizedBox(height: 8),
              Text(
                '${(_progress * 100).toStringAsFixed(1)}% - $_statusText',
                style: const TextStyle(fontSize: 12, color: Colors.grey),
              ),
            ] else if (_statusText.isNotEmpty) ...[
              Text(_statusText, style: const TextStyle(color: Colors.red, fontSize: 12)),
            ],
          ],
        ),
      ),
      actions: [
        if (!update.isMandatory && !_downloading)
          TextButton(
            onPressed: () => Navigator.of(context).pop(),
            child: const Text('Later'),
          ),
        ElevatedButton(
          onPressed: _downloading ? null : _startDownloadAndInstall,
          child: Text(_downloading ? 'Downloading...' : 'Update Now'),
        ),
      ],
    );
  }
}
