import api, { showToast } from './api.js';
import { marked } from 'marked';
import 'highlight.js/styles/github-dark.css';
import hljs from 'highlight.js';
import { icons } from 'lucide';

// Configure Marked & Highlight.js
marked.setOptions({
  breaks: true,
  gfm: true,
  highlight: function(code, lang) {
    if (lang && hljs.getLanguage(lang)) {
      return hljs.highlight(code, { language: lang }).value;
    }
    return hljs.highlightAuto(code).value;
  }
});

class ChatApp {
  constructor() {
    this.currentMode = 'quick';
    this.isStreaming = false;
    this.sessionId = null;
    this.isAIOps = false;
    this.chatHistories = JSON.parse(localStorage.getItem('opspilot_histories') || '[]');
    this.currentSessionMessages = [];
    
    // DOM Elements
    this.messageInput = document.getElementById('messageInput');
    this.sendBtn = document.getElementById('sendBtn');
    this.chatMessages = document.getElementById('chatMessages');
    this.welcomeBox = document.getElementById('welcomeBox');
    this.historyList = document.getElementById('chatHistoryList');
    this.newChatBtn = document.getElementById('newChatBtn');
    this.uploadFileBtn = document.getElementById('uploadFileBtn');
    this.fileInput = document.getElementById('fileInput');
    this.inputContainer = document.querySelector('.input-container');

    // Init
    this.bindEvents();
    this.initSessionFromUrl();
    this.renderHistoryList();
  }

  generateSessionId() {
    return ''; // 让后端自动生成
  }

  // Session & URL Management
  initSessionFromUrl() {
    const params = new URLSearchParams(window.location.search);
    const sid = params.get('session_id');

    if (sid) {
      const existing = this.chatHistories.find(h => h.id === sid);
      if (existing) {
        this.loadSession(existing);
      } else {
        // Unknown sessionId but provided in URL, create new session associated with this ID
        this.startNewSession(sid);
      }
    } else {
      // Default to new session without pushing history
      this.startNewSession(this.generateSessionId(), true);
    }
  }

  updateUrl(id) {
    const url = new URL(window.location);
    url.searchParams.set('session_id', id);
    window.history.replaceState({ sessionId: id }, '', url);
  }

  startNewSession(id, isInitialLoad = false) {
    this.sessionId = id || this.generateSessionId();
    this.isAIOps = false;
    if (this.inputContainer) this.inputContainer.classList.remove('hidden');
    this.currentSessionMessages = [];
    this.chatMessages.innerHTML = '';
    this.welcomeBox.classList.remove('hidden');
    this.chatMessages.appendChild(this.welcomeBox);
    
    if (!isInitialLoad && this.sessionId) {
      this.updateUrl(this.sessionId);
    }
    this.renderHistoryList();
  }

  loadSession(sessionData) {
    this.sessionId = sessionData.id;
    this.isAIOps = sessionData.isAIOps || false;
    if (this.inputContainer) {
      if (this.isAIOps) this.inputContainer.classList.add('hidden');
      else this.inputContainer.classList.remove('hidden');
    }
    this.currentSessionMessages = [...sessionData.messages];
    this.updateUrl(this.sessionId);
    
    this.chatMessages.innerHTML = '';
    this.welcomeBox.classList.add('hidden');
    
    this.currentSessionMessages.forEach(msg => {
      this.appendMessageElement(msg.role, msg.content, false);
    });
    
    this.renderHistoryList();
  }

  saveSessionHistory() {
    if (this.currentSessionMessages.length === 0) return;

    let session = this.chatHistories.find(h => h.id === this.sessionId);
    if (!session) {
      // Create new history
      const titleStr = this.currentSessionMessages.find(m => m.role === 'user')?.content || '新对话';
      session = {
        id: this.sessionId,
        title: titleStr.substring(0, 20) + (titleStr.length > 20 ? '...' : ''),
        timestamp: Date.now(),
        messages: [],
        isAIOps: this.isAIOps
      };
      this.chatHistories.unshift(session);
    }
    
    session.timestamp = Date.now();
    session.messages = [...this.currentSessionMessages];
    session.isAIOps = this.isAIOps;
    
    // Keep max 50 sessions
    if (this.chatHistories.length > 50) {
      this.chatHistories = this.chatHistories.slice(0, 50);
    }
    
    localStorage.setItem('opspilot_histories', JSON.stringify(this.chatHistories));
    this.renderHistoryList();
  }

  deleteSession(id) {
    this.chatHistories = this.chatHistories.filter(h => h.id !== id);
    localStorage.setItem('opspilot_histories', JSON.stringify(this.chatHistories));
    
    if (this.sessionId === id) {
      this.startNewSession();
    } else {
      this.renderHistoryList();
    }
  }

  renderHistoryList() {
    this.historyList.innerHTML = '';
    this.chatHistories.forEach(h => {
      const item = document.createElement('div');
      item.className = `history-item ${h.id === this.sessionId ? 'active' : ''}`;
      
      const titleSpan = document.createElement('span');
      titleSpan.className = 'history-item-title';
      titleSpan.textContent = h.title;
      titleSpan.onclick = () => this.loadSession(h);

      const delBtn = document.createElement('button');
      delBtn.className = 'history-delete-btn';
      delBtn.innerHTML = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"></path><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"></path><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"></path></svg>`;
      delBtn.onclick = (e) => {
        e.stopPropagation();
        this.deleteSession(h.id);
      }

      item.appendChild(titleSpan);
      item.appendChild(delBtn);
      this.historyList.appendChild(item);
    });
  }

  // Events & Sending
  bindEvents() {
    window.addEventListener('chatModeChanged', (e) => {
      this.currentMode = e.detail.mode;
    });

    this.newChatBtn.addEventListener('click', () => {
      if (!this.isStreaming) this.startNewSession(this.generateSessionId());
    });
    
    this.sendBtn.addEventListener('click', () => this.handleSend());
    this.messageInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        this.handleSend();
      }
    });

    // Auto resize textarea
    this.messageInput.addEventListener('input', () => {
      this.messageInput.style.height = 'auto';
      this.messageInput.style.height = (this.messageInput.scrollHeight) + 'px';
      if (this.messageInput.value === '') {
        this.messageInput.style.height = '40px';
      }
    });

    // File Upload
    this.uploadFileBtn.addEventListener('click', () => {
      this.fileInput.click();
    });

    this.fileInput.addEventListener('change', async (e) => {
      const file = e.target.files[0];
      if (!file) return;
      
      const formData = new FormData();
      formData.append('file', file);
      
      showToast(`正在上传 ${file.name}...`, 'info');
      
      try {
        const res = await api.post('/upload', formData, {
          headers: { 'Content-Type': 'multipart/form-data' }
        });
        showToast(`文件 ${res.data?.fileName || file.name} 上传并索引成功`, 'success');
      } catch (err) {
        showToast(`上传失败: ${err.message}`, 'error');
      } finally {
        this.fileInput.value = ''; // Reset
      }
    });
  }

  async handleSend() {
    if (this.isStreaming) return;
    
    const text = this.messageInput.value.trim();
    if (!text) return;

    // Reset input
    this.messageInput.value = '';
    this.messageInput.style.height = '40px';

    // UI Feedback
    this.welcomeBox.classList.add('hidden');
    
    // Store & Render User Message
    this.currentSessionMessages.push({ role: 'user', content: text });
    this.appendMessageElement('user', text, false);
    
    this.isStreaming = true;
    this.sendBtn.disabled = true;

    if (this.currentMode === 'quick') {
      await this.sendQuickMessage(text);
    } else {
      await this.sendStreamMessage(text);
    }

    this.isStreaming = false;
    this.sendBtn.disabled = false;
    this.saveSessionHistory();
  }

  async sendQuickMessage(text) {
    // Add loading placeholder
    const assistantEl = this.appendMessageElement('assistant', '', true);
    
    try {
      const response = await api.post('/chat', {
        Id: this.sessionId || "",
        Question: text
      });
      // response might be structured depending on api.js wrapper block
      const data = response.data || response;
      const answer = data.answer || '无响应内容';
      
      if (data.sessionId && this.sessionId !== data.sessionId) {
        this.sessionId = data.sessionId;
        this.updateUrl(this.sessionId);
      }
      
      // Stop typing anim and render markdown
      assistantEl.classList.remove('streaming');
      const contentEl = assistantEl.querySelector('.message-content');
      contentEl.innerHTML = marked.parse(answer);
      
      this.currentSessionMessages.push({ role: 'assistant', content: answer });
      
    } catch (err) {
      assistantEl.classList.remove('streaming');
      const contentEl = assistantEl.querySelector('.message-content');
      const errText = `调用失败: ${err.message}`;
      contentEl.innerHTML = `<span style="color:var(--color-primary)">${errText}</span>`;
      this.currentSessionMessages.push({ role: 'assistant', content: errText });
    }
  }

  async sendStreamMessage(text) {
    const assistantEl = this.appendMessageElement('assistant', '', true);
    const contentEl = assistantEl.querySelector('.message-content');
    let fullText = '';
    
    try {
      const token = localStorage.getItem('opspilot_token');
      // Create native fetch for SSE Stream reading processing since Axios isn't ideal for streams
      const response = await fetch('/api/chat_stream', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          Id: this.sessionId || "",
          Question: text
        })
      });

      if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
      
      const reader = response.body.getReader();
      const decoder = new TextDecoder('utf-8');
      
      let chunk = '';
      let isDone = false;
      let currentEvent = 'message';
      
      while (!isDone) {
        const { value, done } = await reader.read();
        
        if (done) {
          isDone = true;
          break;
        }

        chunk += decoder.decode(value, { stream: true });
        // Split by lines to parse SSE
        const lines = chunk.split('\n');
        chunk = lines.pop() || ''; // keep the incomplete line in chunk
        
        for (const line of lines) {
          if (line.startsWith('event: ')) {
            currentEvent = line.replace('event: ', '').trim();
          } else if (line.startsWith('data: ')) {
            const dataStr = line.replace('data: ', '');
            if (dataStr.trim() === '[DONE]') {
              isDone = true;
              break;
            }
            if (currentEvent === 'session') {
              if (this.sessionId !== dataStr.trim()) {
                this.sessionId = dataStr.trim();
                this.updateUrl(this.sessionId);
              }
            } else if (currentEvent === 'message') {
              if (dataStr === '') {
                fullText += '\n';
              } else {
                fullText += dataStr;
              }
              contentEl.textContent = fullText;
              this.scrollToBottom();
            }
          }
        }
      }
    } catch (err) {
      fullText += `\n\n[流中断或请求失败: ${err.message}]`;
    } finally {
      assistantEl.classList.remove('streaming');
      contentEl.innerHTML = marked.parse(fullText);
      this.currentSessionMessages.push({ role: 'assistant', content: fullText });
      this.scrollToBottom();
    }
  }

  // --- AIOps Specific Helpers ---
  startAIOpsLoadingSession() {
    this.sessionId = 'aiops_' + Math.random().toString(36).substring(2, 10) + Date.now().toString(36);
    this.isAIOps = true;
    this.currentSessionMessages = [];
    this.chatMessages.innerHTML = '';
    this.welcomeBox.classList.add('hidden');
    if (this.inputContainer) this.inputContainer.classList.add('hidden');
    this.updateUrl(this.sessionId);
    
    // Create loading bubble
    this.currentAIOpsBubble = this.appendMessageElement('assistant', '正在进行全量告警扫描与知识图谱根因分析，请稍候...', true);
    
    // Generate initial history skeleton
    let session = {
      id: this.sessionId,
      title: 'AIOps报告: ' + new Date().toLocaleString(),
      timestamp: Date.now(),
      messages: [],
      isAIOps: true
    };
    this.chatHistories.unshift(session);
    localStorage.setItem('opspilot_histories', JSON.stringify(this.chatHistories));
    this.renderHistoryList();
  }

  finishAIOpsSession(content, error = null) {
    if (error) {
       content = `<div style="color:#f85149">${error}</div>`;
    }
    
    // Stop loading
    if (this.currentAIOpsBubble) {
      this.currentAIOpsBubble.classList.remove('streaming');
      const contentEl = this.currentAIOpsBubble.querySelector('.message-content');
      contentEl.innerHTML = marked.parse(content);
    } else {
      this.appendMessageElement('assistant', content);
    }
    
    this.currentSessionMessages.push({ role: 'assistant', content });
    this.saveSessionHistory();
  }

  // UI Helpers
  appendMessageElement(role, content, isTyping = false) {
    const wrapper = document.createElement('div');
    wrapper.className = `message ${role} ${isTyping ? 'streaming' : ''}`;
    
    const avatar = document.createElement('div');
    avatar.className = 'avatar';
    avatar.innerHTML = role === 'user' 
      ? `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg>`
      : `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 8V4H8"></path><rect x="4" y="8" width="16" height="12" rx="2"></rect><path d="M2 14h2"></path><path d="M20 14h2"></path><path d="M15 13v2"></path><path d="M9 13v2"></path></svg>`;

    const bubble = document.createElement('div');
    bubble.className = 'message-content markdown-body';
    
    if (isTyping) {
      // empty, streaming cursor handles via css
    } else {
      bubble.innerHTML = marked.parse(content);
    }
    
    wrapper.appendChild(avatar);
    wrapper.appendChild(bubble);
    
    this.chatMessages.appendChild(wrapper);
    this.scrollToBottom();
    
    return wrapper;
  }

  scrollToBottom() {
    this.chatMessages.scrollTop = this.chatMessages.scrollHeight;
  }
}

// Auto init
document.addEventListener('DOMContentLoaded', () => {
  if (window.location.pathname.includes('index.html') || window.location.pathname === '/') {
    window.chatApp = new ChatApp();
  }
});
