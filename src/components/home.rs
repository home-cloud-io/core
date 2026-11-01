use crate::api::platform::daemon::v1::{ComputeStats, DriveStats, MemoryStats};
use crate::api::{build_client, platform::server::v1::*};
use crate::components::button::{Button, ButtonVariant};
use crate::components::card::{Card, CardAction, CardContent, CardHeader, CardTitle};
use crate::components::progress::Progress;
use dioxus::prelude::*;
use dioxus_icons::lucide::CircleCheck;

#[component]
pub fn Home() -> Element {
    let mut response = use_resource(|| async move {
        let mut client = build_client();
        let request = tonic::Request::new(GetSystemStatsRequest {});
        client.get_system_stats(request).await.unwrap().into_inner()
    });

    rsx! {
        Card { class: "rounded-md light:shadow-lg dark:border dark:border-white self-center m-5 w-8/10",
            CardHeader { class: "flex items-center justify-between",
                CardTitle { class: "m-5 font-bold dark:text-white", "Status" }
                CardAction { class: "m-5",
                    div { class: "border border-green-300 rounded-md s-15 flex align-center",
                        CircleCheck { class: "m-1 text-green-300" }
                        span { class: "m-1 text-green-300 text-s", "TODO" }
                    }
                }
            }
            CardContent { class: "rounded-md flex flex-col align-center",
                DriveList { stats: response.cloned().unwrap_or_default().stats.unwrap_or_default().drives }
                SystemStats {
                    compute: response
                        .cloned()
                        .unwrap_or_default()
                        .stats
                        .unwrap_or_default()
                        .compute
                        .unwrap_or_default(),
                    memory: response
                        .cloned()
                        .unwrap_or_default()
                        .stats
                        .unwrap_or_default()
                        .memory
                        .unwrap_or_default(),
                }
                Button {
                    class: "m-5 w-25",
                    variant: ButtonVariant::Primary,
                    onclick: move |_| response.restart(),
                    "Reload"
                }
            }
        }
    }
}

#[component]
fn DriveList(stats: Vec<DriveStats>) -> Element {
    rsx! {
        div { class: "flex flex-col",
            span { class: "m-5 font-bold dark:text-white", "Storage" }
            for drive in stats.iter() {
                div { class: "m-5",
                    span { class: "m-1 font-light dark:text-white",
                        {
                            format!(
                                "{} - {}%",
                                drive.mount_point.to_string(),
                                format_percentage(drive.free_bytes as f64, drive.total_bytes as f64),
                            )
                        }
                    }
                    Progress {
                        aria_label: drive.mount_point.to_string(),
                        value: format_percentage(drive.free_bytes as f64, drive.total_bytes as f64),
                    }
                }
            }
        }
    }
}


#[component]
fn SystemStats(compute: ComputeStats, memory: MemoryStats) -> Element {
    rsx! {
        span { class: "m-5 font-bold dark:text-white", "System" }
        div { class: "m-5",
            span { class: "m-1 font-light dark:text-white",
                {format!("cpu - {}%", format_percentage(compute.idle_percent as f64, 1.0))}
            }
            Progress {
                aria_label: "cpu",
                value: format_percentage(compute.idle_percent as f64, 1.0),
            }
        }
        div { class: "m-5",
            span { class: "m-1 font-light dark:text-white",
                {
                    format!(
                        "memory - {}%",
                        format_percentage(
                            (memory.total_bytes - memory.used_bytes) as f64,
                            memory.total_bytes as f64,
                        ),
                    )
                }
            }
            Progress {
                aria_label: "memory",
                value: format_percentage(
                    (memory.total_bytes - memory.used_bytes) as f64,
                    memory.total_bytes as f64,
                ),
            }
        }
    }
}

fn format_percentage(free: f64, total: f64) -> f64 {
    let percent = ((total - free) / total) * 100.0;
    percent.round()
}
