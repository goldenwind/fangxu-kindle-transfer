use tauri::{Manager, Url, WebviewUrl, WebviewWindowBuilder};

const FEEDBACK_URL: &str = "https://api.ip21.cn/products/11/feedback";
const FEEDBACK_WINDOW: &str = "product-feedback";

fn feedback_url(app_version: &str, os: &str, os_version: &str, arch: &str) -> Url {
    let mut url = Url::parse(FEEDBACK_URL).expect("valid product feedback URL");
    // Use the URL form encoder, then move the parameters into the fragment.
    // The feedback page reads and clears it before authentication/submission.
    {
        let mut params = url.query_pairs_mut();
        for (name, value, limit) in [
            ("app_version", app_version, 64),
            ("os", os, 32),
            ("os_version", os_version, 128),
            ("arch", arch, 32),
        ] {
            let value: String = value.trim().chars().take(limit).collect();
            if !value.is_empty() {
                params.append_pair(name, &value);
            }
        }
    }
    let fragment = url.query().unwrap_or_default().to_owned();
    url.set_query(None);
    url.set_fragment(Some(&fragment));
    url
}

#[tauri::command]
pub async fn open_product_feedback(
    app: tauri::AppHandle,
    language: Option<String>,
) -> Result<(), String> {
    // Native window creation must run outside the command/UI thread on Windows.
    // Keep system detection independent of the transfer service as well.
    tauri::async_runtime::spawn_blocking(move || {
        let title = if language.as_deref() == Some("en") {
            "Product feedback · Fangxu Kindle Transfer"
        } else {
            "产品反馈 · 方序传书"
        };
        if let Some(window) = app.get_webview_window(FEEDBACK_WINDOW) {
            // Reuse the page without navigating, preserving the user's draft/login.
            window.set_title(title)?;
            window.unminimize()?;
            window.show()?;
            return window.set_focus();
        }
        let info = os_info::get();
        let os_version = match info.version() {
            os_info::Version::Unknown => String::new(),
            version => version.to_string(),
        };
        let url = feedback_url(
            &app.package_info().version.to_string(),
            std::env::consts::OS,
            &os_version,
            info.architecture().unwrap_or(std::env::consts::ARCH),
        );
        // Load the remote page as its own webview, so its login, storage and file
        // input work normally. Only the main window has native capabilities.
        WebviewWindowBuilder::new(&app, FEEDBACK_WINDOW, WebviewUrl::External(url))
            .title(title)
            .inner_size(560.0, 835.0)
            .min_inner_size(560.0, 580.0)
            .center()
            .disable_drag_drop_handler()
            .build()
            .map(|_| ())
    })
    .await
    .map_err(|error| format!("无法打开产品反馈: {error}"))?
    .map_err(|error| format!("无法打开产品反馈: {error}"))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashMap;

    fn fragment_fields(url: &Url) -> HashMap<String, String> {
        let mut decoded = Url::parse(FEEDBACK_URL).unwrap();
        decoded.set_query(url.fragment());
        decoded.query_pairs().into_owned().collect()
    }

    #[test]
    fn environment_is_encoded_in_the_product_fragment() {
        for (os, version, arch) in [
            ("macos", "15.6.1", "aarch64"),
            ("windows", "Windows 11 24H2 & build=26100+#中文", "x86_64"),
            ("linux", "24.04 LTS", "x86_64"),
        ] {
            let url = feedback_url("0.2.0+test", os, version, arch);
            assert_eq!(url.origin().ascii_serialization(), "https://api.ip21.cn");
            assert_eq!(url.path(), "/products/11/feedback");
            assert_eq!(url.query(), None);
            let fields = fragment_fields(&url);
            assert_eq!(fields.len(), 4);
            assert_eq!(fields["app_version"], "0.2.0+test");
            assert_eq!(fields["os"], os);
            assert_eq!(fields["os_version"], version);
            assert_eq!(fields["arch"], arch);
        }
    }

    #[test]
    fn missing_system_version_does_not_block_feedback() {
        let fields = fragment_fields(&feedback_url("0.2.0", "macos", "  ", "aarch64"));
        assert_eq!(fields.len(), 3);
        assert!(!fields.contains_key("os_version"));
        assert_eq!(fields["app_version"], "0.2.0");
    }

    #[test]
    fn environment_respects_server_character_limits() {
        let fields = fragment_fields(&feedback_url(
            &format!("  {}  ", "版".repeat(65)),
            &"o".repeat(33),
            &"系".repeat(129),
            &"a".repeat(33),
        ));
        assert_eq!(fields["app_version"].chars().count(), 64);
        assert_eq!(fields["os"].chars().count(), 32);
        assert_eq!(fields["os_version"].chars().count(), 128);
        assert_eq!(fields["arch"].chars().count(), 32);
    }
}
