import { useEffect, useState, useRef } from 'react';
import axios from 'axios';
import { authService, User } from '../lib/auth';
import { aiService, Document as DocumentType, AISummaryResponse } from '../lib/ai';

interface Document {
  id: string;
  title: string;
  content: string;
  user_id: string;
  created_at: string;
  updated_at: string;
  version: number;
}

export default function DocsPage() {
  const [user, setUser] = useState<User | null>(null);
  const [documents, setDocuments] = useState<Document[]>([]);
  const [currentDoc, setCurrentDoc] = useState<Document | null>(null);
  const [content, setContent] = useState('');
  const [isConnected, setIsConnected] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState('Connecting...');
  const [aiSummary, setAiSummary] = useState<AISummaryResponse | null>(null);
  const [summarizing, setSummarizing] = useState(false);
  const [loading, setLoading] = useState(true);
  const wsRef = useRef<WebSocket | null>(null);
  const clientIdRef = useRef<string>('');

  useEffect(() => {
    loadUser();
    loadDocuments();
    connectWebSocket();
    
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, []);

  const loadUser = async () => {
    try {
      const currentUser = await authService.getCurrentUser();
      setUser(currentUser);
    } catch (error) {
      console.error('Error loading user:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadDocuments = async () => {
    try {
      const currentUser = await authService.getCurrentUser();
      const userId = currentUser?.id || 'user_demo'; // Fallback to demo user
      const response = await axios.get(`${process.env.NEXT_PUBLIC_DOCS_BASE || 'http://localhost:9300'}/api/documents?user_id=${userId}`);
      setDocuments(response.data);
      if (response.data.length > 0 && !currentDoc) {
        setCurrentDoc(response.data[0]);
        setContent(response.data[0].content);
      }
    } catch (error) {
      console.error('Error loading documents:', error);
    }
  };

  const connectWebSocket = () => {
    const wsUrl = `${process.env.NEXT_PUBLIC_DOCS_BASE || 'http://localhost:9300'}/ws`.replace('http', 'ws');
    wsRef.current = new WebSocket(wsUrl);

    wsRef.current.onopen = () => {
      setIsConnected(true);
      setConnectionStatus('Connected');
    };

    wsRef.current.onmessage = (event) => {
      const data = JSON.parse(event.data);
      
      if (data.type === 'initial') {
        clientIdRef.current = data.client_id;
        setDocuments(data.documents);
        if (data.documents.length > 0 && !currentDoc) {
          setCurrentDoc(data.documents[0]);
          setContent(data.documents[0].content);
        }
      } else if (data.type === 'update') {
        if (currentDoc && data.document_id === currentDoc.id && data.sender_id !== clientIdRef.current) {
          setContent(data.content);
          // Update the document in the list
          setDocuments(prev => prev.map(doc => 
            doc.id === data.document_id 
              ? { ...doc, content: data.content, updated_at: new Date().toISOString() }
              : doc
          ));
        }
      }
    };

    wsRef.current.onclose = () => {
      setIsConnected(false);
      setConnectionStatus('Disconnected');
      setTimeout(connectWebSocket, 3000);
    };

    wsRef.current.onerror = (error) => {
      console.error('WebSocket error:', error);
      setConnectionStatus('Error');
    };
  };

  const handleDocumentSelect = (doc: Document) => {
    setCurrentDoc(doc);
    setContent(doc.content);
    setAiSummary(null); // Clear AI summary when switching documents
  };

  const handleContentChange = (newContent: string) => {
    setContent(newContent);
    
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN && currentDoc) {
      const update = {
        document_id: currentDoc.id,
        content: newContent,
        client_id: clientIdRef.current,
        user_id: user?.id || 'user_demo'
      };
      wsRef.current.send(JSON.stringify(update));
    }
  };

  const createNewDocument = async () => {
    try {
      const userId = user?.id || 'user_demo';
      const newDoc = {
        title: 'New Document',
        content: 'Start typing here...',
        user_id: userId
      };
      const response = await axios.post(`${process.env.NEXT_PUBLIC_DOCS_BASE || 'http://localhost:9300'}/api/documents`, newDoc);
      setDocuments(prev => [...prev, response.data]);
      setCurrentDoc(response.data);
      setContent(response.data.content);
      setAiSummary(null);
    } catch (error) {
      console.error('Error creating document:', error);
    }
  };

  const handleAISummarize = async () => {
    if (!currentDoc) return;
    
    setSummarizing(true);
    try {
      const docForAI: DocumentType = {
        id: currentDoc.id,
        title: currentDoc.title,
        content: currentDoc.content,
        user_id: currentDoc.user_id
      };
      const summary = await aiService.summarizeDocument(docForAI);
      setAiSummary(summary);
    } catch (error) {
      console.error('Error summarizing document:', error);
    } finally {
      setSummarizing(false);
    }
  };

  const canEditDocument = (doc: Document) => {
    if (!user) return true; // Allow demo mode
    return doc.user_id === user.id;
  };

  if (loading) {
    return (
      <div style={{ fontFamily: "Inter, system-ui, sans-serif", maxWidth: 1200, margin: "48px auto", textAlign: "center" }}>
        <p>Loading...</p>
      </div>
    );
  }

  return (
    <div style={{ fontFamily: "Inter, system-ui, sans-serif", maxWidth: 1400, margin: "48px auto" }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "24px" }}>
        <div>
          <h1>ATLAS Docs — Real-time Collaborative Editor</h1>
          <p>Collaborate in real-time with CRDT-powered synchronization and AI assistance</p>
          {user && (
            <div style={{ fontSize: "14px", color: "#666", marginTop: "4px" }}>
              Logged in as: {user.name} ({user.email})
            </div>
          )}
        </div>
        <div style={{ display: "flex", gap: "12px", alignItems: "center" }}>
          <div style={{ 
            padding: "8px 16px", 
            borderRadius: "4px", 
            backgroundColor: isConnected ? "#d4edda" : "#f8d7da",
            color: isConnected ? "#155724" : "#721c24",
            fontSize: "14px"
          }}>
            {connectionStatus}
          </div>
          <button 
            onClick={createNewDocument}
            style={{
              backgroundColor: "#0070f3",
              color: "white",
              border: "none",
              padding: "8px 16px",
              borderRadius: "4px",
              cursor: "pointer",
              fontSize: "14px"
            }}
          >
            New Document
          </button>
        </div>
      </div>

      <div style={{ display: "flex", gap: 16 }}>
        {/* Document List */}
        <div style={{ flex: 1, padding: 16, border: "1px solid #eee", borderRadius: 8 }}>
          <h3>Documents</h3>
          <div style={{ marginTop: 12 }}>
            {documents.length === 0 ? (
              <div>No documents yet</div>
            ) : (
              documents.map(doc => (
                <div
                  key={doc.id}
                  onClick={() => handleDocumentSelect(doc)}
                  style={{
                    padding: "8px",
                    marginBottom: "8px",
                    border: "1px solid #ddd",
                    borderRadius: "4px",
                    cursor: "pointer",
                    backgroundColor: currentDoc?.id === doc.id ? "#f0f8ff" : "white",
                    transition: "background-color 0.2s",
                    opacity: canEditDocument(doc) ? 1 : 0.7
                  }}
                  onMouseEnter={(e) => {
                    if (currentDoc?.id !== doc.id) {
                      e.currentTarget.style.backgroundColor = "#f5f5f5";
                    }
                  }}
                  onMouseLeave={(e) => {
                    if (currentDoc?.id !== doc.id) {
                      e.currentTarget.style.backgroundColor = "white";
                    }
                  }}
                >
                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                    <div style={{ fontWeight: 600, fontSize: "14px" }}>{doc.title}</div>
                    {!canEditDocument(doc) && (
                      <span style={{ fontSize: "10px", color: "#999", backgroundColor: "#f0f0f0", padding: "2px 4px", borderRadius: "2px" }}>
                        Read-only
                      </span>
                    )}
                  </div>
                  <div style={{ fontSize: "12px", color: "#666" }}>
                    {new Date(doc.updated_at).toLocaleString()}
                  </div>
                  <div style={{ fontSize: "10px", color: "#999" }}>
                    v{doc.version}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Editor */}
        <div style={{ flex: 2, padding: 16, border: "1px solid #eee", borderRadius: 8 }}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 12 }}>
            <h3>
              {currentDoc ? currentDoc.title : 'Select a document'}
              {currentDoc && (
                <span style={{ fontSize: "14px", color: "#666", marginLeft: "8px" }}>
                  (ID: {currentDoc.id}, v{currentDoc.version})
                </span>
              )}
            </h3>
            {currentDoc && (
              <button
                onClick={handleAISummarize}
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
            )}
          </div>
          
          {aiSummary && (
            <div style={{ 
              marginBottom: 16, 
              padding: 12, 
              backgroundColor: "#f8f9fa", 
              borderRadius: 4,
              borderLeft: "4px solid #28a745"
            }}>
              <h4 style={{ marginTop: 0, marginBottom: 8, fontSize: "14px" }}>AI Summary</h4>
              <p style={{ marginBottom: 8, fontStyle: "italic", fontSize: "12px" }}>{aiSummary.summary}</p>
              {aiSummary.key_points && aiSummary.key_points.length > 0 && (
                <div style={{ marginTop: 8 }}>
                  <strong style={{ fontSize: "12px" }}>Key Points:</strong>
                  <ul style={{ marginTop: 4, paddingLeft: 20, fontSize: "11px" }}>
                    {aiSummary.key_points.map((point, index) => (
                      <li key={index}>{point}</li>
                    ))}
                  </ul>
                </div>
              )}
              {aiSummary.sentiment && (
                <div style={{ marginTop: 8, fontSize: "11px", color: "#666" }}>
                  <strong>Tone:</strong> {aiSummary.sentiment}
                </div>
              )}
            </div>
          )}
          
          {currentDoc ? (
            <div style={{ marginTop: 12 }}>
              <textarea
                value={content}
                onChange={(e) => handleContentChange(e.target.value)}
                disabled={!canEditDocument(currentDoc)}
                style={{
                  width: "100%",
                  minHeight: "400px",
                  padding: "12px",
                  border: "1px solid #ddd",
                  borderRadius: "4px",
                  fontFamily: "monospace",
                  fontSize: "14px",
                  lineHeight: "1.5",
                  resize: "vertical",
                  backgroundColor: canEditDocument(currentDoc) ? "white" : "#f8f9fa",
                  opacity: canEditDocument(currentDoc) ? 1 : 0.8
                }}
                placeholder={canEditDocument(currentDoc) ? "Start typing..." : "This document is read-only"}
              />
              <div style={{ marginTop: 8, fontSize: "12px", color: "#666", display: "flex", justifyContent: "space-between" }}>
                <div>
                  {isConnected ? "✓ Real-time collaboration active" : "⚠ Disconnected from server"}
                </div>
                <div>
                  {canEditDocument(currentDoc) ? "You can edit this document" : "Read-only access"}
                </div>
              </div>
            </div>
          ) : (
            <div style={{ marginTop: 12, color: "#666" }}>
              Select a document from the list to start editing, or create a new one.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}