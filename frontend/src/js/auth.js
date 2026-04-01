import api, { showToast } from './api.js';

document.addEventListener('DOMContentLoaded', () => {
  const loginForm = document.getElementById('loginForm');
  const registerForm = document.getElementById('registerForm');
  const toggleButtons = document.querySelectorAll('.toggle-form-btn');
  const loading = document.getElementById('authLoading');

  // 初始化时检查是否已登录
  const token = localStorage.getItem('opspilot_token');
  if (token) {
    window.location.href = '/index.html';
    return;
  }

  // 切换表单
  toggleButtons.forEach((btn) => {
    btn.addEventListener('click', (e) => {
      const target = e.target.getAttribute('data-target');
      if (target === 'register') {
        loginForm.classList.remove('active');
        registerForm.classList.add('active');
      } else {
        registerForm.classList.remove('active');
        loginForm.classList.add('active');
      }
    });
  });

  const setLoading = (isLoading) => {
    if (isLoading) {
      loading.classList.remove('hidden');
    } else {
      loading.classList.add('hidden');
    }
  };

  // 登录逻辑
  loginForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    const usernameInput = document.getElementById('loginUsername').value;
    const passwordInput = document.getElementById('loginPassword').value;

    if (!usernameInput || !passwordInput) {
      showToast('请填写用户名和密码', 'error');
      return;
    }

    setLoading(true);
    try {
      const response = await api.post('/auth/login', {
        username: usernameInput,
        password: passwordInput,
      });

      // API 返回格式可能是 { data: { userId, accessToken... } } 或 { userId, accessToken... }
      // 根据 OpsPilot 的 response.OK()
      const data = response.data || response;

      if (data.accessToken) {
        localStorage.setItem('opspilot_token', data.accessToken);
        localStorage.setItem('opspilot_user', JSON.stringify({ userId: data.userId, username: usernameInput }));
        showToast('登录成功！', 'success');
        
        setTimeout(() => {
          window.location.href = '/index.html';
        }, 800);
      } else {
        throw new Error('未获取到有效的令牌');
      }
    } catch (err) {
      const msg = err.response?.data?.error || err.response?.data?.message || '登录失败，请检查用户名或密码';
      showToast(msg, 'error');
    } finally {
      setLoading(false);
    }
  });

  // 注册逻辑
  registerForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    const usernameInput = document.getElementById('regUsername').value;
    const passwordInput = document.getElementById('regPassword').value;

    if (passwordInput.length < 6) {
      showToast('密码长度不能少于6位', 'error');
      return;
    }

    setLoading(true);
    try {
      await api.post('/auth/register', {
        username: usernameInput,
        password: passwordInput,
      });

      showToast('注册成功！请登录', 'success');
      
      // 切换回登录
      setTimeout(() => {
        document.getElementById('loginUsername').value = usernameInput;
        document.getElementById('loginPassword').value = '';
        registerForm.classList.remove('active');
        loginForm.classList.add('active');
      }, 1000);
    } catch (err) {
      const msg = err.response?.data?.error || err.response?.data?.message || '注册失败，可能用户名已存在';
      showToast(msg, 'error');
    } finally {
      setLoading(false);
    }
  });
});
