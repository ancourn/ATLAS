import { useEffect, useRef, useState } from "react";
import * as Y from "yjs";
import { WebsocketProvider } from "y-websocket";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import axios from "axios";
export default function Docs() {
  const [connected, setConnected] = useState(false);
  const [docId, setDocId] = useState("default-doc");
  const ydocRef = useRef<Y.Doc | null>(null);
  const providerRef = useRef<any>(null);
  const editor = useEditor({
    extensions: [StarterKit],
    content: "<p>Loading...</p>",
  });
  useEffect(() => {
    const ydoc = new Y.Doc();
    ydocRef.current = ydoc;
    const wsUrl = (process.env.NEXT_PUBLIC_YWS || "ws://localhost:1234");
    const provider = new WebsocketProvider(wsUrl.replace(/^http/, "ws"), docId, ydoc);
    providerRef.current = provider;
    provider.on("status", (ev: any) => {
      setConnected(ev.status === "connected");
    });
    // bind Y.Text to editor
    const yText = ydoc.getText("prosemirror"); // we'll map to TipTap's content via snapshots
    // apply initial load from server (persistence)
    axios.get((process.env.NEXT_PUBLIC_API_BASE || "http://localhost:9300") + `/api/docs/load/${docId}`, { withCredentials: true })
      .then(r => {
        if (r.data && r.data.content) {
          // set editor content & yText
          editor?.commands.setContent(r.data.content);
          yText.delete(0, yText.length);
          yText.insert(0, r.data.content);
        }
      }).catch(()=>{});
    // On ydoc updates, sync editor content
    const updateHandler = () => {
      const content = yText.toString();
      // naive mapping: set raw HTML content into editor (TipTap expects html)
      try { editor?.commands.setContent(content); } catch {}
    };
    yText.observe(updateHandler);
    // Periodic persistence: push snapshot to server every 5s
    const persistInterval = setInterval(() => {
      const content = editor?.getHTML() || yText.toString();
      axios.post((process.env.NEXT_PUBLIC_API_BASE || "http://localhost:9300") + `/api/docs/save/${docId}`, { content }, { withCredentials: true })
        .catch(()=>{});
      // after saving snapshot to persistence, trigger embedding upsert
      axios.post((process.env.NEXT_PUBLIC_EMB_BASE || "http://localhost:9500") + "/v1/upsert", { id: docId, content }, { withCredentials: true }).catch(()=>{});
    }, 5000);
    return () => {
      clearInterval(persistInterval);
      yText.unobserve(updateHandler);
      provider.destroy();
      ydoc.destroy();
    };
  }, [editor, docId]);
  return (
    <div style={{ maxWidth: 900, margin: 24 }}>
      <h2>Docs — collaborative (Yjs)</h2>
      <div>Status: {connected ? "connected" : "offline"}</div>
      <div style={{ margin: "12px 0" }}>
        <label>Doc ID: <input value={docId} onChange={(e)=>setDocId(e.target.value)} /></label>
      </div>
      <div style={{ margin: "12px 0" }}>
        <button onClick={async () => {
          const content = editor?.getHTML() || "";
          const url = (process.env.NEXT_PUBLIC_AI_BASE || "http://localhost:9400") + "/v1/summarize";
          const r = await axios.post(url, { text: content }, { withCredentials: true });
          alert("Summary:\n\n" + (r.data.summary || JSON.stringify(r.data)));
        }} style={{ marginLeft: 8 }}>Summarize (AI)</button>
      </div>
      <div style={{ border: "1px solid #ddd", padding: 12 }}>
        {editor && <EditorContent editor={editor} />}
      </div>
    </div>
  );
}