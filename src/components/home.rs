use crate::api::{build_client, platform::server::v1::*};
use crate::components::button::Button;
use crate::components::card::{Card, CardAction, CardContent, CardHeader, CardTitle};
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
        div { class: "border-2 border-dashed border-yellow-700 flex flex-col align-center bg-gray-100",
            Card { class: "rounded-md bg-white shadow-lg self-center m-5 w-9/10",
                CardHeader { class: "flex items-center justify-between",
                    CardTitle { class: "m-5 font-bold", "Status" }
                    CardAction { class: "m-5",
                        div { class: "border border-green-300 rounded-sm s-15 flex align-center",
                            CircleCheck { class: "text-green-300" }
                            // Separator {}
                            span { class: "text-green-300 text-s", "Online" }
                        }
                    }
                }
                CardContent { class: "border border-red-500 flex flex-col align-center",
                    span { class: "font-bold", "Storage" }
                    span { class: "font-bold", "System" }
                    Button { onclick: move |_| response.restart(), id: "poll", "poll!" }
                    div { id: "stats", class: "text-blue-500",
                        {
                            response
                                .cloned()
                                .unwrap_or_default()
                                .stats
                                .unwrap_or_default()
                                .compute
                                .unwrap_or_default()
                                .user_percent
                                .to_string()
                        }
                    }
                }
            }
        }
    }
}
