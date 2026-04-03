use tauri_plugin_shell::ShellExt; // 引入 Shell 擴展

#[tauri::command]
async fn start_connection(app_handle: tauri::AppHandle, room: String, pass: String) -> Result<String, String> {
    println!("Rust: 嘗試啟動 Go Sidecar... 房間: {}", room);

    let sidecar_command = app_handle
        .shell()
        .sidecar("ycair-core")
        .map_err(|e| format!("Sidecar 初始化失敗: {}", e))?
        .args([&room, &pass]);

    // 啟動並檢查錯誤
    match sidecar_command.spawn() {
        Ok((mut _rx, _child)) => {
            println!("Rust: Go Sidecar 成功啟動！");
            Ok(format!("Go 核心已啟動"))
        }
        Err(e) => {
            let err_msg = format!("Go Sidecar 啟動崩潰: {}", e);
            println!("{}", err_msg);
            Err(err_msg)
        }
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![start_connection]) // 註冊指令
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}