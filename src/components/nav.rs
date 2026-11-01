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
                class: "border-2 border-dashed border-red-700 w-min",
                variant: SidebarVariant::Floating,
                collapsible: collapsible(),
                side: side(),
                SidebarHeader { class: "border-2 border-dashed border-orange-700 w-max" }
                SidebarContent {
                    SidebarMenu { class: "border-2 border-dashed border-purple-700 flex-auto",
                        SidebarMenuItem { class: "rounded-sm m-1 hover:bg-violet-400 active:bg-violet-700",
                            Link { to: Route::Home,
                                SidebarMenuButton { class: "p-4",
                                    House {}
                                    span { "Home" }
                                }
                            }
                        }
                        SidebarMenuItem { class: "rounded-sm m-1 hover:bg-violet-400 active:bg-violet-700",
                            Link { to: Route::Store,
                                SidebarMenuButton { class: "p-4",
                                    Grid2x2 {}
                                    span { "App Store" }
                                }
                            }
                        }
                        SidebarMenuItem { class: "rounded-sm m-1 hover:bg-violet-400 active:bg-violet-700",
                            Link { to: Route::Settings,
                                SidebarMenuButton { class: "p-4",
                                    Settings {}
                                    span { "Settings" }
                                }
                            }
                        }
                        SidebarMenuItem { class: "rounded-sm m-1 hover:bg-violet-400 active:bg-violet-700",
                            Link { to: Route::About,
                                SidebarMenuButton { class: "p-4",
                                    Info {}
                                    span { "About" }
                                }
                            }
                        }
                    }
                }
                SidebarFooter { class: "border-2 border-dashed border-blue-700 w-max" }
                SidebarRail {}
            }
            SidebarInset {
                header { class: "border-2 border-dashed border-blue-700 flex min-h-fit items-center gap-0.5",
                    SidebarTrigger {}
                    Separator { height: "1rem", horizontal: false }
                    img { class: "scale-200", src: FAVICON }
                    div { class: "flex font-bold", "Home Cloud" }
                }
                Outlet::<Route> {}
            }
        }
    }
}
