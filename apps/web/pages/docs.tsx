import { useEffect, useState, useRef } from 'react';
import axios from 'axios';

interface Document {
  id: string;
  title: string;
  content: string;
  created_at: string;
  updated_at: string;
}

export default function DocsPage() {
  const [documents, setDocuments] = useState<Document[]>([]);
  const [currentDoc, setCurrentDoc] = useState<Document | null>(null);
  const [content, setContent] = useState('');
  const [isConnected, setIsConnected] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState('Connecting...');
  const wsRef = useRef<WebSocket | null>(null);
  const clientIdRef = useRef<string>('');

  useEffect(() => {
    // Load documents
    loadDocuments();
    
    // Connect to WebSocket
    connectWebSocket();
    
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, []);

  const loadDocuments = async () => {
    try {
      const response = await axios.get(`${process.env.NEXT_PUBLIC_DOCS_BASE || 'http://localhost:9300'}/api/documents`);
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
      // Attempt to reconnect after 3 seconds
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
  };

  const handleContentChange = (newContent: string) => {
    setContent(newContent);
    
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN && currentDoc) {
      const update = {
        document_id: currentDoc.id,
        content: newContent,
        client_id: clientIdRef.current
      };
      wsRef.current.send(JSON.stringify(update));
    }
  };

  const createNewDocument = async () => {
    try {
      const newDoc = {
        title: 'New Document',
        content: 'Start typing here...'
      };
      const response = await axios.post(`${process.env.NEXT_PUBLIC_DOCS_BASE || 'http://localhost:9300'}/api/documents`, newDoc);
      setDocuments(prev => [...prev, response.data]);
      setCurrentDoc(response.data);
      setContent(response.data.content);
    } catch (error) {
      console.error('Error creating document:', error);
    }
  };

  return (
    <div style={{ fontFamily: "Inter, system-ui, sans-serif", maxWidth: 1200, margin: "48px auto" }}>
      <h1>ATLAS Docs — Real-time Collaborative Editor</h1>
      <p>Collaborate in real-time with CRDT-powered synchronization</p>
      
      <div style={{ display: "flex", gap: 16, marginBottom: 16 }}>
        <div style={{ 
          padding: "8px 16px", 
          borderRadius: "4px", 
          backgroundColor: isConnected ? "#d4edda" : "#f8d7da",
          color: isConnected ? "#155724" : "#721c24",
          fontSize: "14px"
        }}>
          Status: {connectionStatus}
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
                    transition: "background-color 0.2s"
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
                  <div style={{ fontWeight: 600, fontSize: "14px" }}>{doc.title}</div>
                  <div style={{ fontSize: "12px", color: "#666" }}>
                    {new Date(doc.updated_at).toLocaleString()}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Editor */}
        <div style={{ flex: 3, padding: 16, border: "1px solid #eee", borderRadius: 8 }}>
          <h3>
            {currentDoc ? currentDoc.title : 'Select a document'}
            {currentDoc && (
              <span style={{ fontSize: "14px", color: "#666", marginLeft: "8px" }}>
                (ID: {currentDoc.id})
              </span>
            )}
          </h3>
          
          {currentDoc ? (
            <div style={{ marginTop: 12 }}>
              <textarea
                value={content}
                onChange={(e) => handleContentChange(e.target.value)}
                style={{
                  width: "100%",
                  minHeight: "400px",
                  padding: "12px",
                  border: "1px solid #ddd",
                  borderRadius: "4px",
                  fontFamily: "monospace",
                  fontSize: "14px",
                  lineHeight: "1.5",
                  resize: "vertical"
                }}
                placeholder="Start typing..."
              />
              <div style={{ marginTop: 8, fontSize: "12px", color: "#666" }}>
                {isConnected ? "✓ Real-time collaboration active" : "⚠ Disconnected from server"}
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