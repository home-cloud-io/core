import * as React from "react";
import Title from "../components/Title";

export default function ServiceRegistryPage() {
  return (
    <div className="servreg-container">
      <div className="card servreg-servinv">
        <Title text="Service Inventory" />
        <div className="servreg-servinv-contents">
          <div className="servreg-health">
            <h3>Healthy:</h3>{" "}
            <h4 className="servreg-counter servreg-counter-healthy">1000</h4>
          </div>
          <div className="servreg-health">
            <h3>Unhealthy:</h3>{" "}
            <h4 className="servreg-counter servreg-counter-unhealthy">25</h4>
          </div>
        </div>
      </div>

      <div className="card">
        <Title text="Service Registry" />
      </div>
    </div>
  );
}
