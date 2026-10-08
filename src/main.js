import './styles.css';
import appIconUrl from '../docs/app-icon.svg';
import { invoke, isTauri } from '@tauri-apps/api/core';
import { getCurrentWindow } from '@tauri-apps/api/window';
import { getVersion } from '@tauri-apps/api/app';
import packageInfo from '../package.json';
import { open } from '@tauri-apps/plugin-dialog';
import { writeText } from '@tauri-apps/plugin-clipboard-manager';
import { openUrl, openPath } from '@tauri-apps/plugin-opener';
import { parsePort } from './settings.js';
import { messages, initialLanguage, rememberLanguage, translateError, translateLog } from './i18n.js';

const app = document.querySelector('#app');
const desktop = isTauri();
let status = { running: false, addresses: [], settings: { directory: '', port: 0, language: '' } };
let busy = false;
let storage;
try { storage = window.localStorage; } catch { /* Local settings still persist language. */ }
let lang = initialLanguage(storage, navigator.language);
let languageRevision = 0;
let languageSave = Promise.resolve();
let settingsLoaded = false;
let ready = false;
let polling = false;
let revision = 0;
let noticeMessage = null;
const copiedAddresses = new Map();
let serviceLogs = [];
let renderedStatus = '';
let view = 'transfer';
let currentVersion = packageInfo.version;
let updateState = 'idle';
let updateResult = null;
let updateError = '';
let openingUpdate = false;
let openingFeedback = false;
const t = (key) => messages[lang][key] ?? key;
const request = (command, settings) => invoke('desktop_request', { command, settings: settings ?? null });
const bookIcon = `<img class="app-logo" src="${appIconUrl}" alt="" aria-hidden="true">`;
const icons = {
  transfer: '<path d="M4 7h15m-4-4 4 4-4 4M20 17H5m4-4-4 4 4 4"/>',
  settings: '<path d="M4 7h16M4 17h16"/><circle cx="9" cy="7" r="3"/><circle cx="15" cy="17" r="3"/>',
  folder: '<path d="M3 7a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v10H3Z"/>',
  wifi: '<path d="M2 8a16 16 0 0 1 20 0M5 12a11 11 0 0 1 14 0M9 16a5 5 0 0 1 6 0m-3 4h.01"/>',
  feedback: '<path d="M21 11a8 8 0 0 1-8 8H7l-4 3V11a8 8 0 0 1 8-8h2a8 8 0 0 1 8 8Z"/><path d="M8 9h8M8 13h5"/>',
  info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v6m0-10v1"/>',
  shield: '<path d="m12 3 8 3v6c0 5-8 9-8 9s-8-4-8-9V6Z"/><path d="m8 12 3 3 5-6"/>',
};
const icon = name => `<svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${icons[name]}</svg>`;
const directFormats = 'AZW3 · MOBI · TXT · AZW · KFX · PRC';
const githubLogo = '<svg class="github-logo" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82A7.65 7.65 0 0 1 8 3.86c.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z"/></svg>';
const formatNotice = (linkID, tone = 'danger') => `
  <article class="format-notice format-notice--${tone}" aria-labelledby="${linkID}-title">
    <h3 id="${linkID}-title">${icon('info')}<span data-i18n="unsupportedFormats"></span></h3>
    <p><strong data-i18n="unsupportedHint"></strong></p>
    <p data-i18n="sendHint"></p>
    <a id="${linkID}" class="send-to-kindle-link" href="https://www.amazon.com/sendtokindle" target="_blank" rel="noopener noreferrer" data-send-to-kindle data-i18n="sendAction"></a>
  </article>`;
app.innerHTML = `
  <aside class="sidebar">
    <div class="brand"><span class="brand-mark">${bookIcon}</span><span data-i18n="brandName"></span></div>
    <div class="local-badge">${icon('shield')}<span data-i18n="openSource"></span></div>
    <nav aria-label="Navigation">
      <button class="nav-item active" data-view="transfer">${icon('transfer')}<span data-i18n="transfer"></span><span class="nav-mark"></span></button>
      <button class="nav-item" data-view="settings">${icon('settings')}<span data-i18n="settings"></span><span class="nav-mark"></span></button>
    </nav>
    <div class="sidebar-bottom"><a id="feedback" class="nav-item" href="https://api.ip21.cn/products/11/feedback" target="_blank" rel="noopener noreferrer">${icon('feedback')}<span data-i18n="feedback"></span></a><button class="nav-item" data-view="about">${icon('info')}<span data-i18n="about"></span><span class="nav-mark"></span></button><span class="version"><span data-i18n="brandName"></span><span id="app-version"></span></span></div>
  </aside>
  <div class="workspace">
    <main>
      <div class="top-actions"><a id="github" class="github-link" href="https://github.com/goldenwind/fangxu-kindle-transfer" target="_blank" rel="noopener noreferrer">${githubLogo}<span>GitHub ↗</span></a><div class="language-switch" role="group" data-i18n-aria="languageLabel"><button type="button" class="language-button" data-language="zh-CN" aria-pressed="false">中文</button><button type="button" class="language-button" data-language="en" aria-pressed="false">EN</button></div></div>
      <div id="notice" role="status" aria-live="polite" hidden></div><button id="retry" type="button" class="button secondary" data-i18n="retry" hidden></button>
      <section id="transfer-view" class="view">
        <div class="page-heading"><div><h1 data-i18n="headline"></h1><p data-i18n="intro"></p></div></div>
        <div class="service-card"><div class="service-symbol">${icon('transfer')}</div><div class="service-copy"><div class="service-heading"><h2 data-i18n="service"></h2><span class="status-badge" id="status"></span></div><p id="service-hint"></p></div><button class="button primary" id="toggle-service"></button></div>
        <div class="transfer-grid">
          <article class="panel connection-panel"><div class="panel-heading"><div><h2 data-i18n="connect"></h2><p data-i18n="addresses"></p></div>${icon('wifi')}</div><div class="connection-visual">${bookIcon}<span data-i18n="connectionHint"></span></div><div id="addresses"></div><div class="network"><span data-i18n="network"></span><strong id="network-name"></strong></div><button class="button secondary full" id="browser" data-i18n="browser" hidden></button></article>
          <div class="right-column"><article class="panel directory-panel"><div class="panel-heading"><h2 data-i18n="folder"></h2><button class="text-button" data-view="settings" data-i18n="change"></button></div><div class="folder-visual">${icon('folder')}</div><strong id="directory-name"></strong><p id="directory-path" class="directory-path"></p><p class="helper" data-i18n="folderHint"></p><button id="open-directory" class="button secondary full">${icon('folder')}<span data-i18n="openFolder"></span></button><div class="direct-formats"><h3 data-i18n="directDownloadFormats"></h3><p>${directFormats}</p></div></article>${formatNotice('home-send-to-kindle', 'warning')}<article class="privacy-note">${icon('shield')}<div><h3 data-i18n="privacyTitle"></h3><p data-i18n="privacyHint"></p></div></article></div>
        </div>
        <div class="steps"><div><span>01</span><p data-i18n="stepOne"></p></div><div><span>02</span><p data-i18n="stepTwo"></p></div><div><span>03</span><p data-i18n="stepThree"></p></div></div>
      </section>
      <section id="settings-view" class="view" hidden>
        <div class="page-heading"><div><h1 data-i18n="settings"></h1><p data-i18n="settingsIntro"></p></div></div>
        <form id="settings-form" class="panel settings-panel"><fieldset id="settings-fields" disabled>
          <div class="settings-section"><label class="field-label" for="directory" data-i18n="folder"></label><p class="helper" data-i18n="folderHint"></p><div class="path-input"><input id="directory" name="directory" autocomplete="off" spellcheck="false" required><button type="button" class="button secondary" id="choose" data-i18n="choose"></button></div></div>
          <div class="settings-section"><label class="field-label" for="port" data-i18n="port"></label><div class="port-row"><input id="port" name="port" type="number" min="0" max="65535" step="1" required><p class="helper" data-i18n="portHint"></p></div></div>
          <div class="settings-section startup-note">${icon('info')}<p class="helper" data-i18n="startupHint"></p></div>
          <div class="settings-footer"><span id="settings-note" class="helper"></span><button id="save-settings" type="submit" class="button primary" data-i18n="save" disabled></button></div>
        </fieldset></form><p class="bottom-note" data-i18n="saveHint"></p>
      </section>
      <section id="about-view" class="view" hidden>
        <div class="page-heading"><div><h1 data-i18n="tagline"></h1><p data-i18n="openSource"></p></div></div>
        <article class="panel update-panel" aria-labelledby="update-heading">
          <div class="panel-heading"><div><h2 id="update-heading" data-i18n="appUpdates"></h2><p id="current-version"></p></div><button id="check-update" type="button" class="button secondary"></button></div>
          <p id="update-status" role="status" aria-live="polite"></p>
          <div id="update-details" hidden><h3 id="update-title"></h3><p id="update-notes"></p><button id="download-update" type="button" class="button primary" data-i18n="downloadUpdate"></button></div>
        </article>
        <article class="panel about-panel">
          <section class="about-section"><h2 data-i18n="connect"></h2><ol><li data-i18n="stepOne"></li><li data-i18n="stepTwo"></li><li data-i18n="stepThree"></li></ol></section>
          <section class="about-section about-formats"><h2 data-i18n="supportedFormats"></h2><p>${directFormats}</p>${formatNotice('send-to-kindle')}</section>
          <section class="about-section"><h2 data-i18n="troubleshooting"></h2><p data-i18n="connectionHelp"></p></section>
          <section class="about-section about-privacy"><h2 data-i18n="privacyTitle"></h2><p data-i18n="privacyHint"></p></section>
          <details class="logs"><summary data-i18n="logs"></summary><pre id="logs"></pre></details>
        </article>
      </section>
    </main>
    <footer><span id="footer-status"></span><span>macOS · Windows · Linux</span></footer>
  </div>`;
const $ = (selector) => app.querySelector(selector);
const notice = $('#notice');
function notify(message, error = false) {
  noticeMessage = message ? { message, error } : null;
  renderNotice();
}
function renderNotice() {
  notice.textContent = noticeMessage ? (noticeMessage.error ? translateError(lang, noticeMessage.message) : t(noticeMessage.message)) : '';
  notice.hidden = !noticeMessage;
  notice.className = noticeMessage?.error ? 'notice error' : 'notice';
}
function translate() {
  document.documentElement.lang = lang;
  document.title = t('appName');
  if (desktop) void getCurrentWindow().setTitle(t('appName')).catch(error => console.error(error));
  app.querySelectorAll('[data-i18n]').forEach((element) => { element.textContent = t(element.dataset.i18n); });
  app.querySelectorAll('[data-i18n-aria]').forEach(element => element.setAttribute('aria-label', t(element.dataset.i18nAria)));
  for (const button of app.querySelectorAll('[data-language]')) {
    button.classList.toggle('active', button.dataset.language === lang);
    button.setAttribute('aria-pressed', String(button.dataset.language === lang));
  }
  renderNotice(); renderLogs(); renderSettingsNote(); renderUpdates();
  renderStatus();
}
function renderUpdates() {
  $('#app-version').textContent = `v${currentVersion}`;
  $('#current-version').textContent = `${t('currentVersion')} v${currentVersion}`;
  $('#check-update').textContent = t(updateState === 'checking' ? 'checkingUpdate' : 'checkUpdate');
  $('#check-update').disabled = !desktop || updateState === 'checking' || openingUpdate;
  const available = updateState === 'done' && updateResult?.update_available;
  $('#update-details').hidden = !available;
  const messageKey = !desktop ? 'updateDesktopOnly' : updateState === 'checking' ? 'checkingUpdate'
    : updateState === 'error' ? 'updateFailed' : available ? (updateResult.force_update ? 'updateRequired' : 'updateAvailable')
    : updateState === 'done' ? 'upToDate' : 'updateHint';
  $('#update-status').textContent = `${t(messageKey)}${available ? ` v${updateResult.latest_version.replace(/^[vV]/, '')}` : ''}${updateState === 'error' ? ` ${translateError(lang, updateError)}` : ''}`;
  $('#update-status').classList.toggle('error', updateState === 'error' || Boolean(available && updateResult.force_update));
  $('#update-title').textContent = available ? updateResult.title : '';
  $('#update-title').hidden = !updateResult?.title;
  $('#update-notes').textContent = available ? updateResult.release_notes : '';
  $('#update-notes').hidden = !updateResult?.release_notes;
  $('#download-update').disabled = openingUpdate;
}
$('#check-update').addEventListener('click', async () => {
  if (!desktop || updateState === 'checking' || openingUpdate) return;
  updateState = 'checking'; updateResult = null; updateError = ''; renderUpdates();
  try {
    updateResult = await invoke('check_product_update');
    currentVersion = updateResult.current_version;
    updateState = 'done';
  } catch (error) {
    updateState = 'error'; updateError = String(error?.message || error);
  } finally { renderUpdates(); }
});
$('#download-update').addEventListener('click', async () => {
  if (!desktop || openingUpdate || !updateResult?.update_available) return;
  openingUpdate = true; renderUpdates();
  try { await invoke('open_product_update'); }
  catch (error) { updateError = String(error?.message || error); updateState = 'error'; }
  finally { openingUpdate = false; renderUpdates(); }
});
if (desktop) void getVersion().then(version => { currentVersion = version; renderUpdates(); }).catch(error => console.error(error));
function renderLogs() {
  $('#logs').textContent = serviceLogs.length ? serviceLogs.map(line => translateLog(lang, line)).join('\n') : t('noLogs');
}
function renderSettingsNote() {
  const dirty = settingsLoaded && ($('#directory').value.trim() !== status.settings.directory || $('#port').value !== String(status.settings.port));
  $('#settings-note').textContent = t(dirty ? 'unsaved' : 'settingsNote');
  $('#save-settings').disabled = !dirty || busy || !desktop || !ready;
}
function fillSettings() {
  const settings = status.settings;
  $('#directory').value = settings.directory;
  $('#port').value = settings.port;
  renderSettingsNote();
}
function readSettings() {
  return { directory: $('#directory').value.trim(), port: parsePort($('#port').value), language: lang };
}
function renderStatus() {
  renderSettingsNote();
  const next = JSON.stringify([status.running, status.addresses, status.networkName, status.settings.directory, lang, busy, desktop, ready, [...copiedAddresses.keys()]]);
  if (next === renderedStatus) return;
  renderedStatus = next;
  $('#status').textContent = t(status.running ? 'running' : 'stopped');
  $('#status').className = `status-badge ${status.running ? 'running' : ''}`;
  $('#service-hint').textContent = t(status.running ? 'activeHint' : 'inactiveHint');
  $('#toggle-service').textContent = t(!ready && desktop ? 'loading' : busy ? 'busy' : status.running ? 'stop' : 'start');
  $('#toggle-service').classList.toggle('stop-button', status.running);
  $('#toggle-service').disabled = busy || !desktop || !ready;
  $('#settings-fields').disabled = busy || !desktop || !ready;
  $('#open-directory').disabled = busy || !desktop || !ready;
  $('#directory-path').textContent = status.settings.directory;
  $('#directory-path').title = status.settings.directory;
  $('#directory-name').textContent = status.settings.directory.replace(/[\\/]+$/, '').split(/[\\/]/).pop() || '—';
  $('#footer-status').textContent = t(status.running ? 'running' : 'stopped');
  $('#browser').hidden = !status.running;
  $('#network-name').textContent = status.networkName || t('unknownNetwork');
  const addresses = $('#addresses');
  addresses.replaceChildren();
  if (status.running && status.addresses.length) {
    status.addresses.forEach((address) => {
      const row = document.createElement('div'); row.className = 'address-box';
      const code = document.createElement('code'); code.textContent = address;
      const feedback = document.createElement('span'); feedback.className = 'copy-feedback';
      feedback.setAttribute('role', 'status'); feedback.setAttribute('aria-live', 'polite');
      feedback.textContent = copiedAddresses.has(address) ? t('copied') : '';
      feedback.hidden = !copiedAddresses.has(address);
      const copy = document.createElement('button'); copy.className = 'copy-button'; copy.textContent = t('copy');
      copy.disabled = busy;
      copy.addEventListener('click', () => action(async () => {
        await writeText(address);
        clearTimeout(copiedAddresses.get(address));
        copiedAddresses.set(address, setTimeout(() => {
          copiedAddresses.delete(address);
          renderStatus();
        }, 2000));
      }));
      row.append(code, feedback, copy); addresses.append(row);
    });
  } else {
    const placeholder = document.createElement('p'); placeholder.className = 'address-placeholder'; placeholder.textContent = t(status.running ? 'noNetwork' : 'noAddress'); addresses.append(placeholder);
  }
}
async function action(callback) {
  if (busy) return;
  revision += 1;
  busy = true; renderStatus();
  try { await callback(); } catch (error) { notify(String(error?.message || error), true); }
  finally { busy = false; renderStatus(); }
}
$('#settings-form').addEventListener('submit', (event) => {
  event.preventDefault();
  if ($('#save-settings').disabled) return;
  action(async () => {
    try {
      status = await request('save', readSettings());
      fillSettings(); notify(status.running ? 'savedRestarted' : 'saved');
    } catch (error) {
      // A failed restart may recover the old settings; keep the draft for correction.
      status = await request('status'); renderSettingsNote();
      throw error;
    }
  });
});
$('#toggle-service').addEventListener('click', () => action(async () => {
  notify('');
  if (status.running) status = await request('stop');
  else {
    // Starting always applies the form, so an unsaved folder/port isn't silently ignored.
    status = await request('save', readSettings()); fillSettings();
    status = await request('start');
  }
}));
$('#choose').addEventListener('click', () => action(async () => {
  const chosen = await open({ title: t('choose'), directory: true, multiple: false, defaultPath: $('#directory').value || undefined });
  if (chosen) { $('#directory').value = chosen; renderSettingsNote(); }
}));
for (const button of app.querySelectorAll('[data-language]')) button.addEventListener('click', () => {
  lang = button.dataset.language;
  const currentRevision = ++languageRevision;
  rememberLanguage(storage, lang); translate();
  if (desktop) {
    const language = lang;
    languageSave = languageSave.then(async () => {
      try {
        const next = await request('language', { language });
        if (currentRevision === languageRevision) status.settings.language = next.settings.language;
      } catch (error) { notify(error, true); }
    });
  }
});
$('#settings-form').addEventListener('input', renderSettingsNote);
$('#github').addEventListener('click', (event) => {
  if (!desktop) return;
  event.preventDefault(); action(() => openUrl(event.currentTarget.href));
});
$('#feedback').addEventListener('click', async (event) => {
  if (!desktop) return;
  event.preventDefault();
  if (openingFeedback) return;
  openingFeedback = true;
  // Feedback remains available while the service is starting.
  try { await invoke('open_product_feedback', { language: lang }); }
  catch (error) { notify(error, true); }
  finally { openingFeedback = false; }
});
$('#browser').addEventListener('click', () => action(() => openUrl(status.controlUrl)));
function showView(next) {
  view = next;
  for (const section of app.querySelectorAll('.view')) section.hidden = section.id !== `${view}-view`;
  for (const button of app.querySelectorAll('.nav-item[data-view]')) button.classList.toggle('active', button.dataset.view === view);
}
for (const button of app.querySelectorAll('[data-view]')) button.addEventListener('click', () => showView(button.dataset.view));
$('#open-directory').addEventListener('click', () => action(() => openPath(status.settings.directory)));
for (const link of app.querySelectorAll('[data-send-to-kindle]')) link.addEventListener('click', (event) => {
  if (!desktop) return;
  event.preventDefault();
  action(() => openUrl(link.href));
});
$('.logs').addEventListener('toggle', async () => {
  if ($('.logs').open && desktop) {
    try { serviceLogs = await request('logs'); renderLogs(); }
    catch (error) { $('#logs').textContent = translateError(lang, error); }
  }
});
translate();
async function initialize() {
  if (!desktop) { notify('preview'); return; }
  const bootLanguageRevision = languageRevision;
  await action(async () => {
    try {
      status = await request('status'); ready = true;
      if (!settingsLoaded) {
        settingsLoaded = true;
        if (bootLanguageRevision === languageRevision) lang = initialLanguage(storage, navigator.language, status.settings.language);
        fillSettings(); translate();
      }
      $('#retry').hidden = true;
      notify(status.lastError || '', Boolean(status.lastError));
    } catch (error) { $('#retry').hidden = false; throw error; }
  });
}
async function poll() {
  if (!desktop || !ready || busy || polling || document.hidden) return;
  polling = true;
  const currentRevision = revision;
  try {
    const next = await request('status');
    if (currentRevision !== revision) return;
    const changedError = next.lastError && next.lastError !== status.lastError;
    status = next;
    if (changedError) notify(status.lastError, true);
    renderStatus(); renderSettingsNote();
    if ($('.logs').open) { serviceLogs = await request('logs'); renderLogs(); }
  } catch (error) {
    if (currentRevision !== revision) return;
    status.running = false; status.addresses = []; status.controlUrl = '';
    notify(error, true); renderStatus();
  } finally { polling = false; }
}
$('#retry').addEventListener('click', initialize);
void initialize();
if (desktop) setInterval(poll, 4000);
