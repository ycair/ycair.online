use std::sync::{Arc, Mutex};
use std::process::Command as StdCommand;
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, State};
use tauri_plugin_shell::ShellExt;
use tauri_plugin_shell::process::CommandChild;
#[cfg(not(target_os = "macos"))]
use tauri_plugin_shell::process::CommandEvent;

const DEFAULT_SIGNALING: &str = "signal.ycair.space";

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StatusPeer { pub id: String, pub ip: String }

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CoreStatus {
    #[serde(rename = "type")] pub msg_type: String,
    pub assigned_ip: String, pub peer_id: String,
    pub peers: Vec<StatusPeer>, pub tun: String,
    pub connected: bool, pub public_ip: String,
}

struct AppState {
    status: Arc<Mutex<Option<CoreStatus>>>,
    core_child: Arc<Mutex<Option<CommandChild>>>,
}

fn cleanup_core_processes() {
    #[cfg(target_os = "macos")]
    {
        let _ = StdCommand::new("sudo").args(["pkill", "ycair-core"]).output();
        std::thread::sleep(std::time::Duration::from_millis(500));
        let _ = StdCommand::new("sudo").args(["pkill", "-9", "ycair-core"]).output();
        let _ = StdCommand::new("sudo").args(["sh", "-c", "ifconfig -l | tr ' ' '\n' | grep utun | while read iface; do ifconfig $iface 2>/dev/null | grep -q '10.99.0' && ifconfig $iface destroy 2>/dev/null; done"]).output();
    }
    #[cfg(target_os = "windows")]
    {
        let _ = StdCommand::new("taskkill").args(["/F", "/IM", "ycair-core-x86_64-pc-windows-msvc.exe"]).output();
    }
}

impl Drop for AppState {
    fn drop(&mut self) {
        kill_child(&self.core_child);
        cleanup_core_processes();
    }
}

fn parse_status_line(data: &[u8]) -> Option<CoreStatus> {
    serde_json::from_str(std::str::from_utf8(data).ok()?.trim().strip_prefix("YCAR_STATUS:")?).ok()
}

fn kill_child(o: &Mutex<Option<CommandChild>>) {
    if let Ok(mut g) = o.lock() { if let Some(c) = g.take() { let _ = c.kill(); } *g = None; }
}

#[cfg(target_os = "macos")]
fn sidecar_path(name: &str) -> String {
    let dir = std::env::current_exe().unwrap().parent().unwrap().to_path_buf();
    if dir.join(name).exists() { return dir.join(name).to_str().unwrap_or("").into(); }
    let t = if cfg!(target_arch = "aarch64") { "aarch64-apple-darwin" } else { "x86_64-apple-darwin" };
    dir.join(format!("{}-{}", name, t)).to_str().unwrap_or("").into()
}

#[cfg(not(target_os = "macos"))]
fn spawn_stdout_reader(mut rx: tauri::async_runtime::Receiver<CommandEvent>, s: Arc<Mutex<Option<CoreStatus>>>) {
    std::thread::spawn(move || {
        let rt = tokio::runtime::Builder::new_current_thread().enable_all().build().unwrap();
        let mut buf = Vec::new();
        rt.block_on(async move {
            while let Some(event) = rx.recv().await {
                match event {
                    CommandEvent::Stdout(data) => {
                        buf.extend_from_slice(&data);
                        while let Some(pos) = buf.iter().position(|&b| b == b'\n') {
                            let line = buf[..pos].to_vec();
                            buf.drain(..=pos);
                            if let Some(st) = parse_status_line(&line) {
                                if let Ok(mut g) = s.lock() { *g = Some(st); }
                            }
                        }
                    }
                    CommandEvent::Terminated(_) => {
                        if let Ok(mut g) = s.lock() { *g = None; }
                        return;
                    }
                    _ => {}
                }
            }
        });
    });
}

#[cfg(target_os = "macos")]
fn start_core(room: &str, pass: &str, addr: &str, s: Arc<Mutex<Option<CoreStatus>>>) -> Result<String, String> {
    let core = sidecar_path("ycair-core"); let safe = room.replace('/', "_");
    let lf = format!("/tmp/ycair-core-{}.log", safe);
    let _ = std::fs::remove_file(&lf);
    let (r,p,a,lf2,sc) = (room.to_string(), pass.to_string(), addr.to_string(), lf.clone(), s.clone());
    std::thread::spawn(move || { let _ = StdCommand::new("osascript").arg("-e").arg(&format!("do shell script \"exec '{}' '{}' '{}' '{}' > '{}' 2>> '{}'\" with administrator privileges", core, r, p, a, lf2, lf2)).output(); if let Ok(mut g) = sc.lock() { *g = None; } });
    for _ in 0..50 { if let Ok(c) = std::fs::read_to_string(&lf) { for line in c.lines() { if let Some(st) = parse_status_line(line.as_bytes()) { let ip = st.assigned_ip.clone(); if let Ok(mut g) = s.lock() { *g = Some(st); } return Ok(ip); } } } std::thread::sleep(std::time::Duration::from_millis(200)); }
    Err("ycair-core did not start. Check your admin password.".into())
}

#[cfg(not(target_os = "macos"))]
fn start_core(app_handle: &AppHandle, room: &str, pass: &str, addr: &str, s: Arc<Mutex<Option<CoreStatus>>>, c: Arc<Mutex<Option<CommandChild>>>) -> Result<String, String> {
    let (rx, child) = app_handle.shell().sidecar("ycair-core").map_err(|e| format!("sidecar: {}", e))?.args([room,pass,addr]).spawn().map_err(|e| format!("spawn: {}", e))?;
    if let Ok(mut g) = c.lock() { *g = Some(child); } spawn_stdout_reader(rx, s); Ok("started".into())
}

#[tauri::command]
async fn start_connection(app_handle: AppHandle, state: State<'_, AppState>, room: String, pass: String) -> Result<String, String> {
    let sr = state.status.clone(); let cc = state.core_child.clone();
    kill_child(&cc); if let Ok(mut g) = sr.lock() { *g = None; }
    #[cfg(target_os = "macos")] { start_core(&room, &pass, DEFAULT_SIGNALING, sr) }
    #[cfg(not(target_os = "macos"))] { start_core(&app_handle, &room, &pass, DEFAULT_SIGNALING, sr, cc) }
}

#[tauri::command]
async fn stop_connection(state: State<'_, AppState>) -> Result<(), String> {
    kill_child(&state.core_child);
    cleanup_core_processes();
    if let Ok(mut g) = state.status.lock() { *g = None; } Ok(())
}

#[tauri::command]
async fn get_status(state: State<'_, AppState>) -> Result<Option<CoreStatus>, String> {
    state.status.lock().map(|g| g.clone()).map_err(|_| "locked".into())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default().plugin(tauri_plugin_shell::init())
        .manage(AppState { status: Arc::new(Mutex::new(None)), core_child: Arc::new(Mutex::new(None)) })
        .invoke_handler(tauri::generate_handler![start_connection, stop_connection, get_status])
        .run(tauri::generate_context!()).expect("error");
}
