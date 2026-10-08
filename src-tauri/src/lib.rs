mod feedback;
mod service;
mod updates;

use tauri::Manager;

pub fn run() {
    let app = tauri::Builder::default()
        .plugin(tauri_plugin_single_instance::init(|app, _, _| {
            if let Some(window) = app.get_webview_window("main") {
                let _ = window.unminimize();
                let _ = window.show();
                let _ = window.set_focus();
            }
        }))
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_clipboard_manager::init())
        .manage(service::ServiceController::default())
        .manage(updates::UpdateState::default())
        .setup(|app| {
            // Start the worker before the WebView makes its first request.
            if let Err(error) = app
                .state::<service::ServiceController>()
                .initialize(app.handle())
            {
                eprintln!("{error}"); // Keep the UI available to report/retry startup failures.
            }
            Ok(())
        })
        .on_window_event(|window, event| {
            if window.label() == "main" {
                if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                    api.prevent_close();
                    window.app_handle().exit(0);
                }
            }
        })
        .invoke_handler(tauri::generate_handler![
            service::desktop_request,
            feedback::open_product_feedback,
            updates::check_product_update,
            updates::open_product_update
        ])
        .build(tauri::generate_context!())
        .expect("failed to run Fangxu Kindle Transfer");
    app.run(|app, event| {
        if matches!(event, tauri::RunEvent::Exit) {
            app.state::<service::ServiceController>().shutdown();
        }
    });
}
