use std::sync::{Arc, Mutex};
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, State};
use tauri_plugin_shell::ShellExt;
use tauri_plugin_shell::process::CommandChild;
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
    sidecar_child: Arc<Mutex<Option<CommandChild>>>,
}

fn parse_status_line(data: &[u8]) -> Option<CoreStatus> {
    let line = std::str::from_utf8(data).ok()?;
    let line = line.trim();
    let json_str = line.strip_prefix("YCAR_STATUS:")?;
    serde_json::from_str(json_str).ok()
}

fn spawn_status_reader(
    mut rx: tauri::async_runtime::Receiver<CommandEvent>,
    status: Arc<Mutex<Option<CoreStatus>>>,
    child_ref: Arc<Mutex<Option<CommandChild>>>,
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
                        if let Ok(mut guard) = child_ref.lock() {
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

#[tauri::command]
async fn start_connection(
    app_handle: AppHandle,
    state: State<'_, AppState>,
    room: String,
    pass: String,
) -> Result<String, String> {
    let status = state.status.clone();
    let child_ref = state.sidecar_child.clone();

    if let Ok(mut guard) = child_ref.lock() {
        let existing: Option<CommandChild> = guard.take();
        if let Some(existing) = existing {
            let _ = existing.kill();
        }
        *guard = None;
    }

    let sidecar_command = app_handle
        .shell()
        .sidecar("ycair-core")
        .map_err(|e| format!("Sidecar init failed: {}", e))?
        .args([&room, &pass]);

    let (rx, child) = sidecar_command
        .spawn()
        .map_err(|e| format!("Go sidecar spawn failed: {}", e))?;

    if let Ok(mut guard) = child_ref.lock() {
        *guard = Some(child);
    }

    spawn_status_reader(rx, status, child_ref);

    Ok("Go core started".into())
}

#[tauri::command]
async fn stop_connection(state: State<'_, AppState>) -> Result<String, String> {
    if let Ok(mut guard) = state.sidecar_child.lock() {
        let existing: Option<CommandChild> = guard.take();
        if let Some(existing) = existing {
            let _ = existing.kill();
        }
        *guard = None;
    }
    if let Ok(mut guard) = state.status.lock() {
        *guard = None;
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
            sidecar_child: Arc::new(Mutex::new(None)),
        })
        .invoke_handler(tauri::generate_handler![start_connection, stop_connection, get_status])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
