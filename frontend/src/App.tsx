import { Title } from "@solidjs/meta";
import { Loading } from "solid-js";
import "./styles/global.css";
import Shell from "./components/Shell";
import { PreferenceProvider } from "./preferences/Preferences";
import { Router } from "./router";

export default function App() {
  return (
    <Router>
      {(props) => (
        <PreferenceProvider>
          <Title>TaoTale</Title>
          <Shell>
            <Loading fallback={<p role="status">Loading page…</p>}>{props.children}</Loading>
          </Shell>
        </PreferenceProvider>
      )}
    </Router>
  );
}
