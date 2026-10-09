mod components;

use crate::{
    components::{button::Button, card::Card, *}, platform::server::v1::web_service_client::WebServiceClient,
};
use dioxus::prelude::*;
use tonic_web_wasm_client::Client;
use platform::server::v1::*;



const FAVICON: Asset = asset!("/assets/favicon.ico");
static CSS: Asset = asset!("/assets/tailwind.css");

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

fn main() {
    dioxus::launch(App);
}

fn build_client() -> WebServiceClient<Client> {
    let base_url = "http://localhost:8000".to_string();

    let wasm_client = Client::new(base_url);
    // wasm_client.with_options(FetchOptions::default().timeout(Duration::from_secs(2)));

    WebServiceClient::new(wasm_client)
}

#[derive(Routable, Clone, PartialEq)]
enum Route {
    #[layout(NavBar)]
    #[route("/")]
    DogView,

    #[route("/favorites")]
    Favorites,
    // #[route("/:..segments")]
    // PageNotFound { segments: Vec<String> },
}

#[component]
fn App() -> Element {
    rsx! {
        document::Link { rel: "icon", href: FAVICON }
        document::Stylesheet { href: CSS }
        Router::<Route> {}
    }
}

#[component]
fn Title() -> Element {
    rsx! {
        div { id: "title",
            h1 { "HotDog! 🌭" }
        }
    }
}

#[derive(serde::Deserialize)]
struct DogApi {
    message: String,
}

#[component]
fn DogView() -> Element {
    let mut img_src = use_resource(|| async move {
        reqwest::get("https://dog.ceo/api/breeds/image/random")
            .await
            .unwrap()
            .json::<DogApi>()
            .await
            .unwrap()
            .message
    });

    let mut response = use_resource(|| async move {
        let mut client = build_client();
        let request = tonic::Request::new(GetSystemStatsRequest{ });
        client
            .get_system_stats(request)
            .await
            .unwrap()
            .into_inner()
    });

    rsx! {
        div { id: "dogview",
            img { src: img_src.cloned().unwrap_or_default() }
        }
        div { id: "buttons",
            Button {
                variant: button::ButtonVariant::Destructive,
                onclick: move |_| img_src.restart(),
                id: "next",
                "next"
            }
            // button {
            //     class: "text-amber-950",
            // }
            Button { onclick: move |_| response.restart(), id: "poll", "poll!" }
        }
        div { id: "stats", class: "text-blue-500",
            Card {
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
