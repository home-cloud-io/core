import * as React from "react";
import "../../public/globals.css";

import Title from "./Title";

export default function ClusterDetails() {
  return (
    <>
      <Title text="Cluster Details" />
      <h4>Nodes</h4>
      <div className="clusterDetails-content">
        <div className="clusterDetails-counter">
          <span>Healthy:</span>{" "}
          <span className="clusterDetails-health">25</span>
        </div>
        <div className="clusterDetails-counter">
          <span>Unhealthy:</span>{" "}
          <span className="clusterDetails-health">1</span>
        </div>
      </div>
      <div className="divider-gray" />
    </>
  );
}
