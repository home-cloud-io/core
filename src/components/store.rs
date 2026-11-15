use crate::api::platform::daemon::v1::{ComputeStats, DriveStats, MemoryStats};
use crate::api::{build_client, platform::server::v1::*};
use crate::components::accordion::*;
use crate::components::button::*;
use crate::components::card::*;
use crate::components::progress::*;
use crate::components::tooltip::*;
use dioxus::logger::tracing;
use dioxus::{logger, prelude::*};
use dioxus_icons::lucide::{Ban, ChevronRight, CircleCheck, CircleX, Info};
use dioxus_primitives::ContentSide;

#[component]
pub fn Store() -> Element {
    let mut response = use_resource(|| async move {
        let mut client = build_client();
        let request = tonic::Request::new(GetAppsInStoreRequest {});
        client
            .get_apps_in_store(request)
            .await
            .unwrap()
            .into_inner()
    });

    rsx! {
        Card { class: "rounded-md light:shadow-lg dark:border dark:border-white self-center m-5 w-8/10",
            CardContent { class: "rounded-md flex flex-col align-center",
                span { class: "m-5 font-bold dark:text-white", "App Store" }
                Accordion {
                    class: "w-full",
                    allow_multiple_open: false,
                    horizontal: false,
                    for (i , app) in response.cloned().unwrap_or_default().apps.iter().enumerate() {
                        AccordionItem { class: "w-full", index: i,
                            AccordionTrigger {
                                img {
                                    class: "w-8 m-2 rounded-md",
                                    src: app.clone().icon,
                                }
                                span { class: "font-bold dark:text-white",
                                    {app.clone().annotations.get("displayName").cloned()}
                                }
                            }
                            AccordionContent {
                                div { class: "flex flex-row justify-around",
                                    div {
                                        class: format!(
                                            "rounded-md border border-{} text-{}",
                                            map_status_to_color(
                                                app.clone().annotations.get("status").cloned().unwrap_or_default(),
                                            ),
                                            map_status_to_color(
                                                app.clone().annotations.get("status").cloned().unwrap_or_default(),
                                            ),
                                        ),
                                        {app.clone().annotations.get("status").cloned()}
                                    }
                                    div { class: "rounded-md border border-gray-400 text-gray-400",
                                        {app.clone().app_version}
                                    }
                                    a {
                                        href: app.clone().app_version,
                                        target: "_blank",
                                        rel: "noopener noreferrer",
                                        div { class: "rounded-md border border-blue-300 text-blue-300",
                                            "Website"
                                        }
                                    }
                                }
                                div {
                                    class: "prose dark:prose-invert m-4",
                                    dangerous_inner_html: markdown::to_html(&app.clone().readme),
                                }
                            }
                        }
                    }
                }
                Button {
                    class: "m-5 w-25 border border-blue-500",
                    variant: ButtonVariant::Primary,
                    onclick: move |_| response.restart(),
                    "Reload"
                }
            }
        }
    }
}

fn map_status_to_color(status: String) -> String {
    match status.as_str() {
        "alpha" => "red-400".to_string(),
        "beta" => "orange-300".to_string(),
        "stable" => "green-300".to_string(),
        _ => "white".to_string()
    }
}


fn install_app(app: App) {
    let mut response = use_resource(|| async move {
        let mut client = build_client();
        let request = tonic::Request::new(InstallAppRequest {
            repo: "apps.home-cloud.io".to_string(),
            chart: app.name.clone(),
            release: app.name.clone(),
            version: app.version.clone(),
            values: "".to_string(),
        });
        client
            .install_app(request)
            .await
            .unwrap()
            .into_inner()
    });
}
