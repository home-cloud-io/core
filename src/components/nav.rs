use dioxus::prelude::*;
use crate::components::separator::Separator;
use crate::components::sidebar::{
    Sidebar, SidebarCollapsible, SidebarContent, SidebarFooter,
    SidebarHeader, SidebarInset, SidebarMenu,
    SidebarMenuButton, SidebarMenuItem,
    SidebarProvider, SidebarRail, SidebarSide,
    SidebarTrigger, SidebarVariant,
};
use crate::{Route, FAVICON};
use dioxus_icons::lucide::{Grid2x2, House, Info, Settings};

#[component]
pub fn SideMenu() -> Element {
    let side = use_signal(|| SidebarSide::Left);
    let collapsible = use_signal(|| SidebarCollapsible::Offcanvas);

    rsx! {
        SidebarProvider {
            Sidebar {
                class: "w-min",
                variant: SidebarVariant::Sidebar,
                collapsible: collapsible(),
                side: side(),
                SidebarHeader { class: "w-max" }
                SidebarContent {
                    SidebarMenu { class: "flex-auto",
                        SidebarMenuItem { class: "rounded-sm m-1",
                            Link { to: Route::Home,
                                SidebarMenuButton { class: "p-4",
                                    House {}
                                    span { "Home" }
                                }
                            }
                        }
                        SidebarMenuItem { class: "rounded-sm m-1",
                            Link { to: Route::Store,
                                SidebarMenuButton { class: "p-4",
                                    Grid2x2 {}
                                    span { "App Store" }
                                }
                            }
                        }
                        SidebarMenuItem { class: "rounded-sm m-1",
                            Link { to: Route::Settings,
                                SidebarMenuButton { class: "p-4",
                                    Settings {}
                                    span { "Settings" }
                                }
                            }
                        }
                        SidebarMenuItem { class: "rounded-sm m-1",
                            Link { to: Route::About,
                                SidebarMenuButton { class: "p-4",
                                    Info {}
                                    span { "About" }
                                }
                            }
                        }
                    }
                }
                SidebarFooter { class: "w-max" }
                SidebarRail {}
            }
            SidebarInset {
                header { class: "flex min-h-fit items-center gap-0.5",
                    SidebarTrigger {}
                    Separator { height: "1rem", horizontal: false }
                    img { class: "scale-80", src: FAVICON }
                    div { class: "flex font-bold dark:text-white", "Home Cloud" }
                }
                Outlet::<Route> {}
            }
        }
    }
}
