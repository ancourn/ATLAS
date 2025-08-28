import axios from 'axios';
import Cookies from 'js-cookie';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:9100';

interface User {
  id: string;
  name: string;
  email: string;
}

interface LoginResponse {
  token: string;
  user: User;
  expires_at: number;
}

interface RegisterData {
  name: string;
  email: string;
  password: string;
}

interface LoginData {
  email: string;
  password: string;
}

class AuthService {
  private token: string | null = null;

  constructor() {
    // Initialize token from cookie or localStorage
    this.token = Cookies.get('atlas_token') || localStorage.getItem('atlas_token');
  }

  async register(data: RegisterData): Promise<LoginResponse> {
    try {
      const response = await axios.post(`${API_BASE}/api/register`, data);
      this.setToken(response.data.token);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async login(data: LoginData): Promise<LoginResponse> {
    try {
      const response = await axios.post(`${API_BASE}/api/login`, data);
      this.setToken(response.data.token);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async logout(): Promise<void> {
    try {
      if (this.token) {
        await axios.post(`${API_BASE}/api/logout`, {}, {
          headers: this.getAuthHeaders()
        });
      }
    } catch (error) {
      console.error('Logout error:', error);
    } finally {
      this.clearToken();
    }
  }

  async getCurrentUser(): Promise<User | null> {
    try {
      if (!this.token) {
        // Fallback to cookie-based auth for demo
        const response = await axios.get(`${API_BASE}/api/whoami`);
        return response.data;
      }
      
      const response = await axios.get(`${API_BASE}/api/user`, {
        headers: this.getAuthHeaders()
      });
      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response?.status === 401) {
        this.clearToken();
        return null;
      }
      throw this.handleError(error);
    }
  }

  async updateUser(data: { name?: string; email?: string }): Promise<User> {
    try {
      const response = await axios.put(`${API_BASE}/api/user`, data, {
        headers: this.getAuthHeaders()
      });
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  isAuthenticated(): boolean {
    return !!this.token;
  }

  getToken(): string | null {
    return this.token;
  }

  getAuthHeaders(): Record<string, string> {
    if (!this.token) {
      return {};
    }
    return {
      'Authorization': `Bearer ${this.token}`
    };
  }

  private setToken(token: string): void {
    this.token = token;
    localStorage.setItem('atlas_token', token);
    Cookies.set('atlas_token', token, { expires: 1 }); // 1 day
  }

  private clearToken(): void {
    this.token = null;
    localStorage.removeItem('atlas_token');
    Cookies.remove('atlas_token');
  }

  private handleError(error: any): Error {
    if (axios.isAxiosError(error)) {
      const message = error.response?.data?.message || error.message;
      return new Error(message);
    }
    return error;
  }
}

export const authService = new AuthService();
export type { User, LoginResponse, RegisterData, LoginData };