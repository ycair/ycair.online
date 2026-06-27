fn main() {
    tauri_build::build();
    #[cfg(target_os = "windows")]
    embed_resource::compile("app.rc", embed_resource::NONE);
}
