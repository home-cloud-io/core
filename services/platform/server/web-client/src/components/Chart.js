import * as React from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Label,
  ResponsiveContainer,
} from "recharts";
import Title from "./Title";

function createData(time, amount) {
  return { time, amount };
}

const data = [
  createData("00:00", 9888),
  createData("03:00", 7884),
  createData("06:00", 2000),
  createData("09:00", 800),
  createData("12:00", 1500),
  createData("15:00", 2000),
  createData("18:00", 2400),
  createData("21:00", 10000),
  createData("24:00", undefined),
];

export default function Chart() {
  return (
    <>
      <Title text="Request Volume" />
      <ResponsiveContainer>
        <LineChart
          data={data}
          margin={{
            top: 16,
            right: 16,
            bottom: 0,
            left: 24,
          }}
        >
          <XAxis
            dataKey="time"
            stroke="var(--text-gray)"
            style={{ fontSize: "small" }}
          />
          <YAxis stroke="var(--text-gray)" style={{ fontSize: "small" }}>
            <Label
              angle={270}
              position="left"
              style={{
                textAnchor: "middle",
                fill: "var(--text-gray)",
              }}
            >
              Request Count
            </Label>
          </YAxis>
          <Line
            isAnimationActive={false}
            type="monotone"
            dataKey="amount"
            stroke="var(--a)"
            dot={false}
          />
        </LineChart>
      </ResponsiveContainer>
    </>
  );
}
