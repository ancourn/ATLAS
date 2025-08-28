import axios from 'axios';

const AI_BASE = process.env.NEXT_PUBLIC_AI_BASE || 'http://localhost:9400';

interface InboxMessage {
  id: string;
  source: string;
  from: string;
  to: string;
  subject: string;
  body: string;
  time: string;
}

interface AISummaryResponse {
  summary: string;
  suggestions?: string[];
  key_points?: string[];
  sentiment?: string;
  processing_time_ms: number;
}

interface AISearchResponse {
  results: SearchResult[];
  processing_time_ms: number;
}

interface SearchResult {
  document_id: string;
  title: string;
  snippet: string;
  relevance: number;
}

interface Document {
  id: string;
  title: string;
  content: string;
  user_id: string;
}

class AIService {
  async summarizeInbox(messages: InboxMessage[]): Promise<AISummaryResponse> {
    try {
      const response = await axios.post(`${AI_BASE}/api/ai/inbox/summarize`, messages);
      return response.data;
    } catch (error) {
      console.error('AI inbox summarization error:', error);
      // Return fallback summary
      return {
        summary: `Summary of ${messages.length} messages`,
        suggestions: ['Thank you for your message', 'I will get back to you soon'],
        key_points: ['Multiple messages received', 'Action may be required'],
        sentiment: 'neutral',
        processing_time_ms: 0
      };
    }
  }

  async summarizeDocument(document: Document): Promise<AISummaryResponse> {
    try {
      const response = await axios.post(`${AI_BASE}/api/ai/document/summarize`, document);
      return response.data;
    } catch (error) {
      console.error('AI document summarization error:', error);
      // Return fallback summary
      return {
        summary: 'Document summary unavailable',
        key_points: ['Document content', 'Review required'],
        sentiment: 'neutral',
        processing_time_ms: 0
      };
    }
  }

  async searchDocuments(query: string, documentIds: string[], maxResults: number = 5): Promise<AISearchResponse> {
    try {
      const response = await axios.post(`${AI_BASE}/api/ai/search`, {
        query,
        documents: documentIds,
        max_results: maxResults
      });
      return response.data;
    } catch (error) {
      console.error('AI search error:', error);
      // Return fallback results
      return {
        results: documentIds.slice(0, maxResults).map(id => ({
          document_id: id,
          title: `Document ${id}`,
          snippet: `Content related to "${query}"`,
          relevance: 0.5
        })),
        processing_time_ms: 0
      };
    }
  }
}

export const aiService = new AIService();
export type { InboxMessage, AISummaryResponse, AISearchResponse, SearchResult, Document };