import * as React from "react";
import "../../public/globals.css";
import Title from "./Title";
import { PiSealWarningLight } from "react-icons/pi";

// Generate Order Data
function createData(id, name, status, state) {
  return { id, name, status, state };
}

const rows = [
  createData(0, "CST - US - South 1", "Online", "Voter"),
  createData(1, "MNT - US - North 2", "Online", "Leader"),
  createData(2, "MNT - US - North 1", "Online", "Voter"),
  createData(3, "PST - US - North 1", "Offline", "Abandoned"),
  createData(4, "PST - US - South 1", "Online", "Voter"),
  createData(0, "CST - US - South 1", "Online", "Voter"),
  createData(1, "MNT - US - North 2", "Online", "Leader"),
  createData(2, "MNT - US - North 1", "Online", "Voter"),
  createData(3, "PST - US - North 1", "Offline", "Abandoned"),
  createData(4, "PST - US - South 1", "Online", "Voter"),
];

export default function ClusterNodes() {
  return (
    <>
      <Title text="Cluster Nodes" />
      <div className="clusterNodes-content">
        <div className="table">
          <div className="table-row table-header">
            <div className="table-icon"></div>
            <div className="table-cell">Name</div>
            <div className="table-cell">Status</div>
            <div className="table-cell align-right">Node State</div>
          </div>
          {rows.map((row) => (
            <div key={row.id} className="table-row">
              <div className="table-icon">
                {row.status === "Offline" && (
                  <PiSealWarningLight className="icon-offline" />
                )}
              </div>
              <div
                className={`table-cell ${
                  row.status === "Offline" ? "offline" : ""
                }`}
              >
                {row.name}
              </div>
              <div
                className={`table-cell ${
                  row.status === "Offline" ? "offline" : ""
                }`}
              >
                {row.status}
              </div>
              <div
                className={`table-cell align-right ${
                  row.status === "Offline" ? "offline" : ""
                }`}
              >
                {row.state}
              </div>
            </div>
          ))}
        </div>
      </div>
    </>
  );
}
