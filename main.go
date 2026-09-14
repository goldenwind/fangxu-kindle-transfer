package main

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

var supportedExtensions = map[string]struct{}{
	".azw": {}, ".azw3": {}, ".epub": {}, ".kfx": {}, ".mobi": {},
	".pdf": {}, ".prc": {}, ".txt": {}, ".docx": {}, ".doc": {},
	".rtf": {}, ".html": {}, ".htm": {}, ".jpg": {}, ".jpeg": {},
	".png": {}, ".gif": {}, ".bmp": {},
}

var sendToKindleExtensions = map[string]struct{}{
	".epub": {}, ".pdf": {}, ".docx": {}, ".doc": {}, ".rtf": {},
	".html": {}, ".htm": {}, ".jpg": {}, ".jpeg": {}, ".png": {},
	".gif": {}, ".bmp": {}, ".txt": {},
}

var directDownloadExtensions = map[string]struct{}{
	".azw3": {}, ".kfx": {}, ".azw": {}, ".mobi": {}, ".prc": {}, ".txt": {},
}

var directFormatOrder = []string{"AZW3", "MOBI", "TXT", "AZW", "KFX", "PRC"}

var sendToKindleFormatOrder = []string{
	"EPUB", "PDF", "TXT", "DOCX", "JPG", "PNG", "JPEG", "DOC", "RTF",
	"HTML", "HTM", "GIF", "BMP",
}

const sendToKindleURL = "https://www.amazon.com/sendtokindle"
const sendToKindleMaxFileSize int64 = 200 * 1024 * 1024

const (
	defaultListenAddress = "0.0.0.0:0"
	instanceHeader       = "X-Kindle-Transfer-Instance"
)

var serviceProbe = probeService

type book struct {
	Name         string
	Directory    string
	FilePath     string
	DownloadURL  string
	SendToKindle bool
	Size         string
	Modified     string
	ModifiedUnix int64
	Format       string
	modifiedAt   time.Time
}

type formatFilter struct {
	Name  string
	Count int
}

type pageData struct {
	DirectBooks     []book
	SendBooks       []book
	DirectFilters   []formatFilter
	SendFilters     []formatFilter
	BookCount       int
	Root            string
	Admin           bool
	Addresses       []string
	NetworkName     string
	CSRFToken       string
	Status          string
	SendToKindleURL string
}

type app struct {
	mu            sync.RWMutex
	root          string
	resolvedRoot  string
	addresses     []string
	networkName   string
	csrfToken     string
	instanceToken string
	pickerMu      sync.Mutex
	shutdown      func()
	template      *template.Template
}

type serviceState struct {
	PID        int    `json:"pid"`
	Token      string `json:"token"`
	ControlURL string `json:"control_url,omitempty"`
}

type instanceClaim struct {
	path  string
	token string
}

func randomToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(tokenBytes), nil
}

func serviceStatePath() (string, error) {
	cacheDirectory, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	directory := filepath.Join(cacheDirectory, "kindle-transfer")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(directory, "service.json"), nil
}

func acquireInstance(path string, wait time.Duration) (*instanceClaim, string, error) {
	token, err := randomToken()
	if err != nil {
		return nil, "", err
	}
	state := serviceState{PID: os.Getpid(), Token: token}
	stateData, err := json.Marshal(state)
	if err != nil {
		return nil, "", err
	}
	deadline := time.Now().Add(wait)

	for {
		file, createErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if createErr == nil {
			if _, err := file.Write(stateData); err != nil {
				file.Close()
				_ = os.Remove(path)
				return nil, "", err
			}
			if err := file.Close(); err != nil {
				_ = os.Remove(path)
				return nil, "", err
			}
			return &instanceClaim{path: path, token: token}, "", nil
		}
		if !errors.Is(createErr, os.ErrExist) {
			return nil, "", createErr
		}

		existing, raw, readErr := readServiceState(path)
		if readErr == nil && serviceProbe(existing) {
			return nil, existing.ControlURL, nil
		}
		if time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		removed, err := removeStateIfUnchanged(path, raw)
		if err != nil {
			return nil, "", err
		}
		if !removed {
			deadline = time.Now().Add(wait)
		}
	}
}

func readServiceState(path string) (serviceState, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return serviceState{}, nil, err
	}
	var state serviceState
	if err := json.Unmarshal(raw, &state); err != nil {
		return serviceState{}, raw, err
	}
	if state.Token == "" {
		return serviceState{}, raw, errors.New("服务状态缺少实例标识")
	}
	return state, raw, nil
}

func removeStateIfUnchanged(path string, observed []byte) (bool, error) {
	current, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if !bytes.Equal(current, observed) {
		return false, nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	return true, nil
}

func probeService(state serviceState) bool {
	parsed, err := url.Parse(state.ControlURL)
	if err != nil || parsed.Scheme != "http" || parsed.Port() == "" || !isLoopbackHost(parsed.Hostname()) {
		return false
	}
	request, err := http.NewRequest(http.MethodHead, strings.TrimRight(state.ControlURL, "/")+"/health", nil)
	if err != nil {
		return false
	}
	client := &http.Client{Timeout: 250 * time.Millisecond}
	response, err := client.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusOK && subtle.ConstantTimeCompare([]byte(response.Header.Get(instanceHeader)), []byte(state.Token)) == 1
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (claim *instanceClaim) publish(controlURL string) error {
	state, _, err := readServiceState(claim.path)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(state.Token), []byte(claim.token)) != 1 {
		return errors.New("服务状态已被另一进程接管")
	}
	state.ControlURL = controlURL
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(claim.path, data, 0o600)
}

func (claim *instanceClaim) release() {
	state, raw, err := readServiceState(claim.path)
	if err != nil || subtle.ConstantTimeCompare([]byte(state.Token), []byte(claim.token)) != 1 {
		return
	}
	_, _ = removeStateIfUnchanged(claim.path, raw)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	listenAddress := flag.String("listen", defaultListenAddress, "监听地址，端口为 0 时自动选择空闲端口，例如 0.0.0.0:0")
	bookDirectory := flag.String("dir", "", "电子书目录，默认为用户的下载目录")
	noOpen := flag.Bool("no-open", false, "启动后不自动打开本机设置页面")
	flag.Parse()

	statePath, err := serviceStatePath()
	if err != nil {
		return fmt.Errorf("创建服务状态目录失败: %w", err)
	}
	claim, existingURL, err := acquireInstance(statePath, 3*time.Second)
	if err != nil {
		return fmt.Errorf("检查已运行服务失败: %w", err)
	}
	if existingURL != "" {
		log.Printf("方序传书服务已在运行: %s", existingURL)
		if !*noOpen {
			if err := openBrowser(existingURL); err != nil {
				return fmt.Errorf("无法打开已运行服务，请手动访问 %s: %w", existingURL, err)
			}
		}
		return nil
	}
	defer claim.release()

	root := *bookDirectory
	if root == "" {
		root = defaultBookDirectory()
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("读取电子书目录失败: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("读取电子书目录失败: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("电子书目录不是文件夹: %s", root)
	}

	handler, err := newApp(root)
	if err != nil {
		return fmt.Errorf("启动失败: %w", err)
	}
	handler.instanceToken = claim.token
	handler.setNetworkName(currentNetworkName())

	listener, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		return fmt.Errorf("监听 %s 失败: %w", *listenAddress, err)
	}
	defer listener.Close()

	controlURL := "http://localhost" + displayPort(listener.Addr())
	if err := claim.publish(controlURL); err != nil {
		return fmt.Errorf("保存服务地址失败: %w", err)
	}
	kindleURLs := localNetworkURLs(listener.Addr())
	handler.setAddresses(kindleURLs)
	log.Printf("电子书根目录: %s", root)
	log.Printf("本机设置页面: %s", controlURL)
	for _, address := range kindleURLs {
		log.Printf("Kindle 访问: %s", address)
	}
	if networkName := handler.pageNetworkName(); networkName != "" {
		log.Printf("当前网络: %s", networkName)
	}
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	handler.setShutdown(func() {
		if err := server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("停止服务器失败: %v", err)
		}
	})
	if !*noOpen {
		go func() {
			time.Sleep(300 * time.Millisecond)
			if err := openBrowser(controlURL); err != nil {
				log.Printf("无法自动打开浏览器，请手动访问 %s: %v", controlURL, err)
			}
		}()
	}
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("服务器异常退出: %w", err)
	}
	return nil
}

func newApp(root string) (*app, error) {
	csrfToken, err := randomToken()
	if err != nil {
		return nil, err
	}
	instanceToken, err := randomToken()
	if err != nil {
		return nil, err
	}
	a := &app{
		csrfToken:     csrfToken,
		instanceToken: instanceToken,
		template:      template.Must(template.New("index").Parse(indexTemplate)),
	}
	if err := a.setRoot(root); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *app) setRoot(root string) error {
	absoluteRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return err
	}
	info, err := os.Stat(absoluteRoot)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("不是文件夹: %s", absoluteRoot)
	}
	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.root = absoluteRoot
	a.resolvedRoot = resolvedRoot
	a.mu.Unlock()
	log.Printf("共享目录已设置为: %s", absoluteRoot)
	return nil
}

func (a *app) roots() (string, string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.root, a.resolvedRoot
}

func (a *app) setAddresses(addresses []string) {
	if len(addresses) > 3 {
		addresses = addresses[:3]
	}
	a.mu.Lock()
	a.addresses = append([]string(nil), addresses...)
	a.mu.Unlock()
}

func (a *app) setNetworkName(name string) {
	if !isUsableNetworkName(name) {
		name = ""
	}
	a.mu.Lock()
	a.networkName = strings.TrimSpace(name)
	a.mu.Unlock()
}

func (a *app) setShutdown(shutdown func()) {
	a.mu.Lock()
	a.shutdown = shutdown
	a.mu.Unlock()
}

func (a *app) pageAddresses() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]string(nil), a.addresses...)
}

func (a *app) pageNetworkName() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if !isUsableNetworkName(a.networkName) {
		return ""
	}
	return a.networkName
}

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/health":
		a.serveHealth(w, r)
	case r.URL.Path == "/":
		a.serveIndex(w, r)
	case r.URL.Path == "/download" || strings.HasPrefix(r.URL.Path, "/download/"):
		a.serveDownload(w, r)
	case r.URL.Path == "/settings/directory":
		a.serveDirectorySettings(w, r)
	case r.URL.Path == "/settings/stop":
		a.serveStop(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (a *app) serveHealth(w http.ResponseWriter, r *http.Request) {
	if (r.Method != http.MethodGet && r.Method != http.MethodHead) || !isLoopbackRequest(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set(instanceHeader, a.instanceToken)
	w.WriteHeader(http.StatusOK)
}

func (a *app) serveStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !isLoopbackRequest(r) || subtle.ConstantTimeCompare([]byte(r.FormValue("token")), []byte(a.csrfToken)) != 1 {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = fmt.Fprint(w, `<!doctype html><html lang="zh-CN"><meta charset="utf-8"><title>服务已停止</title><body style="margin:0;padding:32px;color:#141933;background:#edf2ff;font-family:-apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,&quot;PingFang SC&quot;,&quot;Microsoft YaHei&quot;,sans-serif"><h1>方序传书服务已停止</h1><p>现在可以关闭此页面。</p></body></html>`)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	a.mu.RLock()
	shutdown := a.shutdown
	a.mu.RUnlock()
	if shutdown != nil {
		go func() {
			time.Sleep(100 * time.Millisecond)
			shutdown()
		}()
	}
}

func (a *app) serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	root, _ := a.roots()
	books, err := scanBooks(root)
	if err != nil {
		http.Error(w, "读取电子书目录失败", http.StatusInternalServerError)
		log.Printf("扫描目录失败: %v", err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodHead {
		return
	}
	directBooks, sendBooks := splitBooks(books)
	data := pageData{
		DirectBooks:     directBooks,
		SendBooks:       sendBooks,
		DirectFilters:   buildFormatFilters(directBooks, directFormatOrder),
		SendFilters:     buildFormatFilters(sendBooks, sendToKindleFormatOrder),
		BookCount:       len(books),
		Root:            root,
		Admin:           isLoopbackRequest(r),
		Addresses:       a.pageAddresses(),
		NetworkName:     a.pageNetworkName(),
		CSRFToken:       a.csrfToken,
		Status:          statusMessage(r.URL.Query().Get("status")),
		SendToKindleURL: sendToKindleURL,
	}
	if err := a.template.Execute(w, data); err != nil {
		log.Printf("生成页面失败: %v", err)
	}
}

func (a *app) serveDirectorySettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isLoopbackRequest(r) || subtle.ConstantTimeCompare([]byte(r.FormValue("token")), []byte(a.csrfToken)) != 1 {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	directory := r.FormValue("path")
	if r.FormValue("action") == "choose" {
		a.pickerMu.Lock()
		chosen, err := chooseDirectory()
		a.pickerMu.Unlock()
		if err != nil {
			log.Printf("选择目录失败: %v", err)
			http.Redirect(w, r, "/?status=cancelled", http.StatusSeeOther)
			return
		}
		directory = chosen
	}

	if err := a.setRoot(directory); err != nil {
		log.Printf("设置目录失败: %v", err)
		http.Redirect(w, r, "/?status=invalid", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/?status=changed", http.StatusSeeOther)
}

func scanBooks(root string) ([]book, error) {
	var books []book
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		if _, ok := supportedExtensions[strings.ToLower(filepath.Ext(entry.Name()))]; !ok {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		directory := filepath.Dir(relativePath)
		if directory == "." {
			directory = ""
		}
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		_, sendToKindleFormat := sendToKindleExtensions[extension]
		useSendToKindle := sendToKindleFormat && info.Size() <= sendToKindleMaxFileSize
		_, directDownload := directDownloadExtensions[extension]
		downloadURL := ""
		if directDownload {
			downloadURL = kindleDownloadURL(relativePath)
		}
		books = append(books, book{
			Name:         entry.Name(),
			Directory:    filepath.ToSlash(directory),
			FilePath:     path,
			DownloadURL:  downloadURL,
			SendToKindle: useSendToKindle,
			Size:         humanSize(info.Size()),
			Modified:     info.ModTime().Format("2006-01-02 15:04:05"),
			ModifiedUnix: info.ModTime().Unix(),
			Format:       strings.ToUpper(strings.TrimPrefix(extension, ".")),
			modifiedAt:   info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(books, func(i, j int) bool {
		if !books[i].modifiedAt.Equal(books[j].modifiedAt) {
			return books[i].modifiedAt.After(books[j].modifiedAt)
		}
		left := strings.ToLower(filepath.Join(books[i].Directory, books[i].Name))
		right := strings.ToLower(filepath.Join(books[j].Directory, books[j].Name))
		return left < right
	})
	return books, nil
}

func splitBooks(books []book) (directBooks, sendBooks []book) {
	for _, item := range books {
		if item.DownloadURL != "" {
			directBooks = append(directBooks, item)
		}
		if item.SendToKindle {
			sendBooks = append(sendBooks, item)
		}
	}
	return directBooks, sendBooks
}

func buildFormatFilters(books []book, preferredOrder []string) []formatFilter {
	counts := make(map[string]int)
	for _, item := range books {
		counts[item.Format]++
	}
	filters := make([]formatFilter, 0, len(counts))
	for _, format := range preferredOrder {
		if count := counts[format]; count > 0 {
			filters = append(filters, formatFilter{Name: format, Count: count})
		}
	}
	return filters
}

func kindleDownloadURL(relativePath string) string {
	parts := strings.Split(filepath.ToSlash(relativePath), "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return "/download/" + strings.Join(parts, "/")
}

func defaultBookDirectory() string {
	home, err := os.UserHomeDir()
	if err == nil {
		downloads := filepath.Join(home, "Downloads")
		if info, statErr := os.Stat(downloads); statErr == nil && info.IsDir() {
			return downloads
		}
	}
	current, err := os.Getwd()
	if err == nil {
		return current
	}
	return "."
}

func (a *app) serveDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	requested := r.URL.Query().Get("path")
	if strings.HasPrefix(r.URL.Path, "/download/") {
		requested = strings.TrimPrefix(r.URL.Path, "/download/")
	}
	path, err := a.safeBookPath(requested)
	if err != nil {
		http.Error(w, "电子书不存在", http.StatusNotFound)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}

	name := filepath.Base(path)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if contentType := ebookContentType(filepath.Ext(name)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	http.ServeContent(w, r, name, info.ModTime(), file)
}

func (a *app) safeBookPath(requested string) (string, error) {
	if requested == "" || strings.ContainsRune(requested, '\x00') {
		return "", fs.ErrNotExist
	}
	requested = filepath.FromSlash(requested)
	if filepath.IsAbs(requested) {
		return "", fs.ErrPermission
	}
	cleaned := filepath.Clean(requested)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fs.ErrPermission
	}
	if _, ok := supportedExtensions[strings.ToLower(filepath.Ext(cleaned))]; !ok {
		return "", fs.ErrPermission
	}
	if _, directDownload := directDownloadExtensions[strings.ToLower(filepath.Ext(cleaned))]; !directDownload {
		return "", fs.ErrPermission
	}

	root, resolvedRoot := a.roots()
	candidate, err := filepath.EvalSymlinks(filepath.Join(root, cleaned))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(resolvedRoot, candidate)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fs.ErrPermission
	}
	return candidate, nil
}

func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func statusMessage(status string) string {
	switch status {
	case "changed":
		return "共享目录已更新。"
	case "invalid":
		return "目录无效或无法访问，请检查路径。"
	case "cancelled":
		return "未选择新目录。"
	default:
		return ""
	}
}

func chooseDirectory() (string, error) {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("osascript", "-e", `POSIX path of (choose folder with prompt "选择电子书目录")`)
	case "windows":
		script := `Add-Type -AssemblyName System.Windows.Forms; $dialog = New-Object System.Windows.Forms.FolderBrowserDialog; $dialog.Description = '选择电子书目录'; $dialog.ShowNewFolderButton = $true; if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { [Console]::OutputEncoding = [System.Text.Encoding]::UTF8; Write-Output $dialog.SelectedPath }`
		command = exec.Command("powershell.exe", "-NoProfile", "-STA", "-Command", script)
	case "linux":
		if _, err := exec.LookPath("zenity"); err == nil {
			command = exec.Command("zenity", "--file-selection", "--directory", "--title=选择电子书目录")
		} else if _, err := exec.LookPath("kdialog"); err == nil {
			command = exec.Command("kdialog", "--getexistingdirectory", ".", "--title", "选择电子书目录")
		} else {
			return "", errors.New("未找到 zenity 或 kdialog，请安装其中一个或手动输入目录路径")
		}
	default:
		return "", errors.New("当前系统不支持可视化目录选择")
	}
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	directory := strings.TrimSpace(string(output))
	if directory == "" {
		return "", errors.New("未选择目录")
	}
	return directory, nil
}

func openBrowser(address string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", address)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", address)
	default:
		command = exec.Command("xdg-open", address)
	}
	return command.Start()
}

func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	divisor, exponent := int64(unit), 0
	for value := bytes / unit; value >= unit; value /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(divisor), "KMGTPE"[exponent])
}

func ebookContentType(extension string) string {
	switch strings.ToLower(extension) {
	case ".azw", ".azw3", ".kfx":
		return "application/vnd.amazon.ebook"
	case ".mobi", ".prc":
		return "application/x-mobipocket-ebook"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		// Older Kindle browsers ignore Content-Disposition for text/plain and
		// render the file in the browser. A generic binary type triggers saving.
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
}

func displayPort(address net.Addr) string {
	_, port, err := net.SplitHostPort(address.String())
	if err != nil {
		return ":0"
	}
	return ":" + port
}

func currentNetworkName() string {
	var name string
	switch runtime.GOOS {
	case "darwin":
		name = macOSNetworkName()
	case "windows":
		name = windowsNetworkName()
	default:
		name = linuxNetworkName()
	}
	if isUsableNetworkName(name) {
		return strings.TrimSpace(name)
	}
	return ""
}

func macOSNetworkName() string {
	hardwareOutput, _ := exec.Command("/usr/sbin/networksetup", "-listallhardwareports").CombinedOutput()
	device, _ := macOSWiFiDeviceAndActivePort(string(hardwareOutput), "")
	if device != "" {
		if output, err := exec.Command("/usr/sbin/networksetup", "-getairportnetwork", device).CombinedOutput(); err == nil {
			text := strings.TrimSpace(string(output))
			if value := valueAfterColon(text); isUsableNetworkName(value) {
				return value
			}
		}
		if name := macOSSSIDForDevice(device); name != "" {
			return name
		}
	}
	if interfaces, err := net.Interfaces(); err == nil {
		for _, networkInterface := range interfaces {
			if networkInterface.Flags&net.FlagUp != 0 && networkInterface.Flags&net.FlagLoopback == 0 {
				if name := macOSSSIDForDevice(networkInterface.Name); name != "" {
					return name
				}
			}
		}
	}
	return ""
}

func macOSSSIDForDevice(device string) string {
	output, err := exec.Command("/usr/sbin/ipconfig", "getsummary", device).CombinedOutput()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), "SSID") {
			name := strings.TrimSpace(parts[1])
			if isUsableNetworkName(name) {
				return name
			}
		}
	}
	return ""
}

func macOSWiFiDeviceAndActivePort(output, activeDevice string) (string, string) {
	currentPort := ""
	wifiDevice := ""
	activePort := ""
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		switch key {
		case "Hardware Port":
			currentPort = value
		case "Device":
			if currentPort == "Wi-Fi" || currentPort == "AirPort" {
				wifiDevice = value
			}
			if value == activeDevice {
				activePort = currentPort
			}
		}
	}
	return wifiDevice, activePort
}

func windowsNetworkName() string {
	output, err := exec.Command("netsh", "wlan", "show", "interfaces").CombinedOutput()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), "SSID") {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}

func linuxNetworkName() string {
	output, err := exec.Command("nmcli", "-t", "-f", "NAME", "connection", "show", "--active").CombinedOutput()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(output), "\n") {
		if name := strings.TrimSpace(strings.ReplaceAll(line, `\:`, ":")); name != "" && name != "lo" {
			return name
		}
	}
	return ""
}

func isUsableNetworkName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(strings.Map(func(character rune) rune {
		if unicode.Is(unicode.Cf, character) {
			return -1
		}
		return character
	}, name)))
	switch normalized {
	case "", "(null)", "null", "unknown", "n/a":
		return false
	}
	return !strings.Contains(normalized, "redacted") && !strings.Contains(normalized, "not associated")
}

func valueAfterColon(text string) string {
	parts := strings.SplitN(text, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func localNetworkURLs(address net.Addr) []string {
	_, port, err := net.SplitHostPort(address.String())
	if err != nil {
		return nil
	}
	interfaces, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	var urls []string
	for _, item := range interfaces {
		ip, _, err := net.ParseCIDR(item.String())
		if err != nil || ip.IsLoopback() || ip.To4() == nil {
			continue
		}
		urls = append(urls, "http://"+net.JoinHostPort(ip.String(), port))
	}
	sort.Strings(urls)
	return urls
}

const indexTemplate = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="description" content="方序传书是一款免费开源的 Kindle 局域网电子书传输工具，支持 AZW3、MOBI、KFX、EPUB、PDF 和 Send to Kindle，适用于 macOS、Windows 与 Linux。">
  <meta name="keywords" content="Kindle传书,Kindle transfer,Send to Kindle,电子书传输,Kindle浏览器,AZW3,MOBI,EPUB,macOS,Windows,Linux">
  <meta name="robots" content="index,follow">
  <meta property="og:type" content="website">
  <meta property="og:title" content="方序传书 · Kindle Transfer">
  <meta property="og:description" content="Free, open-source Kindle ebook transfer over local Wi-Fi, with Send to Kindle support for macOS, Windows, and Linux.">
  <title>方序传书</title>
  <style>
    * { box-sizing: border-box; }
    body { margin: 0; padding: 20px; color: #172039; background: #f3f4f7; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", Arial, sans-serif; }
    .page { width: 100%; max-width: none; min-height: calc(100vh - 36px); margin: 0; }
    h1 { margin: 6px 0; color: #172039; font-size: 30px; letter-spacing: -.02em; }
    h2 { margin: 0 0 8px; color: #172039; font-size: 22px; }
    .brand-header { display: flex; align-items: center; margin: 0 0 22px; padding-bottom: 16px; border-bottom: 1px solid #dfe2e9; }
    .brand-mark { display: block; width: 52px; height: 52px; margin-right: 13px; color: #3446b7; }
    .brand-mark svg { display: block; width: 100%; height: 100%; }
    .brand-copy { display: block; }
    .brand-name { display: block; color: #172039; font-size: 27px; font-weight: 760; letter-spacing: -.03em; }
    .brand-tagline { display: block; margin-top: 3px; color: #70778a; font-size: 13px; font-weight: 600; letter-spacing: .12em; }
    .language-switch { display: flex; flex: none; margin-left: auto; padding: 3px; border: 1px solid #d6dae4; border-radius: 5px; background: #fff; }
    .language-button { margin: 0; padding: 6px 10px; border: 0; border-radius: 3px; background: transparent; color: #70778a; font-size: 13px; font-weight: 700; }
    .language-button.active { background: #3446b7; color: #fff; }
    .sort-toggle { flex: none; margin: 0 0 8px auto; padding: 8px 13px; border-color: #cbd0dc; border-radius: 5px; background: #fff; color: #3446b7; font-size: 14px; font-weight: bold; white-space: nowrap; }
    .summary { margin: 0 0 18px; color: #70778a; }
    .admin { margin: 0 0 26px; padding: 20px; border: 1px solid #dfe2e9; border-radius: 6px; background: #fff; box-shadow: 0 4px 14px rgba(23,32,57,.04); }
    .admin-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 18px; }
    .admin-heading { flex: 1; min-width: 0; }
    .admin-intro { margin: 0 0 12px; color: #70778a; line-height: 1.6; }
    .service-actions { display: flex; flex: none; align-items: center; gap: 10px; }
    .service-actions form { margin: 0; }
    .service-status { display: inline-flex; align-items: center; color: #2c7250; font-size: 14px; font-weight: 700; white-space: nowrap; }
    .service-status-dot { width: 7px; height: 7px; margin-right: 7px; border-radius: 50%; background: #319364; }
    .network-info { display: flex; align-items: center; gap: 9px; margin: 2px 0 12px; padding: 9px 0; border-bottom: 1px solid #e7e9ee; }
    .network-label { color: #70778a; font-size: 14px; }
    .network-name { color: #245f43; font-size: 15px; word-break: break-word; }
    .address-list { display: flex; flex-wrap: nowrap; gap: 8px; width: 100%; margin: 8px 0; overflow-x: auto; }
    .address { display: block; flex: 1 0 210px; min-width: 0; margin: 0; padding: 11px 13px; border: 1px solid #dfe2e9; border-radius: 4px; background: #f7f8fb; color: #29345f; font-size: 17px; white-space: nowrap; }
    .status { margin: 12px 0; padding: 11px 13px; border-left: 3px solid #b98525; background: #f8f5ec; color: #69551f; }
	.notice { margin: 8px 0 0; color: #765815; line-height: 1.55; }
    label { display: block; margin: 14px 0 6px; font-weight: bold; }
    input[type=text] { box-sizing: border-box; width: 100%; padding: 12px 13px; border: 1px solid #cbd0dc; border-radius: 4px; background: #fff; color: #172039; font-size: 16px; }
    input[type=text]:focus { border-color: #3446b7; outline: 2px solid rgba(52,70,183,.1); }
    .directory-row { display: flex; align-items: center; gap: 10px; }
    .directory-row input[type=text] { flex: 1; min-width: 260px; max-width: 560px; }
    .directory-actions { display: flex; flex: none; align-items: center; gap: 8px; }
    .directory-actions button, .service-actions button { margin: 0; white-space: nowrap; }
    button { margin: 12px 8px 0 0; padding: 11px 16px; border: 1px solid #cbd0dc; border-radius: 5px; background: #fff; color: #172039; font-size: 16px; cursor: pointer; }
    button:hover, button:focus { border-color: #3446b7; outline: none; }
    .primary { border-color: #3446b7; background: #3446b7; color: #fff; }
    .danger { border-color: #bb5656; color: #a83a3a; }
    .tabs { display: flex; margin: 0 0 16px; border-bottom: 1px solid #d7dae2; }
    .tab { flex: 1; margin: 0 0 -1px; padding: 12px 8px; border: 0; border-bottom: 3px solid transparent; border-radius: 0; background: transparent; color: #70778a; font-weight: bold; }
    .tab.active { border-bottom-color: #3446b7; background: transparent; color: #26358f; }
    .tab-panel[hidden] { display: none; }
    .tab-note { margin: 0 0 14px; padding: 14px 16px; border-left: 3px solid #3446b7; background: #f8f9fc; color: #353d55; line-height: 1.65; }
    .important-notice { position: relative; display: block; margin: 9px 0 0; padding: 8px 0 0 21px; border-top: 1px solid #ddd6c4; background: transparent; color: #6c5723; font-weight: 600; line-height: 1.55; }
    .important-notice:before { position: absolute; left: 1px; top: 9px; content: "!"; color: #9a6d16; font-weight: 800; }
    .admin-heading .important-notice { margin: 0 0 12px; padding-top: 0; border-top: 0; }
    .admin-heading .important-notice:before { top: 1px; }
    .upload-guide { margin: 0 0 14px; padding: 13px 15px; border-left: 3px solid #4b8065; background: #f6f9f7; color: #2c4f3c; line-height: 1.65; }
    .upload-guide-title { display: block; margin-bottom: 4px; color: #235e40; }
    .upload-guide-platform { display: block; }
    .filter-toolbar { display: flex; align-items: flex-start; gap: 12px; }
    .format-filters { display: flex; flex: 1; flex-wrap: wrap; min-width: 0; margin: 0 0 8px; }
    .format-filter { margin: 0 8px 8px 0; padding: 8px 13px; border: 1px solid #cbd0dc; border-radius: 5px; background: #fff; color: #3446b7; font-size: 14px; font-weight: bold; }
    .format-filter.active { border-color: #3446b7; background: #3446b7; color: #fff; }
    .book { display: block; box-sizing: border-box; width: 100%; margin: 0 0 9px; padding: 15px 17px; border: 1px solid #dfe2e9; border-radius: 5px; background: #fff; color: #172039; text-align: left; text-decoration: none; }
	.book:hover, .book:focus { border-color: #9ea7ca; background: #fafbfe; box-shadow: 0 3px 10px rgba(23,32,57,.05); outline: none; }
	.book-copy { display: flex; align-items: center; cursor: copy; }
	.book-copy:hover, .book-copy:focus { border-color: #7e89bd; background: #fafbfe; }
	.book-details { flex: 1; min-width: 0; }
	.send-link { display: block; flex: none; margin-left: 18px; padding: 10px 14px; border: 1px solid #3446b7; border-radius: 5px; background: #3446b7; color: #fff; font-size: 16px; font-weight: bold; text-align: center; text-decoration: none; white-space: nowrap; }
    .title { display: block; font-size: 21px; font-weight: bold; line-height: 1.35; word-wrap: break-word; }
    .meta { display: block; margin-top: 8px; color: #70778a; font-size: 14px; }
    .copy-status { position: fixed; z-index: 1000; top: 20px; left: 50%; width: max-content; max-width: min(720px, calc(100vw - 32px)); padding: 13px 18px; border: 1px solid rgba(255,255,255,.22); border-radius: 5px; background: #172039; color: #fff; box-shadow: 0 12px 32px rgba(23,32,57,.28); line-height: 1.55; overflow-wrap: anywhere; transform: translateX(-50%); }
    .empty { padding: 24px 16px; border: 1px dashed #c5cad5; border-radius: 5px; background: #f8f9fb; color: #70778a; line-height: 1.7; }
    .support { margin: 0 0 26px; padding: 20px; border: 1px solid #dfe2e9; border-radius: 5px; background: #fff; }
    .support-heading { margin-bottom: 16px; }
    .support-heading h2 { margin-bottom: 6px; }
    .support-heading p { width: 100%; margin: 0; color: #626a7d; line-height: 1.7; }
    .support-grid { display: flex; flex-wrap: wrap; gap: 12px; }
    .support-card { display: flex; flex: 1 1 240px; align-items: center; min-width: 0; padding: 14px; border: 1px solid #dfe2e9; border-radius: 5px; background: #fff; }
    .support-card img { display: block; flex: none; width: 116px; height: 116px; padding: 4px; border: 1px solid #e5e7ec; background: #fff; object-fit: contain; }
    .support-card-copy { min-width: 0; margin-left: 14px; }
    .support-card-title { display: block; color: #172039; font-size: 17px; line-height: 1.4; }
    .support-card-note { display: block; margin-top: 6px; color: #70778a; font-size: 13px; line-height: 1.55; }
    .support-card-public { border-color: #cbd2eb; background: #fafbff; }
    .footer { margin-top: 24px; padding: 14px 4px 4px; border-top: 1px solid #dfe2e9; color: #70778a; font-size: 12px; line-height: 1.7; word-wrap: break-word; }
    .footer-root { display: block; }
    [hidden] { display: none !important; }
    @media (max-width: 600px) {
      body { padding: 7px; }
      .page { min-height: calc(100vh - 14px); }
      .brand-header { margin: 4px 4px 14px; padding-bottom: 12px; }
      .brand-mark { width: 45px; height: 45px; margin-right: 10px; }
      .brand-name { font-size: 23px; }
      .brand-tagline { font-size: 12px; }
      .language-button { padding: 6px 8px; }
      h1 { font-size: 25px; }
      .admin { padding: 15px; border-radius: 4px; }
      .admin-header { display: block; }
      .service-actions { justify-content: flex-end; margin: -2px 0 10px; }
      .network-info { align-items: flex-start; }
      .network-label { flex: none; }
      .directory-row { flex-wrap: wrap; }
      .directory-row input[type=text] { flex-basis: 100%; min-width: 0; max-width: none; }
      .directory-actions { width: 100%; }
      .directory-actions button { flex: 1; min-width: 0; padding: 10px 7px; font-size: 14px; }
      .filter-toolbar { flex-wrap: wrap; gap: 0; }
      .format-filters { flex-basis: 100%; margin-bottom: 0; }
      .sort-toggle { margin-left: auto; padding: 8px 11px; font-size: 13px; }
      .tab { padding: 11px 4px; font-size: 14px; }
      .tab-note { padding: 10px; }
      .upload-guide { padding: 10px; }
      .book { margin-bottom: 8px; padding: 13px 11px; }
	  .send-link { max-width: 112px; margin-left: 10px; padding: 9px 8px; font-size: 14px; line-height: 1.35; white-space: normal; }
      .title { font-size: 19px; }
      .format-filter { margin: 0 5px 6px 0; padding: 8px 10px; }
      .copy-status { top: 10px; max-width: calc(100vw - 20px); padding: 11px 13px; }
      .support { margin-bottom: 20px; padding: 15px; }
      .support-card { flex-basis: 100%; padding: 11px; }
      .support-card img { width: 104px; height: 104px; }
      .support-card-copy { margin-left: 12px; }
    }
  </style>
</head>
<body>
  <div class="page">
    <header class="brand-header">
      <span class="brand-mark" aria-hidden="true"><svg viewBox="0 0 52 52" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M5.5 10.5c7.2-.9 14 1.1 20.5 6v27c-6.5-4.9-13.3-6.9-20.5-6V10.5Z" stroke="currentColor" stroke-width="2.4" stroke-linejoin="round"/><path d="M46.5 10.5c-7.2-.9-14 1.1-20.5 6v27c6.5-4.9 13.3-6.9 20.5-6V10.5Z" stroke="currentColor" stroke-width="2.4" stroke-linejoin="round"/><path d="M16 27h19m-5-5 5 5-5 5" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"/></svg></span>
      <span class="brand-copy"><strong class="brand-name" data-i18n="brandName">方序传书</strong><span class="brand-tagline" data-i18n="tagline">方寸之间，自有书序</span></span>
      <div class="language-switch" role="group" aria-label="语言 / Language">
        <button class="language-button" type="button" data-language-button="zh" onclick="setLanguage('zh')">中文</button>
        <button class="language-button" type="button" data-language-button="en" onclick="setLanguage('en')">EN</button>
      </div>
    </header>
    {{if .Admin}}
    <div class="admin">
      <div class="admin-header">
        <div class="admin-heading">
          <h2 data-i18n="desktopSettings">电脑端传书设置</h2>
          <p class="important-notice"><strong data-i18n="connectionReminder">连接提醒：</strong><span data-i18n="connectionText">让 Kindle 与电脑连接同一 Wi-Fi，然后在 Kindle 浏览器中打开以下任一地址。</span></p>
        </div>
        <div class="service-actions">
          <span class="service-status"><span class="service-status-dot" aria-hidden="true"></span><span data-i18n="serviceRunning">服务运行中</span></span>
          <form action="/settings/stop" method="post">
            <input type="hidden" name="token" value="{{.CSRFToken}}">
            <button class="danger" type="submit" data-i18n="stopService">停止传书服务</button>
          </form>
        </div>
      </div>
      {{if .NetworkName}}<div class="network-info"><span class="network-label" data-i18n="currentNetwork">当前网络</span><strong class="network-name">{{.NetworkName}}</strong></div>{{end}}
      <div class="address-list">{{range .Addresses}}<strong class="address">{{.}}</strong>{{else}}<span data-i18n="noAddress">未发现局域网 IPv4 地址，请确认电脑已连接 Wi-Fi。</span>{{end}}</div>
      {{if .Status}}<div class="status">{{.Status}}</div>{{end}}
      <form action="/settings/directory" method="post">
        <input type="hidden" name="token" value="{{.CSRFToken}}">
        <label for="path" data-i18n="sharedDirectory">共享电子书目录</label>
        <div class="directory-row">
          <input id="path" type="text" name="path" value="{{.Root}}">
          <div class="directory-actions">
            <button class="primary" type="submit" name="action" value="choose" data-i18n="chooseFolder">选择文件夹...</button>
            <button type="submit" name="action" value="apply" data-i18n="applyPath">应用输入的路径</button>
          </div>
        </div>
      </form>
    </div>
    <section class="support" aria-labelledby="support-title">
      <div class="support-heading">
        <h2 id="support-title" data-i18n="supportTitle">支持与关注</h2>
        <p data-i18n="supportText">方序传书是一个免费项目。如果它为你带来了便利，欢迎请作者喝杯咖啡。感谢大家的支持，让这个免费项目可以持续维护与改进。</p>
      </div>
      <div class="support-grid">
        <div class="support-card">
          <img src="https://cdn.ip21.cn/img/common/alipay-qrcode.jpg" alt="支付宝收款二维码" width="116" height="116" loading="lazy">
          <span class="support-card-copy"><strong class="support-card-title" data-i18n="alipay">支付宝</strong><span class="support-card-note" data-i18n="alipayNote">打开支付宝扫一扫，感谢你的支持。</span></span>
        </div>
        <div class="support-card">
          <img src="https://cdn.ip21.cn/img/common/wechatpay-qrcode.jpg" alt="微信收款二维码" width="116" height="116" loading="lazy">
          <span class="support-card-copy"><strong class="support-card-title" data-i18n="wechatPay">微信支付</strong><span class="support-card-note" data-i18n="wechatPayNote">打开微信扫一扫，感谢你的支持。</span></span>
        </div>
        <div class="support-card support-card-public">
          <img src="https://cdn.ip21.cn/img/common/wechat-pub.png" alt="一灯 AI 微信公众号二维码" width="116" height="116" loading="lazy">
          <span class="support-card-copy"><strong class="support-card-title" data-i18n="followYideng">关注「一灯 AI」</strong><span class="support-card-note" data-i18n="followYidengNote">微信扫码关注公众号，获取 AI 工具与效率实践。</span></span>
        </div>
      </div>
    </section>
    {{end}}
    <div class="shelf-header"><h1 data-i18n="myLibrary">我的书架</h1></div>
	<p class="summary" data-i18n="bookSummary" data-count="{{.BookCount}}">共收录 {{.BookCount}} 本电子书，默认按最近更新排列。</p>
    <div class="tabs" role="tablist" aria-label="传书方式">
      <button class="tab active" id="direct-tab" type="button" role="tab" aria-selected="true" aria-controls="direct-panel" onclick="showTab('direct')" data-i18n="directTab" data-count="{{len .DirectBooks}}">Kindle 内置浏览器下载（{{len .DirectBooks}}）</button>
      {{if .Admin}}
      <button class="tab" id="send-tab" type="button" role="tab" aria-selected="false" aria-controls="send-panel" onclick="showTab('send')" data-i18n="sendTab" data-count="{{len .SendBooks}}">电脑浏览器 Send to Kindle 云端传书（{{len .SendBooks}}）</button>
      {{end}}
    </div>
    <section id="direct-panel" class="tab-panel" role="tabpanel" aria-labelledby="direct-tab">
      <p class="tab-note"><span data-i18n="directNote">在 Kindle 体验版浏览器中，轻点书名即可下载。支持 AZW3、MOBI、TXT、AZW、KFX 和 PRC。</span><span class="important-notice"><strong data-i18n="transferReminder">传书提醒：</strong><span data-i18n="transferReminderText">其他格式无法直接下载，请在电脑端打开本页面，通过电脑浏览器 Send to Kindle 云端传书。</span></span></p>
      <div class="filter-toolbar">
        <div class="format-filters" role="group" aria-label="筛选浏览器下载格式">
          <button class="format-filter active" type="button" data-filter-button onclick="filterBooks('direct', '', this)" data-i18n="allCount" data-count="{{len .DirectBooks}}">全部（{{len .DirectBooks}}）</button>
          {{range .DirectFilters}}<button class="format-filter" type="button" data-filter-button onclick="filterBooks('direct', '{{.Name}}', this)">{{.Name}}（{{.Count}}）</button>{{end}}
        </div>
        <button class="sort-toggle" type="button" onclick="toggleSort()" data-i18n="sortByName">按文件名排序</button>
      </div>
      <div id="direct-filter-empty" class="empty" hidden data-i18n="noFormatBooks">没有这种格式的电子书。</div>
      <div id="direct-list">
      {{range .DirectBooks}}
      <a class="book" href="{{.DownloadURL}}" data-book data-name="{{.Name}}" data-time="{{.ModifiedUnix}}" data-format="{{.Format}}">
        <span class="title">{{.Name}}</span>
        <span class="meta">{{if .Directory}}{{.Directory}} · {{end}}{{.Size}} · {{.Modified}}</span>
      </a>
      {{else}}<div class="empty" data-base-empty data-i18n="noDirectBooks">没有可通过 Kindle 浏览器直接下载的电子书。</div>{{end}}
      </div>
    </section>
    {{if .Admin}}
    <section id="send-panel" class="tab-panel" role="tabpanel" aria-labelledby="send-tab" hidden>
      <p class="tab-note"><span data-i18n="sendNote">Send to Kindle 仅支持 PDF、DOC、DOCX、TXT、RTF、HTM、HTML、PNG、GIF、JPG、JPEG、BMP 和 EPUB，单个文件最大 200 MB。</span><span class="important-notice"><strong data-i18n="fileLimit">文件限制：</strong><span data-i18n="fileLimitText">本页只列出符合格式及大小限制的文件。</span></span><span data-i18n="sendActionNote">点击书籍可复制完整路径；点击“复制并打开Send to Kindle”可复制路径并前往上传页面。</span></p>
      <div class="upload-guide">
        <strong class="upload-guide-title" data-i18n="uploadGuideTitle">选择文件时，快速使用刚复制的完整路径</strong>
        <span class="upload-guide-platform" data-i18n="windowsGuide">Windows：在文件选择窗口的“文件名”输入框按 Ctrl + V 粘贴路径，再点击“打开”。</span>
        <span class="upload-guide-platform" data-i18n="macGuide">macOS：在文件选择窗口按 Command + Shift + G，粘贴路径后按 Return，再点击“打开”。</span>
        <span class="upload-guide-platform" data-i18n="linuxGuide">Linux：在文件选择窗口按 Ctrl + L，粘贴路径后按 Enter，再点击“打开”。</span>
      </div>
      <div class="filter-toolbar">
        <div class="format-filters" role="group" aria-label="筛选 Send to Kindle 格式">
          <button class="format-filter active" type="button" data-filter-button onclick="filterBooks('send', '', this)" data-i18n="allCount" data-count="{{len .SendBooks}}">全部（{{len .SendBooks}}）</button>
          {{range .SendFilters}}<button class="format-filter" type="button" data-filter-button onclick="filterBooks('send', '{{.Name}}', this)">{{.Name}}（{{.Count}}）</button>{{end}}
        </div>
        <button class="sort-toggle" type="button" onclick="toggleSort()" data-i18n="sortByName">按文件名排序</button>
      </div>
      <div id="send-filter-empty" class="empty" hidden data-i18n="noFormatBooks">没有这种格式的电子书。</div>
      <div id="send-list">
      {{range .SendBooks}}
      <div class="book book-copy" role="button" tabindex="0" data-book data-name="{{.Name}}" data-time="{{.ModifiedUnix}}" data-format="{{.Format}}" data-path="{{.FilePath}}" onclick="copyBookPath(event, this)" onkeydown="copyOnKey(event, this)">
		<div class="book-details">
		  <span class="title">{{.Name}}</span>
		  <span class="meta">{{if .Directory}}{{.Directory}} · {{end}}{{.Size}} · {{.Modified}}</span>
		</div>
        <a class="send-link" href="{{$.SendToKindleURL}}" target="_blank" rel="noopener noreferrer" onclick="copyAndOpenSendToKindle(event, this)" data-i18n="copyAndOpen">复制并打开Send to Kindle</a>
      </div>
      {{else}}<div class="empty" data-base-empty data-i18n="noSendBooks">没有需要通过 Send to Kindle 发送的电子书。</div>{{end}}
      </div>
    </section>
    {{end}}
    <div class="footer"><span data-i18n="localOnly">方序传书 · 文件只在你的局域网内流转</span>{{if .Admin}}<span class="footer-root"><span data-i18n="sharedDirectoryPrefix">共享目录：</span>{{.Root}}</span>{{end}}</div>
  </div>
  <div id="copy-status" class="copy-status" role="status" aria-live="polite" hidden></div>
  <script>
    var translations = {
      zh: {
        pageTitle: '方序传书', brandName: '方序传书', tagline: '方寸之间，自有书序',
        desktopSettings: '电脑端传书设置', connectionReminder: '连接提醒：', connectionText: '让 Kindle 与电脑连接同一 Wi-Fi，然后在 Kindle 浏览器中打开以下任一地址。',
        serviceRunning: '服务运行中', stopService: '停止传书服务', currentNetwork: '当前网络', noAddress: '未发现局域网 IPv4 地址，请确认电脑已连接 Wi-Fi。',
        sharedDirectory: '共享电子书目录', chooseFolder: '选择文件夹...', applyPath: '应用输入的路径',
        supportTitle: '支持与关注', supportText: '方序传书是一个免费项目。如果它为你带来了便利，欢迎请作者喝杯咖啡。感谢大家的支持，让这个免费项目可以持续维护与改进。',
        alipay: '支付宝', alipayNote: '打开支付宝扫一扫，感谢你的支持。', wechatPay: '微信支付', wechatPayNote: '打开微信扫一扫，感谢你的支持。',
        followYideng: '关注「一灯 AI」', followYidengNote: '微信扫码关注公众号，获取 AI 工具与效率实践。',
        myLibrary: '我的书架', bookSummary: '共收录 {count} 本电子书，默认按最近更新排列。', directTab: 'Kindle 内置浏览器下载（{count}）', sendTab: '电脑浏览器 Send to Kindle 云端传书（{count}）',
        directNote: '在 Kindle 体验版浏览器中，轻点书名即可下载。支持 AZW3、MOBI、TXT、AZW、KFX 和 PRC。', transferReminder: '传书提醒：', transferReminderText: '其他格式无法直接下载，请在电脑端打开本页面，通过电脑浏览器 Send to Kindle 云端传书。',
        allCount: '全部（{count}）', sortByName: '按文件名排序', sortByTime: '按时间排序', noFormatBooks: '没有这种格式的电子书。', noDirectBooks: '没有可通过 Kindle 浏览器直接下载的电子书。',
        sendNote: 'Send to Kindle 仅支持 PDF、DOC、DOCX、TXT、RTF、HTM、HTML、PNG、GIF、JPG、JPEG、BMP 和 EPUB，单个文件最大 200 MB。', fileLimit: '文件限制：', fileLimitText: '本页只列出符合格式及大小限制的文件。',
        sendActionNote: '点击书籍可复制完整路径；点击“复制并打开Send to Kindle”可复制路径并前往上传页面。', uploadGuideTitle: '选择文件时，快速使用刚复制的完整路径',
        windowsGuide: 'Windows：在文件选择窗口的“文件名”输入框按 Ctrl + V 粘贴路径，再点击“打开”。', macGuide: 'macOS：在文件选择窗口按 Command + Shift + G，粘贴路径后按 Return，再点击“打开”。', linuxGuide: 'Linux：在文件选择窗口按 Ctrl + L，粘贴路径后按 Enter，再点击“打开”。',
        copyAndOpen: '复制并打开Send to Kindle', noSendBooks: '没有需要通过 Send to Kindle 发送的电子书。', localOnly: '方序传书 · 文件只在你的局域网内流转', sharedDirectoryPrefix: '共享目录：',
        copiedPath: '已复制路径：{path}', copyFailed: '复制失败，请手动选择页面中的路径。'
      },
      en: {
        pageTitle: 'Fangxu Kindle Transfer', brandName: 'Fangxu Transfer', tagline: 'BOOKS IN ORDER, WITHIN REACH',
        desktopSettings: 'Desktop transfer settings', connectionReminder: 'Connection: ', connectionText: 'Connect your Kindle and computer to the same Wi-Fi, then open any address below in the Kindle browser.',
        serviceRunning: 'Service running', stopService: 'Stop service', currentNetwork: 'Current network', noAddress: 'No local IPv4 address found. Make sure this computer is connected to Wi-Fi.',
        sharedDirectory: 'Shared ebook folder', chooseFolder: 'Choose folder...', applyPath: 'Use entered path',
        supportTitle: 'Support & follow', supportText: 'Fangxu Transfer is free and open source. If it saves you time, you can buy the author a coffee. Thank you for helping this free project stay maintained and improve.',
        alipay: 'Alipay', alipayNote: 'Scan with Alipay. Thank you for your support.', wechatPay: 'WeChat Pay', wechatPayNote: 'Scan with WeChat. Thank you for your support.',
        followYideng: 'Follow Yideng AI', followYidengNote: 'Scan in WeChat for AI tools and productivity tips.',
        myLibrary: 'My library', bookSummary: '{count} ebooks found, newest files first.', directTab: 'Kindle browser download ({count})', sendTab: 'Desktop Send to Kindle cloud transfer ({count})',
        directNote: 'Tap a book title in the Kindle browser to download it. Supported formats: AZW3, MOBI, TXT, AZW, KFX, and PRC.', transferReminder: 'Transfer tip: ', transferReminderText: 'Other formats cannot be downloaded directly. Open this page on your computer and use Send to Kindle cloud transfer.',
        allCount: 'All ({count})', sortByName: 'Sort by filename', sortByTime: 'Sort by time', noFormatBooks: 'No ebooks in this format.', noDirectBooks: 'No ebooks available for direct Kindle browser download.',
        sendNote: 'Send to Kindle supports PDF, DOC, DOCX, TXT, RTF, HTM, HTML, PNG, GIF, JPG, JPEG, BMP, and EPUB, with a maximum file size of 200 MB.', fileLimit: 'File limits: ', fileLimitText: 'Only files that meet the format and size limits are listed here.',
        sendActionNote: 'Click a book to copy its full path. Use “Copy & open Send to Kindle” to copy the path and open the upload page.', uploadGuideTitle: 'Quickly select a file using the copied full path',
        windowsGuide: 'Windows: paste the path with Ctrl + V into the File name field, then click Open.', macGuide: 'macOS: press Command + Shift + G, paste the path, press Return, then click Open.', linuxGuide: 'Linux: press Ctrl + L in the file picker, paste the path, press Enter, then click Open.',
        copyAndOpen: 'Copy & open Send to Kindle', noSendBooks: 'No ebooks are available for Send to Kindle.', localOnly: 'Fangxu Transfer · Files stay on your local network', sharedDirectoryPrefix: 'Shared folder: ',
        copiedPath: 'Path copied: {path}', copyFailed: 'Copy failed. Please select the path manually.'
      }
    };
    var currentLanguage = 'zh';
    var sortMode = 'time';
    function message(key, values) {
      var text = translations[currentLanguage][key] || translations.zh[key] || key;
      values = values || {};
      return text.replace(/\{([^}]+)\}/g, function (_, name) { return values[name] === undefined ? '' : values[name]; });
    }
    function setLanguage(language) {
      currentLanguage = language === 'en' ? 'en' : 'zh';
      document.documentElement.lang = currentLanguage === 'en' ? 'en' : 'zh-CN';
      document.title = message('pageTitle');
      var elements = document.querySelectorAll('[data-i18n]');
      for (var index = 0; index < elements.length; index++) {
        elements[index].textContent = message(elements[index].getAttribute('data-i18n'), { count: elements[index].getAttribute('data-count') });
      }
      var languageButtons = document.querySelectorAll('[data-language-button]');
      for (var buttonIndex = 0; buttonIndex < languageButtons.length; buttonIndex++) {
        languageButtons[buttonIndex].className = languageButtons[buttonIndex].getAttribute('data-language-button') === currentLanguage ? 'language-button active' : 'language-button';
      }
      updateSortButtons();
      try { window.localStorage.setItem('fangxu-language', currentLanguage); } catch (_) {}
    }
    function initialLanguage() {
      try {
        var saved = window.localStorage.getItem('fangxu-language');
        if (saved === 'zh' || saved === 'en') return saved;
      } catch (_) {}
      var browserLanguage = navigator.language || navigator.userLanguage || 'zh';
      return browserLanguage.toLowerCase().indexOf('zh') === 0 ? 'zh' : 'en';
    }
    function updateSortButtons() {
      var sortButtons = document.querySelectorAll('.sort-toggle');
      for (var index = 0; index < sortButtons.length; index++) {
        sortButtons[index].textContent = message(sortMode === 'time' ? 'sortByName' : 'sortByTime');
      }
    }
    function toggleSort() {
      sortMode = sortMode === 'time' ? 'name' : 'time';
      sortBookList('direct-list');
      sortBookList('send-list');
      updateSortButtons();
    }
    function sortBookList(listID) {
      var list = document.getElementById(listID);
      if (!list) return;
      var books = Array.prototype.slice.call(list.querySelectorAll('[data-book]'));
      books.sort(function (left, right) {
        var leftName = left.getAttribute('data-name').toLowerCase();
        var rightName = right.getAttribute('data-name').toLowerCase();
        if (sortMode === 'name') return leftName.localeCompare(rightName);
        var timeDifference = Number(right.getAttribute('data-time')) - Number(left.getAttribute('data-time'));
        return timeDifference || leftName.localeCompare(rightName);
      });
      books.forEach(function (item) { list.appendChild(item); });
    }
    function showTab(name) {
      var direct = name === 'direct';
      var sendPanel = document.getElementById('send-panel');
      var sendTab = document.getElementById('send-tab');
      if (!sendPanel || !sendTab) return;
      document.getElementById('direct-panel').hidden = !direct;
      sendPanel.hidden = direct;
      document.getElementById('direct-tab').className = direct ? 'tab active' : 'tab';
      sendTab.className = direct ? 'tab' : 'tab active';
      document.getElementById('direct-tab').setAttribute('aria-selected', direct ? 'true' : 'false');
      sendTab.setAttribute('aria-selected', direct ? 'false' : 'true');
    }
    function filterBooks(group, format, selectedButton) {
      var list = document.getElementById(group + '-list');
      if (!list) return;
      var books = list.querySelectorAll('[data-book]');
      var visibleCount = 0;
      for (var index = 0; index < books.length; index++) {
        var visible = !format || books[index].getAttribute('data-format') === format;
        books[index].hidden = !visible;
        if (visible) visibleCount++;
      }
      var baseEmpty = list.querySelector('[data-base-empty]');
      if (baseEmpty) baseEmpty.hidden = !!format;
      var panel = document.getElementById(group + '-panel');
      var buttons = panel.querySelectorAll('[data-filter-button]');
      for (var buttonIndex = 0; buttonIndex < buttons.length; buttonIndex++) {
        buttons[buttonIndex].className = buttons[buttonIndex] === selectedButton ? 'format-filter active' : 'format-filter';
      }
      document.getElementById(group + '-filter-empty').hidden = !format || visibleCount !== 0;
    }
    function copyOnKey(event, element) {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        copyBookPath(event, element);
      }
    }
    function copyBookPath(event, element) {
      var path = element.getAttribute('data-path');
      copyPath(path);
    }
    function copyPath(path, onComplete) {
      if (navigator.clipboard && window.isSecureContext) {
        navigator.clipboard.writeText(path).then(function () {
          showCopyStatus(message('copiedPath', { path: path }));
          if (onComplete) onComplete();
        }, function () { fallbackCopy(path, onComplete); });
      } else {
        fallbackCopy(path, onComplete);
      }
    }
    function fallbackCopy(path, onComplete) {
      var input = document.createElement('textarea');
      input.value = path;
      input.style.position = 'fixed';
      input.style.opacity = '0';
      document.body.appendChild(input);
      input.select();
      var copied = document.execCommand('copy');
      document.body.removeChild(input);
      showCopyStatus(copied ? message('copiedPath', { path: path }) : message('copyFailed'));
      if (onComplete) onComplete();
    }
    function copyAndOpenSendToKindle(event, link) {
      event.preventDefault();
      event.stopPropagation();
      var path = link.parentNode.getAttribute('data-path');
      var destination = link.href;
      copyPath(path);
      var sendWindow = window.open(destination, '_blank');
      if (sendWindow) sendWindow.opener = null;
      else window.location.href = destination;
    }
    function showCopyStatus(message) {
      var status = document.getElementById('copy-status');
      status.textContent = message;
      status.hidden = false;
      window.clearTimeout(window.copyStatusTimer);
      window.copyStatusTimer = window.setTimeout(function () { status.hidden = true; }, 3000);
    }
    setLanguage(initialLanguage());
  </script>
</body>
</html>`
