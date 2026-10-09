// use tonic_prost_build;

// fn main() {
//     configure()
//         .compile_protos(&["proto/api/platform/server/v1/web.proto"], &["proto"])
//         .unwrap();
// }

// fn main() -> Result<(), Box<dyn std::error::Error>> {
//     // tonic_prost_build::compile_protos("./api/platform/server/v1/web.proto")?;
//     // Ok(())
//     tonic_prost_build::configure()
//         .build_server(false)
//         .compile_protos(
//             &["api/platform/server/v1/web.proto"],
//             &["api"],
//         )?;
//    Ok(())
// }

use std::io;

fn main() -> io::Result<()> {
    tonic_prost_build::configure()
        .build_server(false)
        .build_transport(false)
        .build_client(true)
        .compile_protos(
            &[
                "api/platform/daemon/v1/daemon.proto",
                "api/platform/daemon/v1/system.proto",
                "api/platform/server/v1/web.proto",
            ],
            &["api"],
        )
}
