import test from 'node:test';
import assert from 'node:assert/strict';
import { messages, initialLanguage, rememberLanguage, languageStorageKey, translateError, translateLog } from '../src/i18n.js';

test('Chinese and English provide the same UI messages', () => {
  assert.deepEqual(Object.keys(messages.en).sort(), Object.keys(messages['zh-CN']).sort());
  for (const dictionary of Object.values(messages)) for (const value of Object.values(dictionary)) assert.ok(value.trim());
});
test('language restores local preference, settings, then system language safely', () => {
  assert.equal(initialLanguage({ getItem: () => 'en' }, 'zh-CN', 'zh-CN'), 'en');
  assert.equal(initialLanguage({ getItem: () => 'zh' }, 'en-US'), 'zh-CN');
  assert.equal(initialLanguage({ getItem: () => 'invalid' }, 'zh-TW'), 'zh-CN');
  assert.equal(initialLanguage(undefined, 'en-US'), 'en');
  assert.equal(initialLanguage({ getItem() { throw Error('disabled'); } }, 'en-US', 'zh-CN'), 'zh-CN');
  const writes = [];
  rememberLanguage({ setItem: (...args) => writes.push(args) }, 'zh-CN');
  assert.deepEqual(writes, [[languageStorageKey, 'zh']]);
  assert.doesNotThrow(() => rememberLanguage({ setItem() { throw Error('disabled'); } }, 'en'));
});
test('errors translate known labels while keeping OS details, paths and URLs', () => {
  const error = '无法读取目录: stat /Users/demo/中文书籍: no such file or directory';
  assert.equal(translateError('en', error), 'Unable to read folder: stat /Users/demo/中文书籍: no such file or directory');
  assert.equal(translateError('zh-CN', error), error);
  assert.equal(translateError('en', '已有传书服务运行于 http://127.0.0.1:54321，请先关闭该服务'), 'Another transfer service is running at http://127.0.0.1:54321. Stop it first.');
  assert.equal(translateError('en', '/Users/demo/无法读取目录'), '/Users/demo/无法读取目录');
  assert.equal(translateError('en', '检查更新失败: HTTP 503'), 'Update check failed: HTTP 503');
  assert.equal(translateError('en', '更新响应无效'), 'Invalid update response');
  assert.equal(translateLog('en', '2026/10/07 12:34:56 客户端操作失败 (save): 请先停止服务，再修改设置'), '2026/10/07 12:34:56 Client operation failed (save): Stop the service before editing settings');
});
