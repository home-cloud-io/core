import * as React from "react";
import PropTypes from "prop-types";

function Title({ text }) {
  return (
    <>
      <h2 className="title">{text}</h2>
      <div className="divider"></div>
    </>
  );
}

export default Title;
