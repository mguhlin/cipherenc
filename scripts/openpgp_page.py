"""Derive the key-focused page from the shared workspace, keeping one UI source."""
import re
from pathlib import Path

def build(index):
 page=index.replace('<body>','<body data-cipherenc-mode="openpgp">',1)
 page=re.sub(r'<title>.*?</title>','<title>CipherEnc OpenPGP — Public and private keys</title>',page,count=1)
 page=page.replace('CipherEnc encrypts files, folders, and text locally with AES-256-GCM.','Encrypt with existing OpenPGP public keys or decrypt with your private key, locally on your device.')
 page=page.replace('<h1 id="workspaceTitle">Secure workspace</h1>','<h1 id="workspaceTitle">OpenPGP workspace</h1>')
 page=re.sub(r'<p class="workspace-directions">.*?</p>','<p class="workspace-directions">Directions: Follow these three steps to safeguard your data.</p>\n      <p class="workspace-mode-links"><a class="secondary mode-link" href="index.html">Use a password instead</a><a href="help.html#openpgp-keys" target="_blank" rel="noopener">How to add and use your keys</a></p>',page,count=1)
 controls=Path(__file__).with_name('templates').joinpath('openpgp-controls.html').read_text()
 key_start=controls.index('            <div id="pgpCredentials"');key_end=controls.index('            </details>',key_start)
 keys=controls[key_start:key_end].replace('class="pgp-credentials hidden"','class="pgp-credentials"')
 start=page.index('            <div id="passwordCredentials">');end=page.index('            <h3 class="step-heading"><span class="step-number">2',start)
 password=page[start:end].replace('id="passwordCredentials"','id="passwordCredentials" class="hidden"',1)
 password=re.sub(r'<svg class="step-icon".*?</svg>','',password,count=1)
 key_step='''<section id="openpgpKeyStep"><h3 class="step-heading"><span class="step-number">1</span><svg class="step-icon" viewBox="0 0 32 32" aria-hidden="true"><circle cx="11" cy="12" r="7"/><path d="m16 17 11 11m-6-6 3-3m0 6 3-3"/></svg><span>Your OpenPGP Key</span></h3><p id="pgpActionHelp" class="step-hint">Public key to encrypt. Matching private key to decrypt.</p>'''+keys+'</section>'
 page=page[:start]+key_step+password+page[end:]
 page=page.replace('Choose files/folders or text. CipherEnc detects what to do automatically.','Choose one file or paste text. An encrypted OpenPGP message is detected automatically.')
 page=page.replace('2a) Drop files or folders here','2a) Drop one file here')
 page=page.replace('Originals are zipped and encrypted. Encrypted packages are unlocked.','Encrypt a file for a public-key holder, or unlock an OpenPGP message with your private key.')
 return page

def android(index,help_page):
 controls=Path(__file__).with_name('templates').joinpath('openpgp-controls.html').read_text()
 index=index.replace('<button id="runButton"',controls+'\n<button id="runButton"',1)
 sections=[]
 for name in ['browser-help','openpgp-keys']:
  start=help_page.index('    <section id="'+name+'"') if '    <section id="'+name+'"' in help_page else help_page.index('<section id="'+name+'"')
  end=help_page.index('</section>',start)+len('</section>');sections.append(help_page[start:end])
 index=index.replace('</main>','\n'.join(sections)+'\n</main>',1)
 return index.replace('href="help.html#openpgp-keys"','href="#openpgp-keys"').replace('href="help.html"','href="#browser-help"').replace(' target="_blank"','')

if __name__=='__main__':
 root=Path(__file__).resolve().parents[1]/'docs/encrypt'
 print(android((root/'index.html').read_text(),(root/'help.html').read_text()),end='')
