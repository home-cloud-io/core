use crate::api::platform::server::v1::web_service_client::WebServiceClient;
use tonic_web_wasm_client::Client;

pub fn build_client() -> WebServiceClient<Client> {
    let base_url = "http://localhost:8000".to_string();

    let wasm_client = Client::new(base_url);

    WebServiceClient::new(wasm_client)
}