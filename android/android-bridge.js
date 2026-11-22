'use strict';
(() => {
  if (!window.CipherEncAndroid) return;
  const bridge = window.CipherEncAndroid;
  const MAX_INPUT = 16 * 1048576;
  const MAX_PACKAGE = 24 * 1048576;
  let saving = false;
  let pendingSave = null;
  let selectionToken = 0;
  let savedControls = null;
  const note = document.createElement('p');
  note.className = 'android-note';
  note.textContent = 'Offline Android evaluation app · files and folders up to 16 MiB total · encrypted inputs up to 24 MiB. This app uses Android’s built-in WebView; no external browser or internet connection is required.';
  document.querySelector('main')?.prepend(note);
  document.querySelectorAll('a[href]').forEach(link => {
    const href = link.getAttribute('href');
    if (/^(https?:|\.\.\/)/i.test(href)) {
      link.removeAttribute('href');
      link.setAttribute('aria-disabled', 'true');
    }
  });
  const folderButton = document.getElementById('chooseFolder');
  folderButton?.addEventListener('click', event => {
    event.preventDefault(); event.stopImmediatePropagation();
    bridge.pickFolder(String(++selectionToken));
  }, true);
  document.getElementById('clearWorkspace')?.addEventListener('click', () => { selectionToken++; }, true);
  document.getElementById('chooseFiles')?.addEventListener('click', () => { selectionToken++; }, true);
  document.querySelectorAll('.mode-tab,.op,.method').forEach(button => button.addEventListener('click', () => { selectionToken++; }, true));
  // Ancestor capture runs before the browser wrapper's button capture handler.
  document.addEventListener('click', event => {
    if (!event.target.closest?.('#runButton')) return;
    const limit = operation === 'encrypt' ? MAX_INPUT : MAX_PACKAGE;
    const bytes = mode === 'text' ? new TextEncoder().encode(els.text.value).length : selectedEntries.reduce((sum, entry) => sum + entry.file.size, 0);
    if (bytes > limit || selectedEntries.length > 1000) {
      event.preventDefault(); event.stopImmediatePropagation();
      els.error.textContent = 'Android evaluation limit: 16 MiB to encrypt, 24 MiB encrypted input, and 1,000 files. Use the browser or desktop edition for larger selections.';
    }
  }, true);
  const toBase64 = bytes => {
    let text = '';
    for (let i = 0; i < bytes.length; i += 32768) text += String.fromCharCode(...bytes.subarray(i, i + 32768));
    return btoa(text);
  };
  window.CipherEncAndroidResult = (success, message) => {
    saving = false;
    if (pendingSave) { pendingSave({success, message}); pendingSave = null; }
    if (savedControls) {
      savedControls.forEach(({control, disabled}) => { control.disabled = disabled; });
      savedControls = null;
      sync();
    }
    const status = document.getElementById('statusCopy');
    if (status) status.textContent = message;
    if (!success && els?.error) els.error.textContent = message;
  };
  window.CipherEncAndroidFolder = (token, files) => {
    if (String(selectionToken) !== token || !Array.isArray(files) || files.length > 1000) return;
    const entries = files.map(item => {
      const data = Uint8Array.from(atob(item.data), char => char.charCodeAt(0));
      const file = new File([data], item.path.split('/').at(-1), {type:'application/octet-stream'});
      return {file, path:item.path};
    });
    return setEntries(entries);
  };
  window.CipherEncAndroidSave = async () => {
    if (saving) return;
    saving = true;
    const completion = new Promise(resolve => { pendingSave = resolve; });
    savedControls = [...document.querySelectorAll('button,input,textarea,select')].map(control => ({control, disabled:control.disabled}));
    savedControls.forEach(({control}) => { control.disabled = true; });
    try {
      if (resultFiles.length) {
        const total = resultFiles.reduce((sum, file) => sum + file.blob.size, 0);
        if (total > MAX_INPUT || resultFiles.length > 1000) throw Error('Restored folder exceeds the Android evaluation limit.');
        const files = [];
        for (const file of resultFiles) files.push({path:file.path, data:toBase64(new Uint8Array(await file.blob.arrayBuffer()))});
        bridge.saveFolder(JSON.stringify(files));
      } else {
        if (!resultBlob) throw Error('There is no downloadable result.');
        if (resultBlob.size > MAX_PACKAGE) throw Error('Result exceeds the Android evaluation limit.');
        bridge.saveFile(resultName || 'encrypted.cen', toBase64(new Uint8Array(await resultBlob.arrayBuffer())));
      }
    } catch (error) { window.CipherEncAndroidResult(false, error.message); }
    return completion;
  };
  document.addEventListener('click', async event => {
    if (!event.target.closest?.('#downloadResult')) return;
    event.preventDefault(); event.stopImmediatePropagation();
    await window.CipherEncAndroidSave();
  }, true);
})();
