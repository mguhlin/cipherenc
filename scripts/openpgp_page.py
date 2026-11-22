"""Derive the key-focused page from the shared workspace, keeping one UI source."""
import re

def build(index):
 page=index.replace('<body>','<body data-cipherenc-mode="openpgp">',1)
 page=re.sub(r'<title>.*?</title>','<title>CipherEnc OpenPGP — Public and private keys</title>',page,count=1)
 page=page.replace('CipherEnc encrypts files, folders, and text locally with AES-256-GCM.','Encrypt with existing OpenPGP public keys or decrypt with your private key, locally on your device.')
 page=page.replace('<h2 id="workspaceTitle">Secure workspace</h2>','<h2 id="workspaceTitle">OpenPGP workspace</h2>')
 page=re.sub(r'<p class="workspace-mode-links">.*?</p>','<p class="workspace-mode-links"><a class="secondary mode-link" href="index.html">Use a password instead</a><a href="#key-help">How to add and use your keys</a></p>',page,count=1)
 key_start=page.index('            <div id="pgpCredentials"');key_end=page.index('            </details>',key_start)
 keys=page[key_start:key_end].replace('class="pgp-credentials hidden"','class="pgp-credentials"')
 page=page[:key_start]+page[key_end:]
 page=page.replace('<details id="advanced-keys" class="advanced-tools">','<details id="advanced-keys" class="advanced-tools" hidden>')
 start=page.index('            <div id="passwordCredentials">');end=page.index('            <h3 class="step-heading"><span class="step-number">2',start)
 password=page[start:end].replace('id="passwordCredentials"','id="passwordCredentials" class="hidden"',1)
 password=re.sub(r'<svg class="step-icon".*?</svg>','',password,count=1)
 key_step='''<section id="openpgpKeyStep"><h3 class="step-heading"><span class="step-number">1</span><svg class="step-icon" viewBox="0 0 32 32" aria-hidden="true"><circle cx="11" cy="12" r="7"/><path d="m16 17 11 11m-6-6 3-3m0 6 3-3"/></svg><span>Your OpenPGP Key</span></h3><p id="pgpActionHelp" class="step-hint">Public key to encrypt. Matching private key to decrypt.</p>'''+keys+'</section>'
 page=page[:start]+key_step+password+page[end:]
 page=page.replace('Choose files/folders or text. CipherEnc detects what to do automatically.','Choose one file or paste text. An encrypted OpenPGP message is detected automatically.')
 page=page.replace('2a) Drop files or folders here','2a) Drop one file here')
 page=page.replace('Originals are zipped and encrypted. Encrypted packages are unlocked.','Encrypt a file for a public-key holder, or unlock an OpenPGP message with your private key.')
 page=page.replace('Protect a file, folder, or private note.','Protect with public and private keys.')
 page=page.replace('Browser workspace</p>','OpenPGP workspace</p>')
 page=page.replace('Enter a strong password first, then drop your files to start. Or choose files and press the action button. Encrypted files download automatically; your browser controls the save location. On phones and tablets, start with a file or text; folder selection depends on the browser and its file picker.','Paste or import a public key to encrypt, or your private key to decrypt. No vault password is needed. If your private key is protected, its passphrase unlocks it locally. Text results can be copied on screen.')
 help='''<section id="browser-help" class="wrap browser-help" aria-labelledby="keyHelpTitle">
  <span id="key-help"></span><h2 id="keyHelpTitle">OpenPGP FAQs</h2>
  <details open><summary>Which key should I use?</summary><ul>
    <li><strong>To encrypt:</strong> use the recipient’s public key. Compare its fingerprint with them through another trusted channel.</li>
    <li><strong>To unlock:</strong> use the private key matching the public key that encrypted the message.</li>
    <li>If the private key is protected, enter its private-key passphrase too.</li>
  </ul></details>
  <details><summary>How do I add my existing key?</summary><ul>
    <li>Export the key from your OpenPGP key manager.</li>
    <li>In Step 1, paste the complete public/private key block or choose Import key file for its .asc/.pgp export.</li>
    <li>Check the displayed fingerprint, then add your file or text in Step 2.</li>
    <li>Import a key file or paste its contents; a key-server web address will not work here.</li>
  </ul></details>
  <details><summary>Can I encrypt folders or several files?</summary><ul>
    <li>This workspace handles one file or text message at a time. ZIP a folder or group of files first.</li>
    <li>Use the <a href="index.html">password workspace</a> for automatic file/folder packaging and restoration.</li>
  </ul></details>
  <details><summary>Where is my result?</summary><ul>
    <li>Encrypted files download as .cen; your browser controls the save location.</li>
    <li>Encrypted and decrypted text appear on screen with a Copy button.</li>
    <li>These are standard OpenPGP messages. Other OpenPGP tools may expect a .gpg or .asc filename ending.</li>
  </ul></details>
  <details><summary>Are my keys saved or linked to an account?</summary><ul>
    <li>No. Keys are used in this tab and clear after success, Clear or inactivity. Import or paste them again for another operation.</li>
    <li>CipherEnc does not create key pairs, fetch keys from servers or link to an account.</li>
    <li>Keep your private key private and backed up. CipherEnc cannot recover a lost key or passphrase.</li>
  </ul></details>
  <details><summary>Does my data leave my device?</summary><ul>
    <li>Files, text, keys and passphrases are processed locally in this tab.</li>
    <li>The website receives normal page and asset requests.</li>
    <li>Your browser, downloads, clipboard, backups and extensions remain under your device’s controls.</li>
  </ul></details>
  <details><summary>Can I use this key in the desktop app?</summary><ul>
    <li>OpenPGP is available in the browser and Android app. The native desktop apps use password encryption.</li>
    <li>For a password-protected .cen file, open the <a href="index.html">password workspace</a>.</li>
  </ul></details>
</section>'''
 start=page.index('    <section id="browser-help"');end=page.index('</section>',start)+len('</section>')
 page=page[:start]+help+page[end:]
 return page
