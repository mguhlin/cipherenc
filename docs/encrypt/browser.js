'use strict';
let workspaceBusy = false;
const controls = () => [...document.querySelectorAll('button,input,textarea,select')];
async function runWorkspaceTask(task) {
  if (workspaceBusy) return;
  workspaceBusy = true;
  const list = controls(), previous = list.map(control => control.disabled);
  list.forEach(control => control.disabled = true);
  const monitor = document.querySelector('.status-panel');
  monitor.setAttribute('aria-busy', 'true');
  try { await task(); }
  catch (error) { els.error.textContent = error.message || 'The operation could not be completed.'; }
  finally {
    list.forEach((control, index) => control.disabled = previous[index]);
    workspaceBusy = false;
    monitor.setAttribute('aria-busy', 'false');
    sync();
  }
}
document.getElementById('runButton').addEventListener('click', async event => {
  event.stopImmediatePropagation();
  await runWorkspaceTask(runRequested);
}, true);
document.getElementById('downloadResult').addEventListener('click', async event => {
  event.stopImmediatePropagation();
  await runWorkspaceTask(downloadResult);
}, true);
document.addEventListener('drop', event => {
  if (workspaceBusy) { event.preventDefault(); event.stopImmediatePropagation(); }
}, true);
document.getElementById('clearWorkspace').addEventListener('click', () => {
  if (!workspaceBusy) { clearSensitiveMaterial(); document.getElementById('clearWorkspace').focus(); }
});
