package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDesktopSettingsPersistenceAndValidation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(t.TempDir(), "settings.json")
	settings := desktopSettings{Directory: root, Port: 0, Language: "en"}
	if err := saveDesktopSettings(path, settings); err != nil {
		t.Fatal(err)
	}
	// Saving over an existing file also needs to work on Windows.
	settings.Port = 12345
	if err := saveDesktopSettings(path, settings); err != nil {
		t.Fatal(err)
	}
	restored, err := loadDesktopSettings(path)
	if err != nil || restored != settings {
		t.Fatalf("restored = %+v, error = %v", restored, err)
	}
	for _, invalid := range []desktopSettings{
		{Directory: root, Port: -1, Language: "en"},
		{Directory: root, Port: 65536, Language: "en"},
		{Directory: "", Language: "en"},
		{Directory: filepath.Join(root, "missing"), Language: "en"},
		{Directory: root, Language: "unsupported"},
	} {
		if _, err := validateDesktopSettings(invalid); err == nil {
			t.Errorf("accepted invalid settings: %+v", invalid)
		}
	}
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadDesktopSettings(path); err == nil {
		t.Fatal("silently accepted corrupt settings")
	}
}

func TestDesktopServiceLifecycleAndOwnership(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "book.azw3"), "ebook")
	manager := &desktopManager{
		configPath:   filepath.Join(t.TempDir(), "settings.json"),
		instancePath: filepath.Join(t.TempDir(), "service.json"),
		settings:     desktopSettings{Directory: root, Language: "zh-CN"},
	}
	defer manager.stop()
	if err := manager.start(); err != nil {
		t.Fatal(err)
	}
	status := manager.snapshot()
	if !status.Running || status.ControlURL == "" {
		t.Fatalf("not started: %+v", status)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(status.ControlURL + "/download/book.azw3")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("download status = %d", response.StatusCode)
	}
	for _, endpoint := range []string{"/settings/stop", "/settings/directory"} {
		response, err := client.Post(status.ControlURL+endpoint, "application/x-www-form-urlencoded", strings.NewReader("token=ignored"))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusForbidden {
			t.Fatalf("desktop control exposed over HTTP: %s", endpoint)
		}
	}
	claim, existing, err := acquireInstance(manager.instancePath, 0)
	if err != nil || claim != nil || existing != status.ControlURL {
		t.Fatalf("instance ownership = %v, %q, %v", claim, existing, err)
	}
	manager.stop()
	if manager.snapshot().Running {
		t.Fatal("service still running")
	}
	if _, err := os.Stat(manager.instancePath); !os.IsNotExist(err) {
		t.Fatalf("claim remains: %v", err)
	}
	if response, err := client.Get(status.ControlURL); err == nil {
		response.Body.Close()
		t.Fatal("stopped service still reachable")
	}
	if err := manager.start(); err != nil {
		t.Fatalf("cannot restart: %v", err)
	}
}

func TestDesktopSaveRestartsWithNewSettingsAndRecoversFromErrors(t *testing.T) {
	oldRoot, newRoot := t.TempDir(), t.TempDir()
	mustWriteFile(t, filepath.Join(oldRoot, "old.azw3"), "old")
	mustWriteFile(t, filepath.Join(newRoot, "new.azw3"), "new")
	manager := &desktopManager{
		configPath:   filepath.Join(t.TempDir(), "settings.json"),
		instancePath: filepath.Join(t.TempDir(), "service.json"),
		settings:     desktopSettings{Directory: oldRoot, Language: "zh-CN"},
	}
	defer manager.stop()
	if err := manager.start(); err != nil {
		t.Fatal(err)
	}
	previousServer := manager.server
	previousURL := manager.snapshot().ControlURL
	invalid := desktopSettings{Directory: filepath.Join(newRoot, "missing")}
	if _, err := manager.handle(desktopRequest{Command: "save", Settings: &invalid}); err == nil {
		t.Fatal("invalid folder accepted")
	}
	if manager.server != previousServer {
		t.Fatal("validation error interrupted sharing")
	}
	// Reserve a different port while selecting it, then release it for the restart.
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	settings := desktopSettings{Directory: newRoot, Port: port, Language: "en"}
	result, err := manager.handle(desktopRequest{Command: "save", Settings: &settings})
	if err != nil {
		t.Fatal(err)
	}
	status := result.(desktopStatus)
	if !status.Running || status.Settings != settings || manager.server == previousServer || status.ControlURL != fmt.Sprintf("http://127.0.0.1:%d", port) {
		t.Fatalf("settings were not applied by restarting: %+v", status)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	for _, tc := range []struct {
		path string
		code int
	}{{"/download/new.azw3", http.StatusOK}, {"/download/old.azw3", http.StatusNotFound}} {
		response, err := client.Get(status.ControlURL + tc.path)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != tc.code {
			t.Fatalf("%s status = %d", tc.path, response.StatusCode)
		}
	}
	if response, err := client.Get(previousURL); err == nil {
		response.Body.Close()
		t.Fatal("old endpoint still reachable")
	}
	if restored, err := loadDesktopSettings(manager.configPath); err != nil || restored != settings {
		t.Fatalf("saved settings = %+v, %v", restored, err)
	}
	occupied, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	badPort := settings
	badPort.Port = occupied.Addr().(*net.TCPAddr).Port
	if _, err := manager.handle(desktopRequest{Command: "save", Settings: &badPort}); err == nil {
		t.Fatal("occupied port accepted")
	}
	if current := manager.snapshot(); !current.Running || current.Settings != settings || current.ControlURL != status.ControlURL {
		t.Fatalf("working service was not restored: %+v", current)
	}
	if restored, err := loadDesktopSettings(manager.configPath); err != nil || restored != settings {
		t.Fatalf("failed settings were persisted: %+v, %v", restored, err)
	}
	previousServer = manager.server
	manager.configPath = newRoot // Writing settings over a directory must fail before stopping.
	if _, err := manager.handle(desktopRequest{Command: "save", Settings: &settings}); err == nil {
		t.Fatal("save to directory succeeded")
	}
	if manager.server != previousServer {
		t.Fatal("persistence error interrupted sharing")
	}
}

func TestDesktopProtocolRecoversFromMalformedRequests(t *testing.T) {
	root := t.TempDir()
	settings := desktopSettings{Directory: root, Language: "en"}
	manager := &desktopManager{settings: settings, configPath: filepath.Join(root, "settings.json"), instancePath: filepath.Join(root, "service.json")}
	data, _ := json.Marshal(settings)
	input := fmt.Sprintf("{broken\n{\"command\":\"unknown\"}\n{\"command\":\"stop\"}\n{\"command\":\"save\",\"settings\":%s}\n{\"command\":\"status\"}\n", data)
	var output bytes.Buffer
	if err := manager.run(strings.NewReader(input), &output, nil); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&output)
	for i := 0; i < 5; i++ {
		var response struct {
			Data  desktopStatus `json:"data"`
			Error string        `json:"error"`
		}
		if err := decoder.Decode(&response); err != nil {
			t.Fatal(err)
		}
		if i < 2 && response.Error == "" {
			t.Fatal("invalid request was accepted")
		}
		if i >= 2 && response.Error != "" {
			t.Fatalf("request %d: %s", i, response.Error)
		}
		if i == 4 && (response.Data.Running || response.Data.Settings != settings) {
			t.Fatalf("status = %+v", response.Data)
		}
	}
}

func TestDesktopLegacySettingsKeepFolderPortAndLanguage(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "settings.json")
	legacy := fmt.Sprintf(`{"directory":%q,"port":12345,"language":"en","autoStart":false}`, root)
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	settings, err := loadDesktopSettings(path)
	expected := desktopSettings{Directory: root, Port: 12345, Language: "en"}
	if err != nil || settings != expected {
		t.Fatalf("legacy settings = %+v, %v", settings, err)
	}
}

func TestDesktopLanguageCanChangeWhileSharing(t *testing.T) {
	settings := desktopSettings{Directory: t.TempDir(), Port: 0, Language: "zh-CN"}
	manager := &desktopManager{configPath: filepath.Join(t.TempDir(), "settings.json"), instancePath: filepath.Join(t.TempDir(), "service.json"), settings: settings}
	defer manager.stop()
	if err := manager.start(); err != nil {
		t.Fatal(err)
	}
	before := manager.snapshot()
	// A language-only request must ignore directory/port values in a draft form.
	_, err := manager.handle(desktopRequest{Command: "language", Settings: &desktopSettings{Language: "en", Directory: "missing", Port: 65536}})
	if err != nil {
		t.Fatal(err)
	}
	settings.Language = "en"
	after := manager.snapshot()
	if !after.Running || after.ControlURL != before.ControlURL || after.Settings != settings {
		t.Fatalf("language switch changed sharing: %+v", after)
	}
	restored, err := loadDesktopSettings(manager.configPath)
	if err != nil || restored != settings {
		t.Fatalf("language not persisted: %+v, %v", restored, err)
	}
	if _, err := manager.handle(desktopRequest{Command: "language", Settings: &desktopSettings{Language: "fr"}}); err == nil {
		t.Fatal("unsupported language accepted")
	}
}

func TestDesktopStartupBeforeFirstRequestAndEOFCleanup(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprintf("legacy-opt-out-%t", legacy), func(t *testing.T) {
			root := t.TempDir()
			config := filepath.Join(root, "settings.json")
			settings := desktopSettings{Directory: root}
			if legacy {
				data := fmt.Sprintf(`{"directory":%q,"port":0,"autoStart":false}`, root)
				if err := os.WriteFile(config, []byte(data), 0o600); err != nil {
					t.Fatal(err)
				}
				var err error
				settings, err = loadDesktopSettings(config)
				if err != nil {
					t.Fatal(err)
				}
			}
			manager := &desktopManager{configPath: config, instancePath: filepath.Join(root, "service.json"), settings: settings}
			reader, writer := io.Pipe()
			defer reader.Close()
			defer writer.Close()
			observed := make(chan desktopStatus, 1)
			input := &startupObserver{Reader: reader, onRead: func() { observed <- manager.snapshot() }}
			finished := make(chan error, 1)
			go func() { finished <- manager.run(input, io.Discard, nil) }()
			var status desktopStatus
			select {
			case status = <-observed:
			case <-time.After(5 * time.Second):
				t.Fatal("startup blocked before reading input")
			}
			if !status.Running || status.ControlURL == "" {
				t.Fatalf("not started before first request: %+v", status)
			}
			client := &http.Client{Timeout: 2 * time.Second}
			response, err := client.Get(status.ControlURL)
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			writer.Close()
			select {
			case err := <-finished:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("EOF did not stop service")
			}
			if _, err := os.Stat(manager.instancePath); !os.IsNotExist(err) {
				t.Fatalf("instance remains after EOF: %v", err)
			}
			if response, err := client.Get(status.ControlURL); err == nil {
				response.Body.Close()
				t.Fatal("service remains reachable after EOF")
			}
		})
	}
}

type startupObserver struct {
	io.Reader
	onRead func()
}

func (r *startupObserver) Read(p []byte) (int, error) {
	if r.onRead != nil {
		r.onRead()
		r.onRead = nil
	}
	return r.Reader.Read(p)
}

func TestDesktopStartupErrorKeepsSettingsRecoverable(t *testing.T) {
	root := t.TempDir()
	manager := &desktopManager{
		configPath:   filepath.Join(root, "settings.json"),
		instancePath: filepath.Join(root, "service.json"),
		settings:     desktopSettings{Directory: filepath.Join(root, "missing")},
	}
	valid := desktopSettings{Directory: root, Language: "en"}
	data, _ := json.Marshal(valid)
	var output bytes.Buffer
	input := fmt.Sprintf("{\"command\":\"status\"}\n{\"command\":\"save\",\"settings\":%s}\n", data)
	if err := manager.run(strings.NewReader(input), &output, nil); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&output)
	var response struct {
		Data  desktopStatus `json:"data"`
		Error string        `json:"error"`
	}
	if err := decoder.Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Running || !strings.Contains(response.Data.LastError, "无法读取目录") {
		t.Fatalf("missing startup error: %+v", response)
	}
	if err := decoder.Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Error != "" || response.Data.LastError != "" || response.Data.Settings != valid {
		t.Fatalf("could not recover settings: %+v", response)
	}
}
