package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// The desktop protocol travels only over inherited pipes, never over the LAN.
// Closing the parent's stdin stops the service even if the desktop crashes.
type desktopSettings struct {
	Directory string `json:"directory"`
	Port      int    `json:"port"`
	Language  string `json:"language"`
}
type desktopStatus struct {
	Running     bool            `json:"running"`
	ControlURL  string          `json:"controlUrl"`
	Addresses   []string        `json:"addresses"`
	NetworkName string          `json:"networkName"`
	StartedAt   string          `json:"startedAt"`
	LastError   string          `json:"lastError"`
	Settings    desktopSettings `json:"settings"`
}
type desktopRequest struct {
	Command  string           `json:"command"`
	Settings *desktopSettings `json:"settings,omitempty"`
}
type desktopResponse struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}
type desktopManager struct {
	configPath   string
	instancePath string
	settings     desktopSettings
	server       *http.Server
	done         chan error
	status       desktopStatus
}

func loadDesktopSettings(path string) (desktopSettings, error) {
	// An empty language lets a new client follow the system language. Persisted
	// language preferences take precedence. Legacy autoStart values are ignored.
	settings := desktopSettings{Directory: defaultBookDirectory()}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return settings, err
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return settings, fmt.Errorf("设置文件损坏: %w", err)
	}
	if settings.Port < 0 || settings.Port > 65535 {
		return settings, errors.New("设置文件中的端口无效")
	}
	if settings.Language != "zh-CN" && settings.Language != "en" {
		settings.Language = ""
	}
	return settings, nil
}

func validateDesktopSettings(settings desktopSettings) (desktopSettings, error) {
	if settings.Port < 0 || settings.Port > 65535 {
		return settings, errors.New("端口必须为 0 到 65535，0 表示自动分配")
	}
	if settings.Language != "" && settings.Language != "zh-CN" && settings.Language != "en" {
		return settings, errors.New("不支持的语言")
	}
	if strings.TrimSpace(settings.Directory) == "" {
		return settings, errors.New("请选择电子书目录")
	}
	root, err := filepath.Abs(strings.TrimSpace(settings.Directory))
	if err != nil {
		return settings, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return settings, fmt.Errorf("无法读取目录: %w", err)
	}
	if !info.IsDir() {
		return settings, errors.New("电子书目录必须是文件夹")
	}
	settings.Directory = root
	return settings, nil
}

func saveDesktopSettings(path string, settings desktopSettings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	// Write a temporary file first so an interrupted write doesn't corrupt settings.
	file, err := os.CreateTemp(filepath.Dir(path), ".settings-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func (m *desktopManager) refresh() {
	if m.done == nil {
		return
	}
	select {
	case err := <-m.done:
		m.server, m.done = nil, nil
		m.status = desktopStatus{Addresses: []string{}}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			m.status.LastError = err.Error()
		}
	default:
	}
}
func (m *desktopManager) snapshot() desktopStatus {
	m.refresh()
	status := m.status
	status.Settings = m.settings
	if status.Addresses == nil {
		status.Addresses = []string{}
	}
	return status
}
func (m *desktopManager) start() error {
	m.refresh()
	if m.server != nil {
		return nil
	}
	settings, err := validateDesktopSettings(m.settings)
	if err != nil {
		return err
	}
	handler, err := newApp(settings.Directory)
	if err != nil {
		return err
	}
	statePath := m.instancePath
	if statePath == "" {
		statePath, err = serviceStatePath()
		if err != nil {
			return err
		}
	}
	claim, existing, err := acquireInstance(statePath, 3*time.Second)
	if err != nil {
		return err
	}
	if existing != "" {
		return fmt.Errorf("已有传书服务运行于 %s，请先关闭该服务", existing)
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("0.0.0.0", strconv.Itoa(settings.Port)))
	if err != nil {
		claim.release()
		return fmt.Errorf("端口无法使用: %w", err)
	}
	controlURL := "http://127.0.0.1" + displayPort(listener.Addr())
	if err := claim.publish(controlURL); err != nil {
		listener.Close()
		claim.release()
		return err
	}
	handler.instanceToken = claim.token
	handler.desktopManaged = true
	handler.setAddresses(localNetworkURLs(listener.Addr()))
	handler.setNetworkName(currentNetworkName())
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	m.server = server
	m.done = make(chan error, 1)
	m.status = desktopStatus{Running: true, ControlURL: controlURL, Addresses: handler.pageAddresses(), NetworkName: handler.pageNetworkName(), StartedAt: time.Now().Format(time.RFC3339)}
	done := m.done
	go func() {
		err := server.Serve(listener)
		claim.release()
		done <- err
	}()
	log.Printf("传书服务已启动: %s", controlURL)
	return nil
}
func (m *desktopManager) stop() {
	if m.server == nil {
		return
	}
	_ = m.server.Close()
	<-m.done // Wait for the instance claim to be released before allowing restart.
	m.server, m.done = nil, nil
	m.status = desktopStatus{Addresses: []string{}}
	log.Print("传书服务已停止")
}

func (m *desktopManager) handle(request desktopRequest) (any, error) {
	switch request.Command {
	case "status":
		return m.snapshot(), nil
	case "start":
		if err := m.start(); err != nil {
			return nil, err
		}
		return m.snapshot(), nil
	case "stop":
		m.stop()
		return m.snapshot(), nil
	case "language":
		if request.Settings == nil || (request.Settings.Language != "zh-CN" && request.Settings.Language != "en") {
			return nil, errors.New("不支持的语言")
		}
		// Language is independent of sharing and unsaved folder/port edits.
		settings := m.settings
		settings.Language = request.Settings.Language
		if err := saveDesktopSettings(m.configPath, settings); err != nil {
			return nil, fmt.Errorf("保存语言失败: %w", err)
		}
		m.settings = settings
		return m.snapshot(), nil
	case "save":
		m.refresh()
		if request.Settings == nil {
			return nil, errors.New("缺少设置")
		}
		settings, err := validateDesktopSettings(*request.Settings)
		if err != nil {
			return nil, err
		}
		if err := saveDesktopSettings(m.configPath, settings); err != nil {
			return nil, fmt.Errorf("保存设置失败: %w", err)
		}
		previous := m.settings
		wasRunning := m.server != nil
		if wasRunning {
			m.stop()
		}
		m.settings = settings
		m.status.LastError = ""
		if wasRunning {
			if err := m.start(); err != nil {
				// Restore the working settings and sharing if the new port cannot bind.
				m.settings = previous
				if restoreErr := saveDesktopSettings(m.configPath, previous); restoreErr != nil {
					err = errors.Join(err, fmt.Errorf("保存设置失败: %w", restoreErr))
				}
				if restartErr := m.start(); restartErr != nil {
					err = errors.Join(err, restartErr)
					m.status.LastError = err.Error()
				}
				return nil, err
			}
		}
		log.Print("客户端设置已保存")
		return m.snapshot(), nil
	default:
		return nil, errors.New("未知客户端命令")
	}
}

func runDesktop(input io.Reader, output io.Writer, configPath string) error {
	if configPath == "" {
		return errors.New("桌面模式需要 --config 设置文件路径")
	}
	settings, loadErr := loadDesktopSettings(configPath)
	manager := &desktopManager{settings: settings, configPath: configPath}
	return manager.run(input, output, loadErr)
}

func (manager *desktopManager) run(input io.Reader, output io.Writer, loadErr error) error {
	defer manager.stop()
	if loadErr != nil {
		manager.status.LastError = loadErr.Error()
	} else {
		if err := manager.start(); err != nil {
			manager.status.LastError = err.Error()
		}
	}
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	encoder := json.NewEncoder(output)
	for scanner.Scan() {
		var request desktopRequest
		var response desktopResponse
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			response.Error = "客户端请求格式无效"
		} else {
			data, err := manager.handle(request)
			if err != nil {
				response.Error = err.Error()
				log.Printf("客户端操作失败 (%s): %v", request.Command, err)
			} else {
				response.Data = data
			}
		}
		if err := encoder.Encode(response); err != nil {
			return err
		}
	}
	return scanner.Err()
}
