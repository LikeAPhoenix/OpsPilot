import axios from 'axios';

// 基本配置
const api = axios.create({
  baseURL: '/api',
  timeout: 300000, // 300s timeout for long-running AI tasks
});

// 请求拦截器，携带 Token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('opspilot_token');
    if (token) {
      config.headers['Authorization'] = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// 响应拦截器，处理全局错误
api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    // 处理未授权
    if (error.response && error.response.status === 401) {
      // 只有不在 login.html 页面才跳转
      if (!window.location.pathname.includes('login.html')) {
        localStorage.removeItem('opspilot_token');
        localStorage.removeItem('opspilot_user');
        window.location.href = '/login.html';
      }
    }
    return Promise.reject(error);
  }
);

// Toast 通知功能
export const showToast = (message, type = 'info') => {
  const container = document.getElementById('toastContainer');
  if (!container) return;

  const toast = document.createElement('div');
  toast.className = `toast toast-${type}`;
  toast.textContent = message;

  container.appendChild(toast);

  // Trigger animation
  requestAnimationFrame(() => {
    toast.classList.add('show');
  });

  setTimeout(() => {
    toast.classList.remove('show');
    setTimeout(() => {
      container.removeChild(toast);
    }, 300); // Wait for transition out
  }, 3000);
};

export default api;
