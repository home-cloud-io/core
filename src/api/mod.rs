mod client;

pub use client::*;

pub mod platform {
    pub mod daemon {
        pub mod v1 {
            tonic::include_proto!("platform.daemon.v1");
        }
    }
    pub mod server {
        pub mod v1 {
            tonic::include_proto!("platform.server.v1");
        }
    }
}
