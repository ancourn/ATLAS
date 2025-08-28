import { useEffect, useRef, useState } from "react";
import * as Y from "yjs";
import { WebsocketProvider } from "y-websocket";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import axios from "axios";
export default function Docs() {
  const [connected, setConnected] = useState(false);
  const [docId, setDocId] = useState("default-doc");
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [streaming, setStreaming] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);
  const editor = useEditor({ extensions:[StarterKit], content: "<p>Loading...</p>" });
  useEffect(() => {
    const ydoc = new Y.Doc();
    const provider = new WebsocketProvider((process.env.NEXT_PUBLIC_YWS || "ws://localhost:1234"), docId, ydoc);
    provider.on("status", (ev:any) => setConnected(ev.status === "connected"));
    const yText = ydoc.getText("prosemirror");
    // load persisted snapshot
    axios.get((process.env.NEXT_PUBLIC_API_BASE || "http://localhost:9300") + `/api/docs/load/${docId}`, { withCredentials: true })
      .then(r => {
        if (r.data?.content) { editor?.commands.setContent(r.data.content); yText.delete(0,yText.length); yText.insert(0, r.data.content); }
      }).catch(()=>{});
    // persist periodically and upsert embeddings
    const persist = setInterval(async () => {
      const content = editor?.getHTML() || yText.toString();
      await axios.post((process.env.NEXT_PUBLIC_API_BASE || "http://localhost:9300") + `/api/docs/save/${docId}`, { content }, { withCredentials: true }).catch(()=>{});
      // trigger embedding upsert
      axios.post((process.env.NEXT_PUBLIC_EMB_BASE || "http://localhost:9500") + "/v1/upsert", { id: docId, content }, { withCredentials: true }).catch(()=>{});
    }, 5000);
    return () => {
      clearInterval(persist);
      provider.destroy();
      ydoc.destroy();
    };
  }, [editor, docId]);
  // Copilot: request streaming suggestions for current editor selection or prompt
  async function startLiveCopilot(prompt?:string) {
    stopCopilot();
    const q = prompt || (editor ? editor.getText().slice(0,300) : "");
    const url = (process.env.NEXT_PUBLIC_CO_PILOT || "http://localhost:9502") + "/v1/live?q=" + encodeURIComponent(q);
    setSuggestions([]); setStreaming(true);
    const es = new EventSource(url);
    eventSourceRef.current = es;
    es.addEventListener("chunk", (ev:any) => {
      const data = ev.data.replace(/\\n/g, "\n");
      // append last chunk to suggestions[0] for streaming display
      setSuggestions(prev => {
        const arr = prev.slice();
        if (arr.length===0) arr.push(data); else arr[arr.length-1] = arr[arr.length-1] + data;
        return arr;
      });
    });
    es.addEventListener("done", ()=>{ setStreaming(false); es.close(); eventSourceRef.current = null; });
    es.addEventListener("error", (e:any) => { setStreaming(false); es.close(); eventSourceRef.current = null; });
    // prepopulate with empty final item
    setSuggestions([""]);
  }
  function stopCopilot() {
    if (eventSourceRef.current) { eventSourceRef.current.close(); eventSourceRef.current = null; setStreaming(false); }
  }
  // Apply suggestion to editor (insert at selection)
  function applySuggestion(idx:number) {
    if (!editor) return;
    const text = suggestions[idx];
    editor.chain().focus().insertContent(`<p>${text}</p>`).run();
  }
  // Audio recording (browser MediaRecorder)
  const recorderRef = useRef<MediaRecorder|null>(null);
  const [recording, setRecording] = useState(false);
  async function startRecording() {
    setSuggestions([]); // clear
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    const mr = new MediaRecorder(stream);
    const chunks: Blob[] = [];
    mr.ondataavailable = (e) => chunks.push(e.data);
    mr.onstop = async () => {
      const blob = new Blob(chunks, { type: 'audio/webm' });
      const fd = new FormData();
      fd.append("file", blob, "voice.webm");
      try {
        const res = await axios.post((process.env.NEXT_PUBLIC_TRANSCRIBE || "http://localhost:9501") + "/v1/transcribe", fd, { headers: { "Content-Type": "multipart/form-data" }});
        const text = res.data.text || res.data;
        // launch copilot call with transcript
        startLiveCopilot(text);
      } catch(e) {
        alert("Transcription failed: " + e);
      }
    };
    mr.start();
    recorderRef.current = mr;
    setRecording(true);
  }
  function stopRecording() {
    if (recorderRef.current) {
      recorderRef.current.stop();
      setRecording(false);
    }
  }
  return (
    <div style={{ display: "flex", gap: 16, padding: 20 }}>
      <div style={{ flex: 1 }}>
        <h2>Docs — collaborative (Yjs)</h2>
        <div>Status: {connected ? "connected" : "offline"}</div>
        <div style={{ margin: "8px 0" }}>
          <label>Doc ID: <input value={docId} onChange={(e)=>setDocId(e.target.value)} /></label>
        </div>
        <div style={{ border: "1px solid #ddd", padding: 12 }}>
          {editor && <EditorContent editor={editor} />}
        </div>
        <div style={{ marginTop: 12 }}>
          {!recording && <button onClick={startRecording}>🎤 Record Voice</button>}
          {recording && <button onClick={stopRecording}>⏹ Stop</button>}
          <button onClick={()=>startLiveCopilot() } style={{ marginLeft: 8 }}>Live Copilot</button>
          <button onClick={stopCopilot} style={{ marginLeft: 8 }}>Stop</button>
        </div>
      </div>
      <div style={{ width: 360, borderLeft: "1px solid #eee", paddingLeft: 12 }}>
        <h3>Copilot Suggestions</h3>
        {streaming && <div style={{ color: "#0a84ff" }}>Streaming...</div>}
        {suggestions.length===0 && <div>No suggestions yet</div>}
        {suggestions.map((s, i) => (
          <div key={i} style={{ marginBottom: 12 }}>
            <div style={{ background: "#f9f9f9", padding: 8, borderRadius: 6, whiteSpace: "pre-wrap" }}>{s}</div>
            <div style={{ marginTop: 6 }}>
              <button onClick={()=>applySuggestion(i)}>Apply</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}