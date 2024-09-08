use std::io::Result;
fn main() -> Result<()> {
    tonic_build::configure().compile(
        &[
            "../proto/v1rusticdaemon/daemon.proto",
            "../proto/types/value.proto",
        ],
        &["../proto"],
    )?;

    Ok(())
}
