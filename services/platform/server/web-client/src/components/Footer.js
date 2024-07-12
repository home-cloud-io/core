import * as React from "react";
import { RiCopyrightLine } from "react-icons/ri";
import "../../public/globals.css";

export default function Footer() {
  return (
    <div className="footer">
      <p>
        <RiCopyrightLine className="icon-copyright" />
        {new Date().getFullYear()} {"Home Cloud"}
      </p>
    </div>
  );
}
