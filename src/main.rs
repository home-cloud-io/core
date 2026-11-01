mod components;
mod api;

use crate::components::{SideMenu, Home, Store, Settings, About};
use dioxus::prelude::*;

const FAVICON: Asset = asset!("/assets/favicon.ico");
static CSS: Asset = asset!("/assets/tailwind.css");

fn main() {
    dioxus::launch(App);
}

#[derive(Routable, Clone, PartialEq)]
enum Route {
    #[layout(SideMenu)]
    #[route("/")]
    Home,
    #[route("/store")]
    Store,
    #[route("/settings")]
    Settings,
    #[route("/about")]
    About,
}

#[component]
fn App() -> Element {
    rsx! {
        document::Link { rel: "icon", href: FAVICON }
        document::Stylesheet { href: CSS }
        document::Stylesheet { href: asset!("/assets/dx-components-theme.css") }
        Router::<Route> {}
    }
}

#[component]
fn Title() -> Element {
    rsx! {
        div { id: "title",
            h1 { "Home Cloud" }
        }
    }
}
