import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./main.css";
import { createBrowserRouter, Link } from "react-router";
import { RouterProvider } from "react-router/dom";
import { buildRoutes, routePath } from "./routes.ts";
import { ModalProvider } from "./components/modal.tsx";

const router = createBrowserRouter(buildRoutes());

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ModalProvider>
      <RouterProvider router={router} />
    </ModalProvider>
  </StrictMode>,
);
