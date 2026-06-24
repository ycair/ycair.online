use std::sync::{Arc, Mutex};
use std::io::{BufRead, BufReader, Seek, SeekFrom};
use std::fs::File;
use std::process::Command as StdCommand;
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, State};
use tauri_plugin_shell::ShellExt;
use tauri_plugin_shell::process::CommandChild;
#[cfg(not(target_os = "macos"))]
use tauri_plugin_shell::process::CommandEvent;

#[derive(Debug, Clone, Serialize, Deserialize)]
struct StatusPeer {
    id: String,
    ip: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct CoreStatus {
    #[serde(rename = "type")]
    msg_type: String,
    assigned_ip: String,
    peer_id: String,
    peers: Vec<StatusPeer>,
    tun: String,
    connected: bool,
}

struct AppState {
    status: Arc<Mutex<Option<CoreStatus>>>,
    core_child: Arc<Mutex<Option<CommandChild>>>,
    signal_child: Arc<Mutex<Option<CommandChild>>>,
}

fn parse_status_line(data: &[u8]) -> Option<CoreStatus> {
    let line = std::str::from_utf8(data).ok()?;
    let line = line.trim();
    let json_str = line.strip_prefix("YCAR_STATUS:")?;
    serde_json::from_str(json_str).ok()
}

fn kill_child(child_opt: &Mutex<Option<CommandChild>>) {
    if let Ok(mut guard) = child_opt.lock() {
        if let Some(child) = guard.take() {
            let _ = child.kill();
        }
        *guard = None;
    }
}

fn sidecar_path(name: &str) -> String {
    let dir = std::env::current_exe()
        .unwrap()
        .parent()
        .unwrap()
        .to_path_buf();

    #[cfg(target_os = "macos")]
    let triple = if cfg!(target_arch = "aarch64") {
        "aarch64-apple-darwin"
    } else {
        "x86_64-apple-darwin"
    };
    #[cfg(target_os = "linux")]
    let triple = "x86_64-unknown-linux-gnu";
    #[cfg(target_os = "windows")]
    let triple = "x86_64-pc-windows-msvc";

    let bundled_path = dir.join(name);
    if bundled_path.exists() {
        return bundled_path.to_str().unwrap_or("").to_string();
    }

    let dev_path = dir.join(format!("{}-{}", name, triple));
    if dev_path.exists() {
        return dev_path.to_str().unwrap_or("").to_string();
    }

    bundled_path.to_str().unwrap_or("").to_string()
}

#[cfg(not(target_os = "macos"))]
fn spawn_stdout_reader(
    mut rx: tauri::async_runtime::Receiver<CommandEvent>,
    status: Arc<Mutex<Option<CoreStatus>>>,
) {
    std::thread::spawn(move || {
        let rt = tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .build()
            .expect("tokio runtime");
        rt.block_on(async move {
            while let Some(event) = rx.recv().await {
                match event {
                    CommandEvent::Stdout(data) => {
                        if let Some(s) = parse_status_line(&data) {
                            if let Ok(mut guard) = status.lock() {
                                *guard = Some(s);
                            }
                        }
                    }
                    CommandEvent::Terminated(_) => {
                        if let Ok(mut guard) = status.lock() {
                            *guard = None;
                        }
                        break;
                    }
                    _ => {}
                }
            }
        });
    });
}

fn spawn_logfile_reader(logfile: String, status: Arc<Mutex<Option<CoreStatus>>>) {
    std::thread::spawn(move || {
        let mut pos: u64 = 0;
        loop {
            if let Ok(mut file) = File::open(&logfile) {
                let _ = file.seek(SeekFrom::Start(pos));
                let reader = BufReader::new(file);
                for line_result in reader.lines() {
                    if let Ok(line) = line_result {
                        let line_len = (line.len() + 1) as u64;
                        if let Some(s) = parse_status_line(line.as_bytes()) {
                            if let Ok(mut guard) = status.lock() {
                                *guard = Some(s);
                            }
                        }
                        pos += line_len;
                    }
                }
            }
            std::thread::sleep(std::time::Duration::from_secs(2));
        }
    });
}

fn start_signaling_server(
    app_handle: &AppHandle,
    child_ref: Arc<Mutex<Option<CommandChild>>>,
) -> Result<(), String> {
    let (mut rx, child) = app_handle
        .shell()
        .sidecar("signaling-server")
        .map_err(|e| format!("signaling sidecar: {}", e))?
        .args(["-port", "9090"])
        .spawn()
        .map_err(|e| format!("signaling spawn: {}", e))?;

    if let Ok(mut guard) = child_ref.lock() {
        *guard = Some(child);
    }

    std::thread::spawn(move || {
        let rt = tokio::runtime::Builder::new_current_thread()
            .enable_all().build().unwrap();
        rt.block_on(async { while rx.recv().await.is_some() {} });
    });

    Ok(())
}

#[cfg(not(target_os = "macos"))]
fn start_core_direct(
    app_handle: &AppHandle,
    room: &str,
    pass: &str,
    addr: &str,
    status_ref: Arc<Mutex<Option<CoreStatus>>>,
    child_ref: Arc<Mutex<Option<CommandChild>>>,
) -> Result<String, String> {
    let (rx, child) = app_handle
        .shell()
        .sidecar("ycair-core")
        .map_err(|e| format!("sidecar: {}", e))?
        .args([room, pass, addr])
        .spawn()
        .map_err(|e| format!("core spawn: {}", e))?;

    if let Ok(mut guard) = child_ref.lock() {
        *guard = Some(child);
    }

    spawn_stdout_reader(rx, status_ref);
    Ok("Go core started".into())
}

#[cfg(target_os = "macos")]
fn start_core_macos(
    room: &str,
    pass: &str,
    addr: &str,
    status_ref: Arc<Mutex<Option<CoreStatus>>>,
) -> Result<String, String> {
    let core = sidecar_path("ycair-core");
    let safe_room = room.replace('/', "_");
    let logfile = format!("/tmp/ycair-core-{}.log", safe_room);

    let script = format!(
        "do shell script \"nohup '{}' '{}' '{}' '{}' > '{}' 2>&1 &\" with administrator privileges",
        core, room, pass, addr, logfile,
    );

    let output = StdCommand::new("osascript")
        .arg("-e")
        .arg(&script)
        .output()
        .map_err(|e| format!("osascript failed: {}", e))?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(format!("Admin authorization cancelled or failed: {}", stderr.trim()));
    }

    // Wait for initial status
    for _ in 0..30 {
        if let Ok(file) = File::open(&logfile) {
            for line_result in BufReader::new(file).lines() {
                if let Ok(line) = line_result {
                    if let Some(s) = parse_status_line(line.as_bytes()) {
                        if let Ok(mut guard) = status_ref.lock() {
                            *guard = Some(s);
                        }
                        spawn_logfile_reader(logfile, status_ref);
                        return Ok("Go core started with admin privileges".into());
                    }
                }
            }
        }
        std::thread::sleep(std::time::Duration::from_millis(200));
    }

    Err("Go core did not produce status output (TUN may have failed)".into())
}

#[tauri::command]
async fn start_connection(
    app_handle: AppHandle,
    state: State<'_, AppState>,
    mode: String,
    room: String,
    pass: String,
    signaling_addr: Option<String>,
) -> Result<String, String> {
    let status_ref = state.status.clone();
    let core_child = state.core_child.clone();
    let signal_child = state.signal_child.clone();

    kill_child(&core_child);
    kill_child(&signal_child);
    if let Ok(mut guard) = status_ref.lock() {
        *guard = None;
    }

    let addr = match mode.as_str() {
        "host" => {
            start_signaling_server(&app_handle, signal_child)?;
            std::thread::sleep(std::time::Duration::from_secs(1));
            "localhost:9090".to_string()
        }
        "join" => signaling_addr.unwrap_or_else(|| "localhost:9090".into()),
        _ => return Err("mode must be 'host' or 'join'".into()),
    };

    #[cfg(target_os = "macos")]
    {
        start_core_macos(&room, &pass, &addr, status_ref)
    }

    #[cfg(not(target_os = "macos"))]
    {
        start_core_direct(&app_handle, &room, &pass, &addr, status_ref, core_child)
    }
}

#[tauri::command]
async fn stop_connection(state: State<'_, AppState>) -> Result<String, String> {
    kill_child(&state.core_child);
    kill_child(&state.signal_child);
    if let Ok(mut guard) = state.status.lock() {
        *guard = None;
    }
    #[cfg(target_os = "macos")]
    {
        let _ = StdCommand::new("pkill").arg("-f").arg("ycair-core").output();
    }
    Ok("Disconnected".into())
}

#[tauri::command]
async fn get_status(state: State<'_, AppState>) -> Result<Option<CoreStatus>, String> {
    match state.status.lock() {
        Ok(guard) => Ok(guard.clone()),
        Err(_) => Err("Failed to read status".into()),
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(AppState {
            status: Arc::new(Mutex::new(None)),
            core_child: Arc::new(Mutex::new(None)),
            signal_child: Arc::new(Mutex::new(None)),
        })
        .invoke_handler(tauri::generate_handler![start_connection, stop_connection, get_status])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
