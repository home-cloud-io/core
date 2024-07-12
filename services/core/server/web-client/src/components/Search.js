import React, { useState, useEffect, useRef } from "react";
import { FaChevronDown, FaChevronUp } from "react-icons/fa";

const Search = ({ placeholder, options, onSelect }) => {
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const dropdownRef = useRef(null);

  useEffect(() => {
    document.addEventListener("click", handleClickOutside);
    return () => {
      document.removeEventListener("click", handleClickOutside);
    };
  }, []);

  const handleClickOutside = (event) => {
    if (
      dropdownRef.current &&
      !dropdownRef.current.contains(event.target) &&
      event.target.className !== "selected-option"
    ) {
      setIsOpen(false);
    }
  };

  const toggleDropdown = (event) => {
    event.stopPropagation();
    setIsOpen(!isOpen);
  };

  const handleOptionClick = (option) => {
    onSelect(option);
    setIsOpen(false);
    setSearchTerm(option.label);
  };

  const handleInputChange = (event) => {
    setSearchTerm(event.target.value);
    setIsOpen(true);
  };

  const filteredOptions = options.filter((option) =>
    option.label.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="search-container">
      <div className="selected-option" onClick={toggleDropdown}>
        <input
          type="text"
          className="search-input"
          placeholder={placeholder}
          value={searchTerm}
          onChange={handleInputChange}
          onClick={(e) => e.stopPropagation()}
        />
        {isOpen ? (
          <FaChevronUp
            className="icon-search icon-search-up"
            onClick={toggleDropdown}
          />
        ) : (
          <FaChevronDown
            className="icon-search icon-search-down"
            onClick={toggleDropdown}
          />
        )}
      </div>
      {isOpen && (
        <div className="options" ref={dropdownRef}>
          <div className="dropdown-list">
            {filteredOptions.length ? (
              filteredOptions.map((option) => (
                <div
                  key={option.value}
                  className="option"
                  onClick={() => handleOptionClick(option)}
                >
                  {option.label}
                </div>
              ))
            ) : (
              <div className="option-none">No Matches Found</div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default Search;
