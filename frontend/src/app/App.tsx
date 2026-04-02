import { Navigate, Route, Routes } from "react-router-dom";
import { AppLayout } from "../components/AppLayout";
import { DashboardPage } from "./DashboardPage";
import { LoginPage } from "../features/auth/LoginPage";
import { ConversationsPage } from "../features/conversations/ConversationsPage";
import { ConversationDetailPage } from "../features/conversations/ConversationDetailPage";
import { TicketsPage } from "../features/tickets/TicketsPage";
import { TicketDetailPage } from "../features/tickets/TicketDetailPage";
import { SettingsPage } from "../features/settings/SettingsPage";
import { ProtectedRoute } from "../routes/ProtectedRoute";

export function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <AppLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<DashboardPage />} />
        <Route path="conversations" element={<ConversationsPage />} />
        <Route path="conversations/:id" element={<ConversationDetailPage />} />
        <Route path="tickets" element={<TicketsPage />} />
        <Route path="tickets/:id" element={<TicketDetailPage />} />
        <Route path="settings" element={<SettingsPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
