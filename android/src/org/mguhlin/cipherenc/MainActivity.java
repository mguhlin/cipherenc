package org.mguhlin.cipherenc;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.ActivityNotFoundException;
import android.content.Intent;
import android.database.Cursor;
import android.graphics.Bitmap;
import android.net.Uri;
import android.os.Bundle;
import android.provider.DocumentsContract;
import android.provider.OpenableColumns;
import android.util.Base64;
import android.webkit.JavascriptInterface;
import android.webkit.ValueCallback;
import android.webkit.WebChromeClient;
import android.webkit.WebResourceRequest;
import android.webkit.WebResourceResponse;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.Toast;
import org.json.JSONArray;
import org.json.JSONObject;
import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.io.OutputStream;
import java.nio.charset.StandardCharsets;
import java.security.SecureRandom;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public final class MainActivity extends Activity {
    private static final String ORIGIN = "https://appassets.androidplatform.net";
    private static final String PAGE = ORIGIN + "/assets/index.html";
    private static final int OPEN = 10, PICK_FOLDER = 11, SAVE_FILE = 12, SAVE_FOLDER = 13;
    private static final int INPUT_LIMIT = 16 * 1024 * 1024, PACKAGE_LIMIT = 24 * 1024 * 1024, MAX_FILES = 1000;
    private WebView web;
    private volatile boolean trustedPage;
    private final ExecutorService worker = Executors.newSingleThreadExecutor();
    private ValueCallback<Uri[]> chooser;
    private byte[] pendingFile;
    private List<ExportFile> pendingFolder;
    private boolean nativeOperation;
    private String folderToken;

    private static final class ExportFile {
        final String path; final byte[] data;
        ExportFile(String path, byte[] data) { this.path = path; this.data = data; }
    }

    @Override public void onCreate(Bundle state) {
        super.onCreate(state);
        web = new WebView(this);
        web.setBackgroundColor(0xfffbfaff);
        web.setFitsSystemWindows(true);
        WebView.setWebContentsDebuggingEnabled(false);
        WebSettings settings = web.getSettings();
        settings.setJavaScriptEnabled(true);
        settings.setDomStorageEnabled(false);
        settings.setDatabaseEnabled(false);
        settings.setAllowFileAccess(false);
        settings.setAllowContentAccess(true); // Only content URIs selected through the Android picker.
        settings.setAllowFileAccessFromFileURLs(false);
        settings.setAllowUniversalAccessFromFileURLs(false);
        settings.setMixedContentMode(WebSettings.MIXED_CONTENT_NEVER_ALLOW);
        settings.setJavaScriptCanOpenWindowsAutomatically(false);
        settings.setSupportMultipleWindows(false);
        settings.setCacheMode(WebSettings.LOAD_NO_CACHE);
        settings.setSaveFormData(false);
        settings.setMediaPlaybackRequiresUserGesture(true);
        web.addJavascriptInterface(new Bridge(), "CipherEncAndroid");
        web.setWebViewClient(new LocalClient());
        web.setWebChromeClient(new WebChromeClient() {
            @Override public boolean onShowFileChooser(WebView view, ValueCallback<Uri[]> callback, FileChooserParams params) {
                if (!trustedPage || nativeOperation) { callback.onReceiveValue(null); return true; }
                if (chooser != null) chooser.onReceiveValue(null);
                chooser = callback;
                Intent intent = new Intent(Intent.ACTION_OPEN_DOCUMENT).addCategory(Intent.CATEGORY_OPENABLE).setType("*/*");
                intent.putExtra(Intent.EXTRA_ALLOW_MULTIPLE, params.getMode() == FileChooserParams.MODE_OPEN_MULTIPLE);
                try { startActivityForResult(intent, OPEN); }
                catch (ActivityNotFoundException error) { chooser.onReceiveValue(null); chooser = null; notifyResult(false, "No Android document picker is available."); }
                return true;
            }
        });
        setContentView(web);
        web.loadUrl(PAGE);
    }

    private final class LocalClient extends WebViewClient {
        @Override public void onPageStarted(WebView view, String url, Bitmap icon) { trustedPage = isPage(url); }
        @Override public boolean shouldOverrideUrlLoading(WebView view, WebResourceRequest request) { return !isPage(request.getUrl().toString()); }
        @Override public boolean shouldOverrideUrlLoading(WebView view, String url) { return !isPage(url); }
        @Override public WebResourceResponse shouldInterceptRequest(WebView view, WebResourceRequest request) { return asset(request.getUrl()); }
        @Override public WebResourceResponse shouldInterceptRequest(WebView view, String url) { return asset(Uri.parse(url)); }
    }

    private static boolean isPage(String value) {
        Uri uri = Uri.parse(value);
        return "https".equals(uri.getScheme()) && "appassets.androidplatform.net".equals(uri.getHost()) && "/assets/index.html".equals(uri.getPath()) && uri.getPort() == -1;
    }

    private WebResourceResponse asset(Uri uri) {
        try {
            if (!"https".equals(uri.getScheme()) || !"appassets.androidplatform.net".equals(uri.getHost()) || uri.getPort() != -1) return denied();
            String path = uri.getPath();
            if (path == null || !path.startsWith("/assets/")) return denied();
            String name = path.substring(8);
            if (!safePath(name)) return denied();
            String mime = name.endsWith(".js") ? "application/javascript" : name.endsWith(".css") ? "text/css" : name.endsWith(".html") ? "text/html" : name.endsWith(".svg") ? "image/svg+xml" : name.endsWith(".png") ? "image/png" : name.endsWith(".webp") ? "image/webp" : name.endsWith(".json") || name.endsWith(".webmanifest") ? "application/json" : "application/octet-stream";
            InputStream input = getAssets().open("web/" + name);
            Map<String,String> headers = new HashMap<>();
            headers.put("Cache-Control", "no-store");
            headers.put("X-Content-Type-Options", "nosniff");
            headers.put("Referrer-Policy", "no-referrer");
            headers.put("Content-Security-Policy", "default-src 'self'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self'; img-src 'self' data:; font-src 'self'; worker-src 'self'; connect-src 'none'; frame-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'");
            return new WebResourceResponse(mime, "UTF-8", 200, "OK", headers, input);
        } catch (Exception error) { return denied(); }
    }

    private static WebResourceResponse denied() {
        return new WebResourceResponse("text/plain", "UTF-8", 404, "Not Found", new HashMap<>(), new ByteArrayInputStream(new byte[0]));
    }

    private static boolean safePath(String name) {
        if (name == null || name.isEmpty() || name.length() > 4096 || name.indexOf('\\') >= 0 || name.indexOf(':') >= 0) return false;
        String[] parts = name.split("/", -1);
        if (parts.length > 32) return false;
        for (String part : parts) {
            if (part.isEmpty() || part.equals(".") || part.equals("..") || part.length() > 255) return false;
            for (int i = 0; i < part.length(); i++) if (Character.isISOControl(part.charAt(i))) return false;
        }
        return true;
    }

    private final class Bridge {
        @JavascriptInterface public void pickFolder(String token) {
            if (!trustedPage || token == null || !token.matches("[0-9]{1,16}")) return;
            runOnUiThread(() -> {
                if (nativeOperation) { notifyResult(false, "Another Android operation is in progress."); return; }
                nativeOperation = true;
                folderToken = token;
                try { startActivityForResult(new Intent(Intent.ACTION_OPEN_DOCUMENT_TREE), PICK_FOLDER); }
                catch (ActivityNotFoundException error) { nativeOperation = false; notifyResult(false, "No Android folder picker is available."); }
            });
        }
        @JavascriptInterface public void saveFile(String name, String encoded) {
            if (!trustedPage) return;
            worker.execute(() -> {
                try {
                    if (!safePath(name) || name.indexOf('/') >= 0 || encoded == null || encoded.length() > ((PACKAGE_LIMIT + 2) / 3) * 4 + 8) throw new Exception("Invalid or oversized Android save request.");
                    byte[] data = Base64.decode(encoded, Base64.NO_WRAP);
                    if (data.length > PACKAGE_LIMIT) throw new Exception("Result exceeds 24 MiB.");
                    runOnUiThread(() -> {
                        if (nativeOperation) { Arrays.fill(data, (byte) 0); notifyResult(false, "Another Android operation is in progress."); return; }
                        nativeOperation = true; pendingFile = data;
                        Intent intent = new Intent(Intent.ACTION_CREATE_DOCUMENT).addCategory(Intent.CATEGORY_OPENABLE).setType("application/octet-stream").putExtra(Intent.EXTRA_TITLE, name);
                        try { startActivityForResult(intent, SAVE_FILE); }
                        catch (ActivityNotFoundException error) { clearPending(); notifyResult(false, "No Android save picker is available."); }
                    });
                } catch (Exception error) { notifyResult(false, "The result could not be prepared for saving. Check its name and size."); }
            });
        }
        @JavascriptInterface public void saveFolder(String encoded) {
            if (!trustedPage) return;
            worker.execute(() -> {
                List<ExportFile> files = new ArrayList<>();
                try {
                    if (encoded == null || encoded.length() > 25 * 1024 * 1024) throw new Exception("Oversized folder request.");
                    JSONArray entries = new JSONArray(encoded);
                    if (entries.length() == 0 || entries.length() > MAX_FILES) throw new Exception("Invalid file count.");
                    int total = 0; Set<String> names = new HashSet<>();
                    for (int i = 0; i < entries.length(); i++) {
                        JSONObject entry = entries.getJSONObject(i); String name = entry.getString("path");
                        if (!safePath(name) || !names.add(name.toLowerCase(Locale.ROOT))) throw new Exception("Unsafe or duplicate folder path.");
                        String data = entry.getString("data");
                        if (data.length() > ((INPUT_LIMIT + 2) / 3) * 4 + 8) throw new Exception("Oversized folder item.");
                        byte[] bytes = Base64.decode(data, Base64.NO_WRAP); total += bytes.length;
                        if (total > INPUT_LIMIT) { Arrays.fill(bytes, (byte) 0); throw new Exception("Folder exceeds 16 MiB."); }
                        files.add(new ExportFile(name, bytes));
                    }
                    runOnUiThread(() -> {
                        if (nativeOperation) { wipe(files); notifyResult(false, "Another Android operation is in progress."); return; }
                        nativeOperation = true; pendingFolder = files;
                        try { startActivityForResult(new Intent(Intent.ACTION_OPEN_DOCUMENT_TREE), SAVE_FOLDER); }
                        catch (ActivityNotFoundException error) { clearPending(); notifyResult(false, "No Android folder picker is available."); }
                    });
                } catch (Exception error) { wipe(files); notifyResult(false, "Folder could not be prepared. Check paths, file count and the 16 MiB limit."); }
            });
        }
    }

    @Override protected void onActivityResult(int request, int result, Intent intent) {
        super.onActivityResult(request, result, intent);
        if (request == OPEN) {
            ValueCallback<Uri[]> callback = chooser; chooser = null;
            if (callback == null) return;
            if (result != RESULT_OK || intent == null) { callback.onReceiveValue(null); return; }
            List<Uri> uris = new ArrayList<>();
            if (intent.getClipData() != null) for (int i = 0; i < intent.getClipData().getItemCount(); i++) uris.add(intent.getClipData().getItemAt(i).getUri());
            else if (intent.getData() != null) uris.add(intent.getData());
            if (uris.size() > MAX_FILES) { callback.onReceiveValue(null); notifyResult(false, "Choose no more than 1,000 files."); return; }
            callback.onReceiveValue(uris.toArray(new Uri[0])); return;
        }
        if (request != PICK_FOLDER && request != SAVE_FILE && request != SAVE_FOLDER) return;
        if (result != RESULT_OK || intent == null || intent.getData() == null) { clearPending(); notifyResult(true, "Android picker canceled. The current result is still available."); return; }
        Uri selected = intent.getData();
        final String selectedToken = folderToken;
        worker.execute(() -> {
            try {
                if (request == PICK_FOLDER) {
                    JSONArray files = new JSONArray(); int[] total = {0, 0};
                    Uri root = DocumentsContract.buildDocumentUriUsingTree(selected, DocumentsContract.getTreeDocumentId(selected));
                    String name = documentName(root);
                    collectFolder(selected, root, name, files, total, 0);
                    if (files.length() == 0) throw new Exception("The folder contains no regular files.");
                    String payload = files.toString();
                    runOnUiThread(() -> { nativeOperation = false; folderToken = null; if (trustedPage) web.evaluateJavascript("window.CipherEncAndroidFolder(" + JSONObject.quote(selectedToken) + "," + payload + ")", null); });
                    notifyResult(true, "Folder loaded locally. Empty directories are not recorded.");
                } else if (request == SAVE_FILE) {
                    byte[] bytes = pendingFile;
                    if (bytes == null) throw new Exception("The pending result is unavailable.");
                    try (OutputStream output = getContentResolver().openOutputStream(selected, "w")) {
                        if (output == null) throw new Exception("The selected destination could not be opened.");
                        output.write(bytes); output.flush();
                    }
                    runOnUiThread(this::clearPending); notifyResult(true, "Result saved to the location you selected.");
                } else {
                    List<ExportFile> files = pendingFolder;
                    if (files == null) throw new Exception("The pending folder is unavailable.");
                    restoreFolder(selected, files);
                    runOnUiThread(this::clearPending); notifyResult(true, "Folder restored in a new CipherEnc-restored subfolder. Existing files were not replaced.");
                }
            } catch (Exception error) { runOnUiThread(this::clearPending); notifyResult(false, "Android file operation failed. No success is claimed. Try a local folder or a smaller selection."); }
        });
    }

    private void collectFolder(Uri tree, Uri folder, String prefix, JSONArray files, int[] total, int depth) throws Exception {
        if (depth > 24 || !safePath(prefix)) throw new Exception("Folder is too deep or has unsupported names.");
        Uri children = DocumentsContract.buildChildDocumentsUriUsingTree(tree, DocumentsContract.getDocumentId(folder));
        try (Cursor cursor = getContentResolver().query(children, new String[]{DocumentsContract.Document.COLUMN_DOCUMENT_ID, DocumentsContract.Document.COLUMN_DISPLAY_NAME, DocumentsContract.Document.COLUMN_MIME_TYPE}, null, null, null)) {
            if (cursor == null) throw new Exception("Folder cannot be read.");
            while (cursor.moveToNext()) {
                if (++total[1] > 5000) throw new Exception("Folder contains too many directory entries.");
                String id = cursor.getString(0), name = cursor.getString(1), mime = cursor.getString(2);
                if (!safePath(name) || name.indexOf('/') >= 0) throw new Exception("Unsupported filename.");
                Uri child = DocumentsContract.buildDocumentUriUsingTree(tree, id); String path = prefix + "/" + name;
                if (DocumentsContract.Document.MIME_TYPE_DIR.equals(mime)) collectFolder(tree, child, path, files, total, depth + 1);
                else {
                    if (files.length() >= MAX_FILES) throw new Exception("Too many files.");
                    byte[] data = boundedRead(getContentResolver().openInputStream(child), INPUT_LIMIT - total[0]); total[0] += data.length;
                    JSONObject file = new JSONObject().put("path", path).put("data", Base64.encodeToString(data, Base64.NO_WRAP));
                    files.put(file); Arrays.fill(data, (byte) 0);
                }
            }
        }
    }

    private String documentName(Uri uri) throws Exception {
        try (Cursor cursor = getContentResolver().query(uri, new String[]{OpenableColumns.DISPLAY_NAME}, null, null, null)) {
            if (cursor != null && cursor.moveToFirst()) { String name = cursor.getString(0); if (safePath(name) && name.indexOf('/') < 0) return name; }
        }
        return "folder";
    }

    private void restoreFolder(Uri tree, List<ExportFile> files) throws Exception {
        Uri parent = DocumentsContract.buildDocumentUriUsingTree(tree, DocumentsContract.getTreeDocumentId(tree));
        byte[] random = new byte[6]; new SecureRandom().nextBytes(random);
        StringBuilder suffix = new StringBuilder(); for (byte item : random) suffix.append(String.format(Locale.ROOT, "%02x", item));
        Uri root = DocumentsContract.createDocument(getContentResolver(), parent, DocumentsContract.Document.MIME_TYPE_DIR, "CipherEnc-restored-" + suffix);
        if (root == null) throw new Exception("Could not create restoration folder.");
        Map<String,Uri> directories = new HashMap<>(); directories.put("", root); boolean success = false;
        try {
            for (ExportFile file : files) {
                if (!safePath(file.path)) throw new Exception("Unsafe path.");
                String[] parts = file.path.split("/"); String current = ""; Uri directory = root;
                for (int i = 0; i < parts.length - 1; i++) {
                    current = current.isEmpty() ? parts[i] : current + "/" + parts[i];
                    Uri next = directories.get(current);
                    if (next == null) { next = DocumentsContract.createDocument(getContentResolver(), directory, DocumentsContract.Document.MIME_TYPE_DIR, parts[i]); if (next == null) throw new Exception("Could not create directory."); directories.put(current, next); }
                    directory = next;
                }
                Uri destination = DocumentsContract.createDocument(getContentResolver(), directory, "application/octet-stream", parts[parts.length - 1]);
                if (destination == null) throw new Exception("Could not create file.");
                try (OutputStream output = getContentResolver().openOutputStream(destination, "w")) { if (output == null) throw new Exception("Could not open file."); output.write(file.data); }
            }
            success = true;
        } finally { if (!success) { try { DocumentsContract.deleteDocument(getContentResolver(), root); } catch (Exception ignored) {} } }
    }

    private static byte[] boundedRead(InputStream input, int limit) throws Exception {
        if (input == null) throw new Exception("The selected file cannot be opened.");
        try (InputStream source = input; ByteArrayOutputStream output = new ByteArrayOutputStream()) {
            byte[] buffer = new byte[65536]; int count, total = 0;
            while ((count = source.read(buffer)) != -1) { total += count; if (total > limit) throw new Exception("File selection exceeds the Android evaluation limit."); output.write(buffer, 0, count); }
            Arrays.fill(buffer, (byte) 0); return output.toByteArray();
        }
    }

    private void notifyResult(boolean success, String message) {
        runOnUiThread(() -> {
            if (isFinishing() || web == null) return;
            Toast.makeText(this, message, Toast.LENGTH_LONG).show();
            if (trustedPage) web.evaluateJavascript("window.CipherEncAndroidResult && window.CipherEncAndroidResult(" + success + "," + JSONObject.quote(message) + ")", null);
        });
    }
    private static void wipe(List<ExportFile> files) { if (files != null) for (ExportFile file : files) Arrays.fill(file.data, (byte) 0); }
    private void clearPending() { if (pendingFile != null) Arrays.fill(pendingFile, (byte) 0); pendingFile = null; wipe(pendingFolder); pendingFolder = null; folderToken = null; nativeOperation = false; }
    @Override public void onBackPressed() { new AlertDialog.Builder(this).setTitle("Close CipherEnc?").setMessage("Unsaved input and results will be cleared. Saved files remain under your control.").setNegativeButton("Keep working", null).setPositiveButton("Close", (dialog, which) -> finish()).show(); }
    @Override protected void onDestroy() { trustedPage = false; clearPending(); if (chooser != null) chooser.onReceiveValue(null); chooser = null; worker.shutdownNow(); if (web != null) { web.removeJavascriptInterface("CipherEncAndroid"); web.loadUrl("about:blank"); web.clearHistory(); web.destroy(); web = null; } super.onDestroy(); }
}
