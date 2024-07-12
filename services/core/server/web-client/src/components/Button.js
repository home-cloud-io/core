import React from "react";
import "../../public/globals.css";

const Button = ({ text, type, onClick }) => {
  const btnType = type === "solid" ? "btn btn-solid" : "btn btn-outline";
  return (
    <button className={btnType} onClick={onClick}>
      {text}
    </button>
  );
};

export default Button;
