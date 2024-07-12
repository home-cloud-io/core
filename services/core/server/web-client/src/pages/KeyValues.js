import React, { useState } from "react";
import { useSelector, useDispatch } from "react-redux";

import { decrement, increment, incrementByAmount } from "../store/counter";

import Button from "../components/Button";
import Title from "../components/Title";
import Search from "../components/Search";
import Snackbar from "../components/Snackbar";

export default function KeyValuesPage() {
  const count = useSelector((state) => state.counter.value);
  const dispatch = useDispatch();

  const [selectedOption, setSelectedOption] = useState(null);
  const [snackbarMessage, setSnackbarMessage] = useState("");
  const [snackbarType, setSnackbarType] = useState("");

  const handleSelect = (option) => {
    setSelectedOption(option);
  };

  const clickApi = () => {
    console.log(GetValue);
  };

  const options = [
    { label: "Option 1", value: "option1" },
    { label: "Option 2", value: "option2" },
    { label: "Another Option", value: "anotherOption" },
    { label: "Something Else", value: "somethingElse" },
    { label: "One more thing", value: "oneMoreThing" },
    { label: "Another Thing", value: "anotherThing" },
    { label: "Some Other Stuff", value: "someOtherStuff" },
    { label: "Option 7", value: "option7" },
    { label: "Option 8", value: "option8" },
    { label: "Another Option 2", value: "anotherOption2" },
  ];

  // ------ TESTING SNACKBAR (Display only. TODO: Connect to storage)
  const snackbarSuccess = () => {
    setSnackbarMessage("Key/Value pair saved!");
    setSnackbarType("success");

    setTimeout(() => {
      setSnackbarMessage("");
      setSnackbarType("");
    }, 3000);
  };

  const snackbarFailure = () => {
    setSnackbarMessage("Failed to save Key/Value pair!");
    setSnackbarType("failure");

    setTimeout(() => {
      setSnackbarMessage("");
      setSnackbarType("");
    }, 2000);
  };
  // ----- TESTING SNACKBAR

  return (
    <div className="keyvalue-container">
      <div className="keyvalue-search">
        <Search
          placeholder="Search Key/Value"
          options={options}
          onSelect={handleSelect}
        />
      </div>

      <div className="card">
        <Title text="Counter RTK Test:" />
        <span className="keyvalue-rtkcounter">{count}</span>
        <br />
        <div className="keyvalue-cardfooter">
          <Button
            type="outline"
            text="Increment"
            onClick={() => dispatch(increment())}
          />
          <br />
          <Button
            text="Decrement"
            type="outline"
            onClick={() => dispatch(decrement())}
          />
          <br />
          <Button
            text="Add 10"
            type="outline"
            onClick={() => dispatch(incrementByAmount(10))}
          />
        </div>
      </div>

      <div className="card">
        <Title text="Set:" />
        <div className="keyvalue-cardfooter">
          <Button type="outline" text="Set" onClick={snackbarSuccess} />
        </div>
      </div>

      <div className="card">
        <Title text="Get:" />
      </div>

      <div className="card">
        <Title text="Remove:" />
      </div>

      <div className="card">
        <Title text="List:" />
      </div>
      <div className="snackbar-test">
        <Button text="Test Save Success" onClick={snackbarSuccess} />
        <Button text="Test Save Fail" onClick={snackbarFailure} />
      </div>

      {snackbarMessage && (
        <Snackbar message={snackbarMessage} type={snackbarType} />
      )}
    </div>
  );
}
