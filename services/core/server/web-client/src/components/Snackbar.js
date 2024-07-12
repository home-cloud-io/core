import React from "react";
import { MdCheckCircleOutline, MdErrorOutline } from "react-icons/md";

const Snackbar = ({ message, type }) => {
  const icon =
    type === "success" ? <MdCheckCircleOutline /> : <MdErrorOutline />;

  return (
    <div className={`snackbar snackbar-${type}`}>
      <span className="icon-snackbar">{icon}</span>
      <p>{message}</p>
    </div>
  );
};

export default Snackbar;
