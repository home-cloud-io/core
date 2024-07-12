import React, { useState, useRef, useEffect } from "react";
import { NavLink } from "react-router-dom";
import { IoMdNotificationsOutline } from "react-icons/io";
import {
  MdAutoGraph,
  MdOutlineSettings,
  MdKeyboardCommandKey,
} from "react-icons/md";
import { BiNetworkChart } from "react-icons/bi";
import { FaKeycdn } from "react-icons/fa";
import { RiMenuUnfoldLine, RiMenuFoldLine } from "react-icons/ri";
import "../../public/globals.css";

export default function Header() {
  // ----- State for toggling Menu and Notify
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [notificationCount, setNotificationCount] = useState(5); // <-- init to 5 for Testing
  const menuRef = useRef(null);

  // ----- Reset Counter
  const handleNotificationClick = () => {
    setNotificationCount(0);
  };

  // ----- Open/Close Menu
  const handleMenuClick = () => {
    setIsMenuOpen(!isMenuOpen);
  };

  // ----- Menu starts closed...
  const closeMenu = () => {
    setIsMenuOpen(false);
  };

  // ----- Close Menu if click outside Menu
  useEffect(() => {
    function handleClickOutside(event) {
      if (menuRef.current && !menuRef.current.contains(event.target)) {
        closeMenu();
      }
    }

    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [menuRef]);

  return (
    <div className="header">
      <div className="header-left">
        {/* ----- Menu Icon ----- */}
        <div className="menu-icon" onClick={handleMenuClick} ref={menuRef}>
          {isMenuOpen ? (
            <RiMenuFoldLine className="icon-menu icon-menu-open" />
          ) : (
            <RiMenuUnfoldLine className="icon-menu" />
          )}
          {/* ----- Menu Content ----- */}
          {isMenuOpen && (
            <div className="menu">
              <div className="menu-row">
                <div className="menu-left">
                  <NavLink to="/" onClick={closeMenu}>
                    <MdAutoGraph className="menuIcons" />
                    Metrics{" "}
                  </NavLink>
                </div>
                <div className="menu-right">
                  <MdKeyboardCommandKey className="icon-command" />M
                </div>
              </div>
              <div className="menu-row">
                <div className="menu-left">
                  <NavLink to="/service-registry" onClick={closeMenu}>
                    <BiNetworkChart className="menuIcons" />
                    Services
                  </NavLink>
                </div>
                <div className="menu-right">
                  <MdKeyboardCommandKey className="icon-command" />X
                </div>
              </div>

              <div className="menu-row">
                <div className="menu-left">
                  <NavLink to="/key-values" onClick={closeMenu}>
                    <FaKeycdn className="menuIcons" />
                    Key/Values
                  </NavLink>
                </div>
                <div className="menu-right">
                  <MdKeyboardCommandKey className="icon-command" />K
                </div>
              </div>
              <div className="divider-menu" />
              {/* ----- TODO: change this class to menu-left when active */}
              <div className="menu-bottom">
                <MdOutlineSettings className="menuIcons" />
                Settings
              </div>
            </div>
          )}
          {/* ----- "Logo"/Text ----- */}
          <span className="logo-text">{"Home Cloud"}</span>
        </div>
      </div>
      <div className="header-right">
        {/* ----- Notifications ----- */}
        <div className="badge-notify" onClick={handleNotificationClick}>
          <IoMdNotificationsOutline className="icon-notify" />
          {/* ----- Notifications Badge ----- */}
          {notificationCount > 0 && (
            <div className="badge-overlay">{notificationCount}</div>
          )}
        </div>
      </div>
    </div>
  );
}
