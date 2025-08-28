import { useEffect, useState } from "react";
import axios from "axios";
import { authService, User } from "../lib/auth";
import { aiService, InboxMessage, AISummaryResponse } from "../lib/ai";
import AuthModal from "../components/AuthModal";

export default function Home() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [showAuthModal, setShowAuthModal] = useState(false);
  const [aiSummary, setAiSummary] = useState<AISummaryResponse | null>(null);
  const [summarizing, setSummarizing] = useState(false);

  useEffect(() => {
    loadUser();
  }, []);

  const loadUser = async () => {
    try {
      const currentUser = await authService.getCurrentUser();
      setUser(currentUser);
    } catch (error) {
      console.error("Error loading user:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleAuthSuccess = () => {
    loadUser();
  };

  const handleSignOut = async () => {
    try {
      await authService.logout();
      setUser(null);
      setAiSummary(null);
    } catch (error) {
      console.error("Error signing out:", error);
    }
  };

  const handleDemoSignIn = () => {
    // For demo purposes, set a demo user
    document.cookie = "atlas_user=user_demo; path=/";
    setUser({
      id: "user_demo",
      name: "Demo User",
      email: "demo@atlas.local"
    });
  };

  const handleAISummarize = async (messages: InboxMessage[]) => {
    if (messages.length === 0) return;
    
    setSummarizing(true);
    try {
      const summary = await aiService.summarizeInbox(messages);
      setAiSummary(summary);
    } catch (error) {
      console.error("Error summarizing inbox:", error);
    } finally {
      setSummarizing(false);
    }
  };

  if (loading) {
    return (
      <div style={{ fontFamily: "Inter, system-ui, sans-serif", maxWidth: 900, margin: "48px auto", textAlign: "center" }}>
        <p>Loading...</p>
      </div>
    );
  }

  return (
    <div style={{ fontFamily: "Inter, system-ui, sans-serif", maxWidth: 1200, margin: "48px auto" }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "24px" }}>
        <div>
          <h1>ATLAS — unified suite (MVP)</h1>
          <p>One inbox, docs, and AI copilots — local first. Enhanced with authentication and AI.</p>
        </div>
        <div>
          {user ? (
            <div style={{ display: "flex", alignItems: "center", gap: "12px" }}>
              <span style={{ color: "#666" }}>
                Welcome, {user.name} ({user.email})
              </span>
              <button
                onClick={handleSignOut}
                style={{
                  padding: "8px 16px",
                  backgroundColor: "#dc3545",
                  color: "white",
                  border: "none",
                  borderRadius: "4px",
                  cursor: "pointer"
                }}
              >
                Sign Out
              </button>
            </div>
          ) : (
            <div style={{ display: "flex", gap: "12px" }}>
              <button
                onClick={handleDemoSignIn}
                style={{
                  padding: "8px 16px",
                  backgroundColor: "#6c757d",
                  color: "white",
                  border: "none",
                  borderRadius: "4px",
                  cursor: "pointer"
                }}
              >
                Demo Mode
              </button>
              <button
                onClick={() => setShowAuthModal(true)}
                style={{
                  padding: "8px 16px",
                  backgroundColor: "#0070f3",
                  color: "white",
                  border: "none",
                  borderRadius: "4px",
                  cursor: "pointer"
                }}
              >
                Sign In
              </button>
            </div>
          )}
        </div>
      </div>

      <div style={{ marginBottom: 24, display: "flex", gap: "16px" }}>
        <a 
          href="/docs" 
          style={{
            display: "inline-block",
            padding: "8px 16px",
            backgroundColor: "#0070f3",
            color: "white",
            textDecoration: "none",
            borderRadius: "4px"
          }}
        >
          Collaborative Docs →
        </a>
        <button
          onClick={() => window.open("https://github.com/ancourn/ATLAS", "_blank")}
          style={{
            padding: "8px 16px",
            backgroundColor: "#24292e",
            color: "white",
            border: "none",
            borderRadius: "4px",
            cursor: "pointer"
          }}
        >
          View on GitHub →
        </button>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "1fr 2fr", gap: 16, marginBottom: 16 }}>
        <div style={{ padding: 16, border: "1px solid #eee", borderRadius: 8 }}>
          <h3>User Status</h3>
          <div style={{ marginTop: 12 }}>
            <strong>Status:</strong> {user ? `Authenticated as ${user.name}` : "Anonymous/Demo"}
          </div>
          {user && (
            <div style={{ marginTop: 8 }}>
              <strong>User ID:</strong> {user.id}
            </div>
          )}
          <div style={{ marginTop: 8 }}>
            <strong>Features:</strong>
            <ul style={{ marginTop: 4, paddingLeft: 20 }}>
              <li>✅ Real-time collaborative documents</li>
              <li>✅ User authentication & permissions</li>
              <li>✅ AI-powered inbox summarization</li>
              <li>✅ Persistent document storage</li>
              <li>✅ CRDT conflict resolution</li>
            </ul>
          </div>
        </div>
        
        <div style={{ padding: 16, border: "1px solid #eee", borderRadius: 8 }}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 12 }}>
            <h3>Unified Inbox with AI Copilot</h3>
            <button
              onClick={() => {
                const messages = [
                  {
                    id: "1",
                    source: "email",
                    from: "alice@example.com",
                    to: "user@atlas.local",
                    subject: "Project Update",
                    body: "Great work on the latest deliverables. The client is very happy with the progress.",
                    time: new Date().toISOString()
                  },
                  {
                    id: "2",
                    source: "email",
                    from: "bob@example.com",
                    to: "user@atlas.local",
                    subject: "Meeting Request",
                    body: "Would you be available for a quick sync tomorrow afternoon to discuss the roadmap?",
                    time: new Date().toISOString()
                  }
                ];
                handleAISummarize(messages);
              }}
              disabled={summarizing}
              style={{
                padding: "6px 12px",
                backgroundColor: "#28a745",
                color: "white",
                border: "none",
                borderRadius: "4px",
                cursor: summarizing ? "not-allowed" : "pointer",
                fontSize: "12px"
              }}
            >
              {summarizing ? "Analyzing..." : "AI Summarize"}
            </button>
          </div>
          
          {aiSummary && (
            <div style={{ 
              marginBottom: 16, 
              padding: 12, 
              backgroundColor: "#f8f9fa", 
              borderRadius: 4,
              borderLeft: "4px solid #0070f3"
            }}>
              <h4 style={{ marginTop: 0, marginBottom: 8 }}>AI Summary</h4>
              <p style={{ marginBottom: 8, fontStyle: "italic" }}>{aiSummary.summary}</p>
              {aiSummary.sentiment && (
                <div style={{ fontSize: "12px", color: "#666" }}>
                  <strong>Sentiment:</strong> {aiSummary.sentiment}
                </div>
              )}
              {aiSummary.suggestions && aiSummary.suggestions.length > 0 && (
                <div style={{ marginTop: 8 }}>
                  <strong>Reply Suggestions:</strong>
                  <ul style={{ marginTop: 4, paddingLeft: 20, fontSize: "12px" }}>
                    {aiSummary.suggestions.map((suggestion, index) => (
                      <li key={index}>{suggestion}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}
          
          <Inbox onSummarize={handleAISummarize} />
        </div>
      </div>

      <AuthModal
        isOpen={showAuthModal}
        onClose={() => setShowAuthModal(false)}
        onAuthSuccess={handleAuthSuccess}
      />
    </div>
  );
}

function Inbox({ onSummarize }: { onSummarize: (messages: any[]) => void }) {
  const [msgs, setMsgs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadMessages();
  }, []);

  const loadMessages = async () => {
    try {
      const response = await axios.get((process.env.NEXT_PUBLIC_API_BASE || "http://localhost:9200") + "/api/inbox");
      setMsgs(response.data);
    } catch (error) {
      console.error("Error loading messages:", error);
      setMsgs([]);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div>Loading messages...</div>;
  }

  return (
    <div>
      {msgs.length === 0 ? (
        <div>No messages yet — demo mode</div>
      ) : (
        <>
          <div style={{ marginBottom: 8, fontSize: "12px", color: "#666" }}>
            {msgs.length} message{msgs.length !== 1 ? 's' : ''}
          </div>
          {msgs.map(m => (
            <div key={m.id} style={{ padding: 8, borderBottom: "1px solid #f0f0f0" }}>
              <div style={{ fontSize: 13, color: "#666" }}>
                {m.source} • {m.from} • {new Date(m.time).toLocaleString()}
              </div>
              <div style={{ fontWeight: 600 }}>{m.subject}</div>
              <div style={{ color: "#333" }}>{m.body}</div>
            </div>
          ))}
        </>
      )}
    </div>
  );
}