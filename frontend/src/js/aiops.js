import api, { showToast } from './api.js';
import { marked } from 'marked';

document.addEventListener('DOMContentLoaded', () => {
  const btn = document.getElementById('aiopsBtn');
  if (!btn) return;

  const runAnalysis = async () => {
    if (!window.chatApp) {
      showToast('聊天系统尚未初始化完成', 'error');
      return;
    }

    // Tell chat app to start a dedicated loading session
    window.chatApp.startAIOpsLoadingSession();

    try {
      const res = await api.post('/ai_ops', {});
      const data = res.data || res;
      
      let resultText = data.result || '暂无运维分析总结';
      // Attempt to parse out the clean response if it's encapsulated in JSON
      try {
        const parsed = JSON.parse(resultText);
        if (parsed && parsed.response) {
          resultText = parsed.response;
        }
      } catch (e) {
        // Keep raw resultText
      }

      const details = data.detail || [];

      // Create Result Markdown String (instead of HTML)
      let markdownStr = `### 💡 分析大模型总结\n\n${resultText}\n\n`;

      if (details.length > 0) {
        markdownStr += `<details>\n<summary><b>🧠 点击展开：大模型推盘思考与执行链路 (${details.length} 个步骤节点)</b></summary>\n\n`;
        details.forEach((detail, index) => {
          markdownStr += `#### 步骤 ${index + 1}\n\`\`\`text\n${detail}\n\`\`\`\n\n`;
        });
        markdownStr += `</details>\n`;
      }

      window.chatApp.finishAIOpsSession(markdownStr, null);
      showToast('AIOps 分析完成', 'success');
    } catch (err) {
      window.chatApp.finishAIOpsSession('', `告警分析失败: ${err.message}`);
    }
  };

  btn.addEventListener('click', runAnalysis);
});
