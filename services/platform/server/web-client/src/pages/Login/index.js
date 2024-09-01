import * as React from "react";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useDispatch } from "react-redux";

import { useLoginMutation } from "../../services/web_rpc";
import logo from '../../../public/assets/home_cloud_logo.png';

import "./Login.css";

export default function Login() {
  const dispatch = useDispatch();
  const [login, result] = useLoginMutation();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const navigate = useNavigate();

  const handleSubmit = (e) => {
    e.preventDefault();    

    try {
      login({ username, password });
    } catch (error) {
      console.error(error);
    }

    navigate('/home');
  }

  // TODO: Add a loading spinner

  // TODO: Add a error message

  // TODO: Add validation for the form

  return (
    <>
        <main className="form-signin w-100 m-auto">
        <form>
            <img className="mb-4" src={logo} alt="" width="300" height="200" />
            <h1 className="h3 mb-3 fw-normal">Login</h1>

            <div className="form-floating">
            <input
              className="form-control"
              id="floatingInput"
              type="text"
              placeholder="Username" 
              onChange={e => setUsername(e.target.value)}/>

              <label htmlFor="floatingInput">Username</label>
            </div>

            <div className="form-floating">
              <input
                className="form-control"
                id="floatingPassword"
                type="password"
                placeholder="Password"
                onChange={e => setPassword(e.target.value)} />

                <label htmlFor="floatingPassword">Password</label>
            </div>

            <button
              className="btn btn-primary w-100 py-2"
              type="submit"
              onClick={e => handleSubmit(e)}>
                Sign in
              </button>
            <p className="mt-5 mb-3 text-body-secondary">&copy; 2024–2028</p>
        </form>
        </main>
    </>
  );
}