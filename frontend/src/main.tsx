import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import "@fontsource/ibm-plex-sans/400.css";
import "@fontsource/ibm-plex-sans/500.css";
import "@fontsource/ibm-plex-sans/600.css";
import "@fontsource/ibm-plex-sans/700.css";
import "@fontsource/space-grotesk/500.css";
import "@fontsource/space-grotesk/700.css";
import { App } from "./app/App";
import { AppUpdateMonitor } from "./components/AppUpdateMonitor";
import { AuthProvider } from "./lib/auth";
import { ThemeProvider } from "./lib/theme";
import { I18nProvider } from "./lib/i18n";
import { WorkspaceStatusProvider } from "./lib/workspace-status";
import "./styles/index.css";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <BrowserRouter>
      <ThemeProvider>
        <I18nProvider>
          <WorkspaceStatusProvider>
            <AppUpdateMonitor />
            <AuthProvider>
              <App />
            </AuthProvider>
          </WorkspaceStatusProvider>
        </I18nProvider>
      </ThemeProvider>
    </BrowserRouter>
  </React.StrictMode>,
);
