import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios';
import { firebaseAuth } from '@/firebase';
import { exchangeToken } from './auth/authService';

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
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
      // Don't intercept auth attempt endpoints to prevent recursive infinite loops
      const requestUrl = originalRequest.url || '';
      if (
        requestUrl.includes('/auth/refresh') ||
        requestUrl.includes('/auth/login') ||
        requestUrl.includes('/auth/register') ||
        requestUrl.includes('/auth/logout')
      ) {
        return Promise.reject(error);
      }

      const user = firebaseAuth.currentUser;

      // If a token refresh cycle is already ongoing, queue this request
      if (isRefreshing) {
        return new Promise<void>(function(resolve, reject) {
          failedQueue.push({ resolve, reject });
        })
          .then(() => {
            return apiClient(originalRequest);
          })
          .catch((err) => Promise.reject(err));
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        if (user) {
          // Firebase user silent token exchange
          await exchangeToken(user);
        } else {
          // Direct backend username/password sliding session refresh
          const baseUrl = apiClient.defaults.baseURL || '';
          await axios.post(
            `${baseUrl}/api/auth/refresh`,
            {},
            { withCredentials: true }
          );
        }

        processQueue(null);
        return apiClient(originalRequest);

      } catch (refreshError: any) {
        processQueue(refreshError as AxiosError);
        
        // If the token exchange fails due to MFA requirement, don't force a signout/redirect.
        // Let the error propagate so the UI can redirect the user to the TOTP challenge page.
        const isMfaRequired =
          refreshError?.message === 'mfa_required' ||
          refreshError?.response?.data?.error === 'mfa_required';

        if (!isMfaRequired) {
          if (user) {
            await firebaseAuth.signOut().catch(() => {});
          }
          if (
            typeof window !== 'undefined' &&
            window.location.pathname !== '/login' &&
            window.location.pathname !== '/register' &&
            !requestUrl.includes('/auth/me')
          ) {
            window.location.href = '/login';
          }
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