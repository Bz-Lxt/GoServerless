import { Navigate, Route, Routes } from "react-router-dom";
import { getToken } from "./lib/api";
import Login from "./pages/Login";
import Shell from "./pages/Shell";
import Dashboard from "./pages/Dashboard";
import Editor from "./pages/Editor";

function Guard({ children }: { children: JSX.Element }) {
  if (!getToken()) return <Navigate to="/login" replace />;
  return children;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/"
        element={
          <Guard>
            <Shell />
          </Guard>
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="fn/:name" element={<Editor />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
