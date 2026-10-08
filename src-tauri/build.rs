fn main() {
    println!(
        "cargo:rustc-env=KINDLE_TARGET={}",
        std::env::var("TARGET").unwrap()
    );
    tauri_build::build()
}
