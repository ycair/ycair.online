use std::sync::{Arc, Mutex};
use std::process::Command as StdCommand;
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, State};
use tauri_plugin_shell::ShellExt;
use tauri_plugin_shell::process::CommandChild;
#[cfg(not(target_os = "macos"))]
use tauri_plugin_shell::process::CommandEvent;

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
    signal_child: Arc<Mutex<Option<CommandChild>>>,
    tunnel_url: Arc<Mutex<Option<String>>>,
}

impl Drop for AppState {
    fn drop(&mut self) {
        kill_child(&self.core_child); kill_child(&self.signal_child);
        let _ = StdCommand::new("sudo").args(["pkill", "-f", "ycair-core"]).output();
        let _ = StdCommand::new("pkill").args(["-f", "cloudflared.*9090"]).output();
    }
}

fn parse_status_line(data: &[u8]) -> Option<CoreStatus> { let line = std::str::from_utf8(data).ok()?; serde_json::from_str(line.trim().strip_prefix("YCAR_STATUS:")?).ok() }
fn kill_child(o: &Mutex<Option<CommandChild>>) { if let Ok(mut g) = o.lock() { if let Some(c) = g.take() { let _ = c.kill(); } *g = None; } }

fn sidecar_path(name: &str) -> String {
    let dir = std::env::current_exe().unwrap().parent().unwrap().to_path_buf();
    if dir.join(name).exists() { return dir.join(name).to_str().unwrap_or("").into(); }
    let t = if cfg!(target_arch = "aarch64") { "aarch64-apple-darwin" } else { "x86_64-apple-darwin" };
    dir.join(format!("{}-{}", name, t)).to_str().unwrap_or("").into()
}

#[cfg(not(target_os = "macos"))]
fn spawn_stdout_reader(mut rx: tauri::async_runtime::Receiver<CommandEvent>, s: Arc<Mutex<Option<CoreStatus>>>) {
    std::thread::spawn(move || { let rt = tokio::runtime::Builder::new_current_thread().enable_all().build().unwrap(); rt.block_on(async { while let Some(CommandEvent::Stdout(d)) = rx.recv().await { if let Some(st) = parse_status_line(&d) { if let Ok(mut g) = s.lock() { *g = Some(st); } } } }); });
}

fn start_signaling(app_handle: &AppHandle, c: Arc<Mutex<Option<CommandChild>>>) -> Result<(), String> {
    let (mut rx, child) = app_handle.shell().sidecar("signaling-server").map_err(|e| format!("signal: {}", e))?.args(["-port","9090"]).spawn().map_err(|e| format!("spawn: {}", e))?;
    if let Ok(mut g) = c.lock() { *g = Some(child); }
    std::thread::spawn(move || { let rt = tokio::runtime::Builder::new_current_thread().enable_all().build().unwrap(); rt.block_on(async { while rx.recv().await.is_some() {} }); });
    Ok(())
}

fn try_start_tunnel(tunnel_url: Arc<Mutex<Option<String>>>) {
    std::thread::spawn(move || {
        if !StdCommand::new("which").arg("cloudflared").output().map(|o| o.status.success()).unwrap_or(false) { return; }
        if let Ok(mut child) = StdCommand::new("cloudflared").args(["tunnel","--url","http://localhost:9090","--no-autoupdate"]).stdout(std::process::Stdio::piped()).stderr(std::process::Stdio::null()).spawn() {
            use std::io::{BufRead, BufReader};
            if let Some(o) = child.stdout.take() { for line in BufReader::new(o).lines().flatten() { if let Some(s) = line.find("https://") { if let Some(e) = line[s..].find(".trycloudflare.com") { let url = &line[s..s+e+".trycloudflare.com".len()]; if let Ok(mut g) = tunnel_url.lock() { *g = Some(url.strip_prefix("https://").unwrap_or(url).to_string()); } break; } } } }
        }
    });
}

#[cfg(target_os = "macos")]
fn start_core_macos(room: &str, pass: &str, addr: &str, s: Arc<Mutex<Option<CoreStatus>>>) -> Result<String, String> {
    let core = sidecar_path("ycair-core"); let safe = room.replace('/', "_"); let lf = format!("/tmp/ycair-core-{}.log", safe);
    let _ = std::fs::remove_file(&lf);
    let (r,p,a,lf2,sc) = (room.to_string(), pass.to_string(), addr.to_string(), lf.clone(), s.clone());
    std::thread::spawn(move || { let _ = StdCommand::new("osascript").arg("-e").arg(&format!("do shell script \"exec '{}' '{}' '{}' '{}' > '{}' 2>> '{}'\" with administrator privileges", core, r, p, a, lf2, lf2)).output(); if let Ok(mut g) = sc.lock() { *g = None; } });
    for _ in 0..50 { if let Ok(c) = std::fs::read_to_string(&lf) { for line in c.lines() { if let Some(st) = parse_status_line(line.as_bytes()) { let ip = st.assigned_ip.clone(); if let Ok(mut g) = s.lock() { *g = Some(st); } return Ok(ip); } } } std::thread::sleep(std::time::Duration::from_millis(200)); }
    Err("Go core did not start".into())
}

#[cfg(not(target_os = "macos"))]
fn start_core_direct(app_handle: &AppHandle, room: &str, pass: &str, addr: &str, s: Arc<Mutex<Option<CoreStatus>>>, c: Arc<Mutex<Option<CommandChild>>>) -> Result<String, String> {
    let (rx, child) = app_handle.shell().sidecar("ycair-core").map_err(|e| format!("sidecar: {}", e))?.args([room,pass,addr]).spawn().map_err(|e| format!("spawn: {}", e))?;
    if let Ok(mut g) = c.lock() { *g = Some(child); } spawn_stdout_reader(rx, s); Ok("started".into())
}

#[tauri::command]
async fn start_connection(app_handle: AppHandle, state: State<'_, AppState>, mode: String, room: String, pass: String, signaling_addr: Option<String>) -> Result<String, String> {
    let sr = state.status.clone(); let cc = state.core_child.clone(); let sc = state.signal_child.clone();
    kill_child(&cc); kill_child(&sc);
    if let Ok(mut g) = sr.lock() { *g = None; }
    let addr = match mode.as_str() {
        "host" => { start_signaling(&app_handle, sc)?; std::thread::sleep(std::time::Duration::from_secs(1)); try_start_tunnel(state.tunnel_url.clone()); "localhost:9090".into() }
        "join" => signaling_addr.unwrap_or_else(|| "localhost:9090".into()),
        _ => return Err("mode must be 'host' or 'join'".into()),
    };
    #[cfg(target_os = "macos")] { start_core_macos(&room, &pass, &addr, sr) }
    #[cfg(not(target_os = "macos"))] { start_core_direct(&app_handle, &room, &pass, &addr, sr, cc) }
}

#[tauri::command] async fn stop_connection(state: State<'_, AppState>) -> Result<(), String> { kill_child(&state.core_child); kill_child(&state.signal_child); let _ = StdCommand::new("sudo").args(["pkill","-f","ycair-core"]).output(); let _ = StdCommand::new("pkill").args(["-f","cloudflared.*9090"]).output(); if let Ok(mut g) = state.status.lock() { *g = None; } Ok(()) }
#[tauri::command] async fn get_status(state: State<'_, AppState>) -> Result<Option<CoreStatus>, String> { state.status.lock().map(|g| g.clone()).map_err(|_| "locked".into()) }
#[tauri::command] async fn get_tunnel_url(state: State<'_, AppState>) -> Result<Option<String>, String> { state.tunnel_url.lock().map(|g| g.clone()).map_err(|_| "locked".into()) }

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default().plugin(tauri_plugin_shell::init())
        .manage(AppState { status: Arc::new(Mutex::new(None)), core_child: Arc::new(Mutex::new(None)), signal_child: Arc::new(Mutex::new(None)), tunnel_url: Arc::new(Mutex::new(None)) })
        .invoke_handler(tauri::generate_handler![start_connection, stop_connection, get_status, get_tunnel_url])
        .run(tauri::generate_context!()).expect("error");
}
