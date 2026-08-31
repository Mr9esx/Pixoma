import { StrictMode } from "react";
import ReactDOM from "react-dom/client";
import { App } from "./App";
import { ThemeProvider } from "@/context/theme-provider";
import "./styles/index.css";

function bootstrap() {
  const rootElement = document.getElementById("root")!;
  if (!rootElement.innerHTML) {
    ReactDOM.createRoot(rootElement).render(
      <StrictMode>
        <ThemeProvider>
          <App />
        </ThemeProvider>
      </StrictMode>,
    );
  }
}

void bootstrap();
