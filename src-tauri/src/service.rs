use serde_json::{json, Value};
use std::{
    collections::VecDeque,
    io::{BufRead, BufReader, Write},
    path::PathBuf,
    process::{Child, ChildStdin, Command, Stdio},
    sync::{mpsc, Arc, Mutex},
    time::{Duration, Instant},
};
use tauri::Manager;

#[derive(Default)]
pub struct ServiceController {
    backend: Mutex<Option<Backend>>,
}

struct Backend {
    child: Child,
    input: Option<ChildStdin>,
    responses: mpsc::Receiver<Result<String, String>>,
    logs: Arc<Mutex<VecDeque<String>>>,
    healthy: bool,
}

fn sidecar_path() -> Result<PathBuf, String> {
    let executable = std::env::current_exe().map_err(|e| e.to_string())?;
    let name = format!("fangxu-kindle-service{}", std::env::consts::EXE_SUFFIX);
    let bundled = executable.with_file_name(name);
    if bundled.is_file() {
        return Ok(bundled);
    }
    #[cfg(debug_assertions)]
    {
        let development = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("binaries")
            .join(format!(
                "fangxu-kindle-service-{}{}",
                env!("KINDLE_TARGET"),
                std::env::consts::EXE_SUFFIX
            ));
        if development.is_file() {
            return Ok(development);
        }
    }
    Err("找不到传书服务，请完整解压免安装包或重新安装客户端（开发环境请运行 npm run build:sidecar）".into())
}

impl Backend {
    fn spawn(config: PathBuf) -> Result<Self, String> {
        let mut command = Command::new(sidecar_path()?);
        command
            .args(["--desktop", "--config"])
            .arg(config)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());
        #[cfg(windows)]
        {
            use std::os::windows::process::CommandExt;
            command.creation_flags(0x08000000); // CREATE_NO_WINDOW
        }
        let mut child = command
            .spawn()
            .map_err(|e| format!("无法启动传书服务: {e}"))?;
        let input = child.stdin.take();
        let output = child.stdout.take().ok_or("无法读取服务输出")?;
        let stderr = child.stderr.take().ok_or("无法读取服务日志")?;
        let (sender, responses) = mpsc::channel();
        std::thread::spawn(move || {
            for line in BufReader::new(output).lines() {
                if sender.send(line.map_err(|e| e.to_string())).is_err() {
                    return;
                }
            }
        });
        let logs = Arc::new(Mutex::new(VecDeque::new()));
        let log_buffer = logs.clone();
        std::thread::spawn(move || {
            for line in BufReader::new(stderr).lines().map_while(Result::ok) {
                if let Ok(mut buffer) = log_buffer.lock() {
                    if buffer.len() == 100 {
                        buffer.pop_front();
                    }
                    buffer.push_back(line);
                }
            }
        });
        Ok(Self {
            child,
            input,
            responses,
            logs,
            healthy: true,
        })
    }

    fn request(&mut self, command: &str, settings: Option<Value>) -> Result<Value, String> {
        self.healthy = false;
        if let Some(status) = self.child.try_wait().map_err(|e| e.to_string())? {
            return Err(format!("传书服务已退出 ({status})"));
        }
        let request = json!({ "command": command, "settings": settings });
        let input = self.input.as_mut().ok_or("服务连接已关闭")?;
        writeln!(input, "{request}")
            .and_then(|_| input.flush())
            .map_err(|e| e.to_string())?;
        let line = self
            .responses
            .recv_timeout(Duration::from_secs(45))
            .map_err(|e| format!("传书服务未响应: {e}"))??;
        let response: Value =
            serde_json::from_str(&line).map_err(|e| format!("服务响应无效: {e}"))?;
        self.healthy = true;
        if let Some(error) = response.get("error").and_then(Value::as_str) {
            return Err(error.to_owned());
        }
        Ok(response.get("data").cloned().unwrap_or(Value::Null))
    }
}

impl Drop for Backend {
    fn drop(&mut self) {
        self.input.take(); // Go treats EOF as shutdown and releases its listener/claim.
        let deadline = Instant::now() + Duration::from_secs(2);
        loop {
            match self.child.try_wait() {
                Ok(Some(_)) => return,
                Ok(None) if Instant::now() < deadline => {
                    std::thread::sleep(Duration::from_millis(20))
                }
                _ => break,
            }
        }
        let _ = self.child.kill();
        let _ = self.child.wait();
    }
}

impl ServiceController {
    pub fn initialize(&self, app: &tauri::AppHandle) -> Result<(), String> {
        let config = app
            .path()
            .app_config_dir()
            .map_err(|e| e.to_string())?
            .join("settings.json");
        let mut slot = self.backend.lock().map_err(|_| "服务状态不可用")?;
        if slot.is_none() {
            *slot = Some(Backend::spawn(config)?);
        }
        Ok(())
    }

    pub fn shutdown(&self) {
        if let Ok(mut backend) = self.backend.lock() {
            backend.take();
        }
    }

    fn request(
        &self,
        config: PathBuf,
        command: &str,
        settings: Option<Value>,
    ) -> Result<Value, String> {
        if !["status", "save", "language", "start", "stop", "logs"].contains(&command) {
            return Err("未知客户端命令".into());
        }
        let mut slot = self.backend.lock().map_err(|_| "服务状态不可用")?;
        if slot.is_none() {
            *slot = Some(Backend::spawn(config)?);
        }
        let backend = slot.as_mut().unwrap();
        if command == "logs" {
            let logs = backend.logs.lock().map_err(|_| "日志不可用")?;
            return Ok(json!(logs.iter().collect::<Vec<_>>()));
        }
        let result = backend.request(command, settings);
        // Discard a broken pipe or timed-out protocol so responses can't become misaligned.
        if !backend.healthy {
            slot.take();
        }
        result
    }
}

#[tauri::command]
pub async fn desktop_request(
    app: tauri::AppHandle,
    command: String,
    settings: Option<Value>,
) -> Result<Value, String> {
    let config = app
        .path()
        .app_config_dir()
        .map_err(|e| e.to_string())?
        .join("settings.json");
    tauri::async_runtime::spawn_blocking(move || {
        app.state::<ServiceController>()
            .request(config, &command, settings)
    })
    .await
    .map_err(|e| e.to_string())?
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bundled_go_protocol_persists_settings_and_recovers_from_validation_errors() {
        let root = std::env::temp_dir().join(format!("kindle-rust-test-{}", std::process::id()));
        std::fs::create_dir_all(&root).unwrap();
        let config = root.join("settings.json");
        let settings = json!({"directory": root, "port": 0, "language": "en"});
        std::fs::write(&config, serde_json::to_vec(&settings).unwrap()).unwrap();
        let mut backend = Backend::spawn(config.clone()).unwrap();
        assert_eq!(backend.request("status", None).unwrap()["running"], true);
        let restarted = backend.request("save", Some(settings.clone())).unwrap();
        assert_eq!(restarted["running"], true);
        assert_eq!(restarted["settings"], settings);
        assert!(backend.request("save", Some(json!({"port": -1}))).is_err());
        assert_eq!(backend.request("status", None).unwrap()["running"], true);
        backend.request("stop", None).unwrap();
        let status = backend.request("save", Some(settings.clone())).unwrap();
        assert_eq!(status["settings"], settings);
        assert_eq!(status["running"], false);
        assert!(backend.request("save", Some(json!({"port": -1}))).is_err());
        assert!(backend.healthy); // A business error must not break the next response.
        let pid = backend.child.id();
        drop(backend); // EOF must allow Go to finish without leaving a child process.
        let mut restored = Backend::spawn(config).unwrap();
        assert_ne!(restored.child.id(), pid);
        assert_eq!(
            restored.request("status", None).unwrap()["settings"],
            settings
        );
        assert_eq!(restored.request("status", None).unwrap()["running"], true);
        drop(restored);
        std::fs::remove_dir_all(root).unwrap();
    }
}
