import { StrictMode } from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { initI18n } from "@/i18n";
import { App } from "./App";
import "./styles/index.css";

async function bootstrap() {
  await initI18n();
  const rootElement = document.getElementById("root")!;
  if (!rootElement.innerHTML) {
    ReactDOM.createRoot(rootElement).render(
      <StrictMode>
        <BrowserRouter>
          <App />
        </BrowserRouter>
      </StrictMode>,
    );
  }
}

void bootstrap();
