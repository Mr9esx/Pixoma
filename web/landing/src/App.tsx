import DuskLandingPage from "@/pages/dusk-landing-page";
import DuskLanding7Page from "@/pages/dusk-landing-7-page";

export function App() {
  const version = new URLSearchParams(window.location.search).get("version");

  return version === "7" ? <DuskLanding7Page /> : <DuskLandingPage />;
}
