import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios';
import { firebaseAuth } from '@/firebase';
import { exchangeToken } from './auth/authService';

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
});

let isRefreshing = false;
let failedQueue: Array<{
   resolve: () => void;
  reject: (error: AxiosError) => void;
}> = [];

const processQueue = (error: AxiosError | null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(); 
    }
  });
  failedQueue = [];
};


apiClient.interceptors.response.use(
  (response) => response, 
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

    if (error.response?.status === 401 && originalRequest && !originalRequest._retry) {
      const user = firebaseAuth.currentUser;
      if (!user) {
        // If there's no active Firebase user, do not attempt refresh or force redirect.
        // Let the 401 propagate normally so the application router can handle it cleanly.
        return Promise.reject(error);
      }

      // If a token refresh cycle is already ongoing, queue this request
      if (isRefreshing) {
        return new Promise<void>(function(resolve, reject) {
          failedQueue.push({ resolve, reject });
        })
      .then(() => {
          // Update authorization header with the fresh token issued while this request waited
          return apiClient(originalRequest);
        })
      .catch((err) => Promise.reject(err));
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        await exchangeToken(user);
        processQueue(null);
        return apiClient(originalRequest);

      } catch (refreshError: any) {
        processQueue(refreshError as AxiosError);
        
        // If the token exchange fails due to MFA requirement, don't force a signout/redirect.
        // Let the error propagate so the UI can redirect the user to the TOTP challenge page.
        if (refreshError.message !== 'mfa_required') {
          await firebaseAuth.signOut().catch(() => {});
          window.location.href = '/login';
        }
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  }
);

export default apiClient;