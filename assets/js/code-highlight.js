(function () {
  const extensionMap = {
    go: 'go',
    sql: 'sql',
    json: 'json',
    js: 'javascript',
    jsx: 'jsx',
    ts: 'typescript',
    tsx: 'typescript',
    http: 'http',
    sh: 'bash',
    bash: 'bash',
    conf: 'nginx',
    env: 'bash',
    yaml: 'yaml',
    yml: 'yaml',
    md: 'markdown',
    dockerfile: 'docker'
  };

  const normalizeLanguage = (value) => {
    const key = String(value || '').toLowerCase().trim();
    if (!key) return '';
    if (key.includes('dockerfile')) return 'docker';
    if (key.includes('terminal') || key.includes('shell') || key.includes('bash') || key.endsWith('.sh')) return 'bash';
    if (key.includes('nginx')) return 'nginx';
    if (key.includes('mysql')) return 'sql';
    if (key.includes('json') || key.includes('response')) return 'json';
    if (key.includes('gorm') || key.includes('.go')) return 'go';
    const extension = key.split(/[\\/]/).pop().split('.').pop();
    return extensionMap[extension] || extensionMap[key] || '';
  };

  const detectByContent = (text) => {
    const source = text.trim();
    if (!source) return '';
    if (/^(\{|\[)[\s\S]*(\}|\])$/.test(source)) return 'json';
    if (/^(SELECT|CREATE|ALTER|INSERT|UPDATE|DELETE|SHOW|USE)\b/im.test(source)) return 'sql';
    if (/^(docker|go get|npm|curl|cd|mkdir|cp|export)\b/im.test(source) || source.includes('docker run')) return 'bash';
    if (/^(package|func|type|var|const|import)\b/m.test(source)) return 'go';
    if (/(^|\n)\s*(db|gormDB|result)\.|:=/.test(source)) return 'go';
    if (/^(GET|POST|PUT|PATCH|DELETE)\s+\/\S+/m.test(source)) return 'http';
    if (/^(server|location)\s*\{/m.test(source)) return 'nginx';
    if (/^(FROM|COPY|RUN|CMD|EXPOSE|ENTRYPOINT|WORKDIR)\b/m.test(source)) return 'docker';
    if (/(async function|const |let |=>|import .* from)/.test(source)) return 'javascript';
    if (/^(\d+\.|[-*])\s+/m.test(source)) return 'markdown';
    return 'none';
  };

  const highlightChildLines = (code, grammar, language) => {
    Array.from(code.childNodes).forEach((node) => {
      const source = node.textContent || '';
      if (!source.trim()) return;

      const highlighted = window.Prism.highlight(source, grammar, language);
      if (node.nodeType === Node.ELEMENT_NODE) {
        node.innerHTML = highlighted;
        return;
      }

      const template = document.createElement('template');
      template.innerHTML = highlighted;
      node.replaceWith(template.content);
    });
  };

  const highlightCodeBlock = (code) => {
    if (code.dataset.courseHighlighted === 'true') return;

    const block = code.closest('.code-block');
    const title = block?.querySelector('.code-header span')?.textContent || '';
    const language = normalizeLanguage(title) || detectByContent(code.textContent || '');
    if (!language || language === 'none') return;

    const grammar = window.Prism?.languages?.[language];

    code.classList.add(`language-${language}`);
    code.parentElement.classList.add(`language-${language}`);

    if (!grammar) {
      code.dataset.courseHighlighted = 'true';
      return;
    }

    if (code.children.length > 0) {
      highlightChildLines(code, grammar, language);
      code.dataset.courseHighlighted = 'true';
      return;
    }

    window.Prism.highlightElement(code);
    code.dataset.courseHighlighted = 'true';
  };

  const highlightCodeBlocks = (root = document) => {
    root.querySelectorAll?.('.code-block pre code').forEach(highlightCodeBlock);
  };

  window.highlightCourseCodeBlocks = highlightCodeBlocks;
  highlightCodeBlocks();

  new MutationObserver((mutations) => {
    mutations.forEach((mutation) => {
      mutation.addedNodes.forEach((node) => {
        if (node.nodeType !== Node.ELEMENT_NODE) return;
        if (node.matches?.('.code-block pre code')) {
          highlightCodeBlock(node);
          return;
        }
        highlightCodeBlocks(node);
      });
    });
  }).observe(document.body, { childList: true, subtree: true });
})();
