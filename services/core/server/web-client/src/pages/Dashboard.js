import * as React from "react";
import { Routes, Route } from "react-router-dom";
import MetricsPage from "./Metrics";
import ServiceRegistryPage from "./ServiceRegistry";
import KeyValuesPage from "./KeyValues";
import ErrorPage from "./ErrorPage";
import Header from "../components/Header";
import MainContainer from "../components/MainContainer";
import Container from "../components/Container";
import Footer from "../components/Footer";

export default function Dashboard() {
  return (
    <>
      <Header />
      <MainContainer>
        <Container>
          <Routes>
            <Route index element={<MetricsPage />} />
            <Route path="service-registry" element={<ServiceRegistryPage />} />
            <Route path="key-values" element={<KeyValuesPage />} />
            {/* <Route path="*" element={<ErrorPage />} /> */}
          </Routes>
        </Container>
      </MainContainer>
      <Footer />
    </>
  );
}
