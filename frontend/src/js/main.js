import { createIcons } from 'lucide';
import { icons } from 'lucide';

// Initialize Lucide icons
document.addEventListener('DOMContentLoaded', () => {
  createIcons({ icons });

  // Auth Check
  const token = localStorage.getItem('opspilot_token');
  if (!token) {
    window.location.href = '/login.html';
    return;
  }

  // Set Username
  try {
    const userStr = localStorage.getItem('opspilot_user');
    if (userStr) {
      const user = JSON.parse(userStr);
      document.getElementById('usernameDisplay').textContent = user.username;
    }
  } catch (e) {
    console.warn('Failed to parse user info', e);
  }

  // Logout
  document.getElementById('logoutBtn').addEventListener('click', () => {
    localStorage.removeItem('opspilot_token');
    localStorage.removeItem('opspilot_user');
    window.location.href = '/login.html';
  });

  // Mobile menu toggle
  const mobileMenuBtn = document.getElementById('mobileMenuBtn');
  const sidebar = document.getElementById('sidebar');
  if (mobileMenuBtn && sidebar) {
    mobileMenuBtn.addEventListener('click', () => {
      sidebar.classList.toggle('open');
    });
  }

  // Mode Dropdown
  const modeSelectorBtn = document.getElementById('modeSelectorBtn');
  const modeDropdownWrapper = document.getElementById('modeDropdownWrapper');
  const modeItems = document.querySelectorAll('.dropdown-item');
  const currentModeText = document.getElementById('currentModeText');
  
  if (modeSelectorBtn && modeDropdownWrapper) {
    modeSelectorBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      modeDropdownWrapper.classList.toggle('open');
    });

    // Handle clicks outside dropdown
    document.addEventListener('click', (e) => {
      if (!modeDropdownWrapper.contains(e.target)) {
        modeDropdownWrapper.classList.remove('open');
      }
    });

    modeItems.forEach(item => {
      item.addEventListener('click', () => {
        const mode = item.getAttribute('data-mode');
        
        modeItems.forEach(i => i.classList.remove('active'));
        item.classList.add('active');
        
        currentModeText.textContent = mode === 'quick' ? '快速' : '流式';
        modeDropdownWrapper.classList.remove('open');
        
        // Dispatch custom event for chat.js
        window.dispatchEvent(new CustomEvent('chatModeChanged', { detail: { mode }}));
      });
    });
  }
});
