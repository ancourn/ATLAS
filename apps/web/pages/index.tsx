import axios from "axios";
import { useEffect, useState } from "react";
export default function Home() {
  const [who, setWho] = useState<any>(null);
  useEffect(() => {
    axios.get((process.env.NEXT_PUBLIC_API_BASE || "http://localhost:9100") + "/api/whoami", { withCredentials: true })
      .then(r => setWho(r.data))
      .catch(() => setWho(null));
  }, []);
  return (
    <div style={{ fontFamily: "Inter, system-ui, sans-serif", maxWidth: 900, margin: "48px auto" }}>
      <h1>ATLAS — unified suite (MVP)</h1>
      <p>One inbox, docs, and AI copilots — local first. Starter repo.</p>
      <div style={{ display: "flex", gap: 16 }}>
        <div style={{ flex: 1, padding: 16, border: "1px solid #eee", borderRadius: 8 }}>
          <h3>Sign in (demo)</h3>
          <button onClick={() => {
            document.cookie = "atlas_user=user_demo; path=/";
            window.location.reload();
          }}>Sign in as demo</button>
          <button style={{ marginLeft: 8 }} onClick={() => {
            document.cookie = "atlas_user=; path=/; max-age=0";
            window.location.reload();
          }}>Sign out</button>
          <div style={{ marginTop: 12 }}>
            <strong>User:</strong> {who ? JSON.stringify(who) : "Anonymous"}
          </div>
        </div>
        <div style={{ flex: 2, padding: 16, border: "1px solid #eee", borderRadius: 8 }}>
          <h3>Unified Inbox (preview)</h3>
          <Inbox />
        </div>
      </div>
    </div>
  );
}
function Inbox() {
  const [msgs, setMsgs] = useState<any[]>([]);
  useEffect(() => {
    axios.get((process.env.NEXT_PUBLIC_API_BASE || "http://localhost:9200") + "/api/inbox")
      .then(r => setMsgs(r.data))
      .catch(() => setMsgs([]));
  }, []);
  return (
    <div>
      {msgs.length === 0 && <div>No messages yet — demo mode</div>}
      {msgs.map(m => (
        <div key={m.id} style={{ padding: 8, borderBottom: "1px solid #f0f0f0" }}>
          <div style={{ fontSize: 13, color: "#666" }}>{m.source} • {new Date(m.time).toLocaleString()}</div>
          <div style={{ fontWeight: 600 }}>{m.subject}</div>
          <div style={{ color: "#333" }}>{m.body}</div>
        </div>
      ))}
    </div>
  );
}