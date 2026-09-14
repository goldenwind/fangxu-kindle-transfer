package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultListenAddressUsesRandomPort(t *testing.T) {
	if defaultListenAddress != "0.0.0.0:0" {
		t.Fatalf("defaultListenAddress = %q, want random port", defaultListenAddress)
	}
}

func TestAcquireInstanceFindsRunningService(t *testing.T) {
	originalProbe := serviceProbe
	serviceProbe = func(state serviceState) bool {
		return state.Token == "running-token" && state.ControlURL == "http://localhost:49152"
	}
	t.Cleanup(func() { serviceProbe = originalProbe })

	statePath := filepath.Join(t.TempDir(), "service.json")
	state := serviceState{PID: 123, Token: "running-token", ControlURL: "http://localhost:49152"}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	claim, existingURL, err := acquireInstance(statePath, 200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if claim != nil {
		t.Fatal("running service should not create a new instance claim")
	}
	if existingURL != state.ControlURL {
		t.Fatalf("existingURL = %q, want %q", existingURL, state.ControlURL)
	}
}

func TestHealthEndpointReturnsInstanceTokenToLocalhost(t *testing.T) {
	handler, err := newApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodHead, "/health", nil)
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get(instanceHeader); got != handler.instanceToken {
		t.Fatalf("instance header = %q, want %q", got, handler.instanceToken)
	}
}

func TestAcquireInstanceReplacesStaleState(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "service.json")
	if err := os.WriteFile(statePath, []byte(`{"pid":123,"token":"stale","control_url":"http://localhost:1"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	claim, existingURL, err := acquireInstance(statePath, 0)
	if err != nil {
		t.Fatal(err)
	}
	if claim == nil || existingURL != "" {
		t.Fatalf("claim = %#v, existingURL = %q", claim, existingURL)
	}
	claim.release()
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatalf("state file remains after release: %v", err)
	}
}

func TestIndexListsOnlySupportedBooks(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "一本书.azw3"), "book")
	mustWriteFile(t, filepath.Join(root, "nested", "manual.PDF"), "pdf")
	mustWriteFile(t, filepath.Join(root, "nested", "novel.epub"), "epub")
	mustWriteFile(t, filepath.Join(root, "cover.jpg"), "image")
	mustWriteFile(t, filepath.Join(root, "notes.md"), "unsupported")

	handler, err := newApp(root)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	for _, expected := range []string{"一本书.azw3", "manual.PDF", "novel.epub", "cover.jpg", "nested"} {
		if !strings.Contains(body, expected) {
			t.Errorf("index does not contain %q", expected)
		}
	}
	if strings.Contains(body, "notes.md") {
		t.Error("index contains unsupported Markdown file")
	}
}

func TestIndexIncludesSupportAndPublicAccountQRCodes(t *testing.T) {
	handler, err := newApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	body := response.Body.String()
	for _, expected := range []string{
		"支持与关注",
		"感谢大家的支持，让这个免费项目可以持续维护与改进",
		"https://cdn.ip21.cn/img/common/alipay-qrcode.jpg",
		"https://cdn.ip21.cn/img/common/wechatpay-qrcode.jpg",
		"https://cdn.ip21.cn/img/common/wechat-pub.png",
		"关注「一灯 AI」",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("support section does not contain %q", expected)
		}
	}
	if supportPosition, shelfPosition := strings.Index(body, "支持与关注"), strings.Index(body, "我的书架"); supportPosition < 0 || shelfPosition < 0 || supportPosition > shelfPosition {
		t.Error("support section should appear before the bookshelf on the desktop page")
	}
}

func TestIndexIncludesSEOAndLanguageSwitcher(t *testing.T) {
	handler, err := newApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	body := response.Body.String()
	for _, expected := range []string{
		`name="description"`,
		`name="keywords"`,
		`property="og:title"`,
		`data-language-button="zh"`,
		`data-language-button="en"`,
		`function setLanguage(language)`,
		`window.localStorage.setItem('fangxu-language', currentLanguage)`,
		`navigator.language`,
		`Fangxu Kindle Transfer`,
		`Desktop Send to Kindle cloud transfer`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("localized page does not contain %q", expected)
		}
	}
}

func TestBooksAreSortedNewestFirst(t *testing.T) {
	root := t.TempDir()
	older := filepath.Join(root, "older.mobi")
	newer := filepath.Join(root, "newer.mobi")
	mustWriteFile(t, older, "old")
	mustWriteFile(t, newer, "new")
	oldTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newTime := oldTime.Add(24 * time.Hour)
	if err := os.Chtimes(older, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newer, newTime, newTime); err != nil {
		t.Fatal(err)
	}

	books, err := scanBooks(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 2 {
		t.Fatalf("len(books) = %d, want 2", len(books))
	}
	if books[0].Name != "newer.mobi" || books[1].Name != "older.mobi" {
		t.Fatalf("book order = [%s, %s], want newest first", books[0].Name, books[1].Name)
	}
}

func TestBookModifiedTimeIncludesHoursMinutesAndSeconds(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "timed.epub")
	mustWriteFile(t, filePath, "book")
	modified := time.Date(2026, 9, 14, 18, 27, 36, 0, time.Local)
	if err := os.Chtimes(filePath, modified, modified); err != nil {
		t.Fatal(err)
	}

	books, err := scanBooks(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 1 {
		t.Fatalf("len(books) = %d, want 1", len(books))
	}
	if books[0].Modified != "2026-09-14 18:27:36" {
		t.Fatalf("Modified = %q, want complete timestamp", books[0].Modified)
	}
}

func TestDownload(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "books", "hello world.mobi"), "ebook data")
	handler, err := newApp(root)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/download/books/hello%20world.mobi", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	result := response.Result()
	defer result.Body.Close()
	if result.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", result.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ebook data" {
		t.Fatalf("body = %q", body)
	}
	if got := result.Header.Get("Content-Disposition"); !strings.Contains(got, "attachment") {
		t.Fatalf("Content-Disposition = %q", got)
	}
}

func TestTextDownloadUsesBinaryContentType(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "notes.txt"), "plain text")
	handler, err := newApp(root)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/download/notes.txt", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	result := response.Result()
	defer result.Body.Close()
	if got := result.Header.Get("Content-Type"); got != "application/octet-stream" {
		t.Fatalf("Content-Type = %q, want application/octet-stream", got)
	}
	if got := result.Header.Get("Content-Disposition"); !strings.Contains(got, "attachment") {
		t.Fatalf("Content-Disposition = %q", got)
	}
}

func TestLocalPageOffersSendToKindleOnlyForSupportedBooks(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "manual.pdf"), "%PDF test")
	mustWriteFile(t, filepath.Join(root, "novel.epub"), "epub test")
	mustWriteFile(t, filepath.Join(root, "reader.azw3"), "azw3 test")
	handler, err := newApp(root)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	body := response.Body.String()
	if count := strings.Count(body, `href="https://www.amazon.com/sendtokindle"`); count != 2 {
		t.Fatalf("Send to Kindle link count = %d, want 2", count)
	}
	if strings.Contains(body, `/download/manual.pdf`) || strings.Contains(body, `/download/novel.epub`) {
		t.Fatal("PDF or EPUB has a direct download link")
	}
	if !strings.Contains(body, `/download/reader.azw3`) || strings.Count(body, `data-format="AZW3"`) != 1 {
		t.Fatal("AZW3 should appear only in the browser-download tab")
	}
}

func TestSendToKindleExcludesFilesLargerThan200MB(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "too-large.pdf")
	mustWriteFile(t, filePath, "pdf")
	if err := os.Truncate(filePath, sendToKindleMaxFileSize+1); err != nil {
		t.Fatal(err)
	}

	books, err := scanBooks(root)
	if err != nil {
		t.Fatal(err)
	}
	_, sendBooks := splitBooks(books)
	if len(sendBooks) != 0 {
		t.Fatalf("oversized Send to Kindle file was listed: %#v", sendBooks)
	}
}

func TestRemotePageHidesSendToKindleTab(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "manual.pdf"), "%PDF test")
	handler, err := newApp(root)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "192.168.1.20:54321"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	body := response.Body.String()
	for _, forbidden := range []string{`id="send-tab"`, `id="send-panel"`, `href="https://www.amazon.com/sendtokindle"`, "manual.pdf", `<section class="support"`, "alipay-qrcode.jpg", "wechat-pub.png"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("remote page exposes Send to Kindle content %q", forbidden)
		}
	}
}

func TestBrowserAndSendToKindleDownloadRules(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "manual.pdf"), "%PDF test")
	mustWriteFile(t, filepath.Join(root, "novel.epub"), "epub test")
	mustWriteFile(t, filepath.Join(root, "book.azw3"), "azw3 test")
	mustWriteFile(t, filepath.Join(root, "book.kfx"), "kfx test")
	handler, err := newApp(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, target := range []string{"/download/manual.pdf", "/download/novel.epub"} {
		request := httptest.NewRequest(http.MethodGet, target, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want %d", target, response.Code, http.StatusNotFound)
		}
	}
	for _, target := range []string{"/download/book.azw3", "/download/book.kfx"} {
		request := httptest.NewRequest(http.MethodGet, target, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", target, response.Code, http.StatusOK)
		}
	}
}

func TestBooksAreSortedNewestFirstAcrossFormats(t *testing.T) {
	root := t.TempDir()
	names := []string{"best.azw3", "modern.kfx", "master.epub", "legacy.mobi", "scan.pdf", "plain.txt"}
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for index, name := range names {
		path := filepath.Join(root, name)
		mustWriteFile(t, path, name)
		modified := baseTime.Add(time.Duration(index) * time.Hour)
		if err := os.Chtimes(path, modified, modified); err != nil {
			t.Fatal(err)
		}
	}

	books, err := scanBooks(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"plain.txt", "scan.pdf", "legacy.mobi", "master.epub", "modern.kfx", "best.azw3"}
	if len(books) != len(want) {
		t.Fatalf("len(books) = %d, want %d", len(books), len(want))
	}
	for index := range want {
		if books[index].Name != want[index] {
			t.Fatalf("books[%d] = %q, want %q", index, books[index].Name, want[index])
		}
	}
}

func TestEveryDisplayedFormatHasSupportedTransferMethods(t *testing.T) {
	directOnly := []string{".azw3", ".kfx", ".azw", ".mobi", ".prc"}
	sendOnly := []string{".epub", ".pdf", ".docx", ".doc", ".rtf", ".html", ".htm", ".jpg", ".jpeg", ".png", ".gif", ".bmp"}

	for _, extension := range directOnly {
		if _, ok := supportedExtensions[extension]; !ok {
			t.Errorf("direct format %s is not scanned", extension)
		}
		if _, ok := directDownloadExtensions[extension]; !ok {
			t.Errorf("direct format %s is not downloadable", extension)
		}
		if _, ok := sendToKindleExtensions[extension]; ok {
			t.Errorf("direct format %s is also marked Send to Kindle", extension)
		}
	}
	for _, extension := range sendOnly {
		if _, ok := supportedExtensions[extension]; !ok {
			t.Errorf("Send to Kindle format %s is not scanned", extension)
		}
		if _, ok := sendToKindleExtensions[extension]; !ok {
			t.Errorf("Send to Kindle format %s has no transfer method", extension)
		}
		if _, ok := directDownloadExtensions[extension]; ok {
			t.Errorf("Send to Kindle format %s is also directly downloadable", extension)
		}
	}
	if _, ok := supportedExtensions[".txt"]; !ok {
		t.Error("TXT is not scanned")
	}
	if _, ok := directDownloadExtensions[".txt"]; !ok {
		t.Error("TXT is not directly downloadable")
	}
	if _, ok := sendToKindleExtensions[".txt"]; !ok {
		t.Error("TXT is not available through Send to Kindle")
	}
}

func TestFormatFiltersAndSendToKindleExplanation(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "reader.azw3"), "book")
	mustWriteFile(t, filepath.Join(root, "classic.mobi"), "book")
	mustWriteFile(t, filepath.Join(root, "notes.txt"), "text")
	mustWriteFile(t, filepath.Join(root, "master.epub"), "epub")
	mustWriteFile(t, filepath.Join(root, "backup.epub"), "epub")
	mustWriteFile(t, filepath.Join(root, "scan.pdf"), "pdf")

	handler, err := newApp(root)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	body := response.Body.String()

	explanation := "Send to Kindle 仅支持 PDF、DOC、DOCX、TXT、RTF、HTM、HTML、PNG、GIF、JPG、JPEG、BMP 和 EPUB，单个文件最大 200 MB。"
	if !strings.Contains(body, explanation) {
		t.Fatalf("page does not contain the Send to Kindle format explanation")
	}
	directDownloadNotice := "其他格式无法直接下载，请在电脑端打开本页面，通过电脑浏览器 Send to Kindle 云端传书。"
	if !strings.Contains(body, directDownloadNotice) {
		t.Fatal("page does not explain how to transfer formats unsupported by the Kindle browser")
	}
	for _, reminder := range []string{"连接提醒：", "传书提醒：", "文件限制："} {
		if !strings.Contains(body, reminder) {
			t.Errorf("page does not emphasize reminder %q", reminder)
		}
	}
	if count := strings.Count(body, `class="important-notice"`); count != 3 {
		t.Fatalf("important reminder count = %d, want 3", count)
	}
	directFormatExplanation := "支持 AZW3、MOBI、TXT、AZW、KFX 和 PRC。"
	if !strings.Contains(body, directFormatExplanation) {
		t.Fatal("direct-download explanation does not match the format filter order")
	}
	if strings.Count(body, `data-format="TXT"`) != 2 {
		t.Fatalf("TXT should appear in both transfer tabs")
	}
	if !strings.Contains(body, "function filterBooks") || !strings.Contains(body, `data-format="EPUB"`) {
		t.Fatal("page does not contain working format filter metadata")
	}
	for _, expected := range []string{"全部（3）", "AZW3（1）", "MOBI（1）", "TXT（1）", "全部（4）", "EPUB（2）", "PDF（1）"} {
		if !strings.Contains(body, expected) {
			t.Errorf("page does not contain format filter count %q", expected)
		}
	}
	for _, absent := range []string{"filterBooks('direct', 'AZW'", "filterBooks('direct', 'KFX'", "filterBooks('send', 'DOCX'", "filterBooks('send', 'HTML'"} {
		if strings.Contains(body, absent) {
			t.Errorf("page contains zero-count filter %q", absent)
		}
	}

	assertMarkersInOrder(t, body, []string{
		"filterBooks('direct', 'AZW3'", "filterBooks('direct', 'MOBI'", "filterBooks('direct', 'TXT'",
	})
	assertMarkersInOrder(t, body, []string{
		"filterBooks('send', 'EPUB'", "filterBooks('send', 'PDF'", "filterBooks('send', 'TXT'",
	})
}

func assertMarkersInOrder(t *testing.T, text string, markers []string) {
	t.Helper()
	position := -1
	for _, marker := range markers {
		next := strings.Index(text[position+1:], marker)
		if next < 0 {
			t.Fatalf("missing ordered marker %q", marker)
		}
		position += next + 1
	}
}

func TestSendToKindleTabCopiesAbsolutePaths(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "资料", "document.docx")
	mustWriteFile(t, filePath, "document")
	handler, err := newApp(root)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	body := response.Body.String()
	for _, expected := range []string{"Kindle 内置浏览器下载（0）", "电脑浏览器 Send to Kindle 云端传书（1）", `id="send-panel"`, filePath, "copyBookPath", `class="book-details"`, "复制并打开Send to Kindle", "copyAndOpenSendToKindle", "按文件名排序", "toggleSort", `data-time="`} {
		if !strings.Contains(body, expected) {
			t.Errorf("Send to Kindle tab does not contain %q", expected)
		}
	}
	if strings.Contains(body, "点击复制路径：") {
		t.Error("Send to Kindle card still exposes the redundant full-path prompt")
	}
	if !strings.Contains(body, `window.open(destination, '_blank')`) {
		t.Error("Send to Kindle link does not open its destination during the click event")
	}
	if strings.Contains(body, `window.open('', '_blank')`) {
		t.Error("Send to Kindle link still opens an intermediate blank page")
	}
	if !strings.Contains(body, ".copy-status { position: fixed; z-index: 1000; top: 20px; left: 50%") || !strings.Contains(body, "transform: translateX(-50%)") {
		t.Error("copy status should be prominently positioned at the top center")
	}
	for _, expected := range []string{
		"选择文件时，快速使用刚复制的完整路径",
		"Windows：", "Ctrl + V", "文件名",
		"macOS：", "Command + Shift + G", "Return",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("Send to Kindle upload guide does not contain %q", expected)
		}
	}
	if count := strings.Count(body, `class="sort-toggle"`); count != 2 {
		t.Fatalf("sort button count = %d, want one in each format filter row", count)
	}
}

func TestIndexDownloadURLPathEndsWithBookExtension(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "books", "hello world.mobi"), "ebook data")
	handler, err := newApp(root)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	body := response.Body.String()
	if !strings.Contains(body, `href="/download/books/hello%20world.mobi"`) {
		t.Fatalf("index does not contain a Kindle-compatible .mobi URL: %s", body)
	}
}

func TestDownloadRejectsTraversalAndUnsupportedFiles(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "notes.md"), "secret")
	handler, err := newApp(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, target := range []string{"../outside.mobi", "notes.md"} {
		t.Run(target, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/download?path="+target, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
			}
		})
	}
}

func TestNewAppUsesConfiguredRoot(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "configured.azw3"), "book")

	handler, err := newApp(root)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "configured.azw3") {
		t.Error("index does not list a book from the configured root")
	}
}

func TestLocalControlPageShowsAddressesAndSettings(t *testing.T) {
	root := t.TempDir()
	handler, err := newApp(root)
	if err != nil {
		t.Fatal(err)
	}
	handler.setAddresses([]string{
		"http://192.168.1.8:8080",
		"http://192.168.1.9:8080",
		"http://192.168.1.10:8080",
		"http://192.168.1.11:8080",
	})
	handler.setNetworkName("方序书房 Wi-Fi")

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	body := response.Body.String()
	for _, expected := range []string{"方序传书", "方寸之间，自有书序", "电脑端传书设置", "服务运行中", "当前网络", "方序书房 Wi-Fi", "address-list", "directory-row", "http://192.168.1.8:8080", "http://192.168.1.9:8080", "http://192.168.1.10:8080", "选择文件夹", "<svg", "#3446b7"} {
		if !strings.Contains(body, expected) {
			t.Errorf("control page does not contain %q", expected)
		}
	}
	if strings.Contains(body, "http://192.168.1.11:8080") {
		t.Error("control page displays more than three IP addresses")
	}
	if count := strings.Count(body, `class="address"`); count != 3 {
		t.Fatalf("address count = %d, want 3", count)
	}
}

func TestMacOSNetworkHardwareParsing(t *testing.T) {
	output := "Hardware Port: Ethernet\nDevice: en7\n\nHardware Port: Wi-Fi\nDevice: en0\n"
	wifiDevice, activePort := macOSWiFiDeviceAndActivePort(output, "en7")
	if wifiDevice != "en0" || activePort != "Ethernet" {
		t.Fatalf("parsed Wi-Fi device and active port = %q, %q", wifiDevice, activePort)
	}
}

func TestUnavailableNetworkNameIsHidden(t *testing.T) {
	for _, unavailable := range []string{"", "<redacted>", "redacted", "SSID: <redacted>", "<red\u200bacted>", "not associated"} {
		if isUsableNetworkName(unavailable) {
			t.Errorf("network placeholder %q should not be displayed", unavailable)
		}
	}

	handler, err := newApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Simulate an invalid value already stored by an older detection path. Rendering
	// must still suppress it rather than relying only on setNetworkName validation.
	handler.networkName = "SSID: <redacted>"
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	body := response.Body.String()
	if strings.Contains(body, `class="network-info"`) || strings.Contains(body, "&lt;redacted&gt;") {
		t.Fatal("unavailable network name should be hidden")
	}
	if cacheControl := response.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cacheControl)
	}
}

func TestLocalControlCanChangeDirectory(t *testing.T) {
	initialRoot := t.TempDir()
	newRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(newRoot, "new-book.mobi"), "book")
	handler, err := newApp(initialRoot)
	if err != nil {
		t.Fatal(err)
	}

	form := url.Values{
		"token":  {handler.csrfToken},
		"path":   {newRoot},
		"action": {"apply"},
	}
	request := httptest.NewRequest(http.MethodPost, "/settings/directory", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusSeeOther)
	}
	root, _ := handler.roots()
	if root != newRoot {
		t.Fatalf("root = %q, want %q", root, newRoot)
	}
}

func TestRemoteClientCannotChangeDirectory(t *testing.T) {
	root := t.TempDir()
	handler, err := newApp(root)
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"token": {handler.csrfToken}, "path": {t.TempDir()}, "action": {"apply"}}
	request := httptest.NewRequest(http.MethodPost, "/settings/directory", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.RemoteAddr = "192.168.1.20:54321"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func mustWriteFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
