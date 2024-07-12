import * as React from "react";
import { createConnectTransport } from "@connectrpc/connect-web";
import { createCallbackClient } from "@connectrpc/connect";
import { KeyValueService } from "../grpc/registry/key_value/v1/service_connect";

import Chart from "../components/Chart";
import Button from "../components/Button";
import ClusterDetails from "../components/ClusterDetails";
import ClusterNodes from "../components/ClusterNodes";

export default function MetricsPage() {
  // TODO -> make a grpc provider
  const transport = createConnectTransport({
    baseUrl: "http://localhost:2221",
  });

  const key_value_client = createCallbackClient(KeyValueService, transport);

  const handleClick = () => {
    key_value_client.set(
      { key: "andrew", value: "needs to take a break" },
      (err, res) => {
        if (!err) {
          console.log(res);
        }
      }
    );
  };

  return (
    <div className="metrics-container">
      <div className="metrics-content">
        <div className="metrics-top">
          <div className="card metrics-topleft">
            <Chart />
          </div>

          <div className="metrics-topright card">
            <ClusterDetails />
            <div
              style={{
                display: "flex",
                justifyContent: "center",
                paddingTop: "5px",
              }}
            >
              <Button type="solid" onclick={handleClick} text={"Click Me"} />
            </div>
          </div>
        </div>
        <div className="metrics-bottom">
          <div className="card">
            <ClusterNodes />
          </div>
        </div>
      </div>
    </div>
  );
}
