(function () {
  const themeKey = 'album-course-theme';
  if (localStorage.getItem(themeKey) === 'dark') document.body.classList.add('dark');
  document.querySelectorAll('[data-theme-toggle]').forEach((button) => button.addEventListener('click', () => { document.body.classList.toggle('dark'); localStorage.setItem(themeKey, document.body.classList.contains('dark') ? 'dark' : 'light'); }));
  document.querySelectorAll('.reveal-answer').forEach((button) => button.addEventListener('click', () => { const answer = button.nextElementSibling; const hidden = answer.hasAttribute('hidden'); answer.toggleAttribute('hidden'); button.textContent = hidden ? '收起分析' : '查看分析'; }));
  document.querySelectorAll('.copy-code').forEach((button) => button.addEventListener('click', async () => { await navigator.clipboard.writeText(button.closest('.code-block').querySelector('code').innerText); const label = button.textContent; button.textContent = '已复制'; setTimeout(() => button.textContent = label, 1200); }));

  const chapterButton = document.querySelector('[data-chapter-menu]');
  if (chapterButton) {
    const albumChapters = [
      ['01', '上线问题', 'index.html'], ['02', '表设计评审', 'chapters/chapter-01-database-design.html'],
      ['03', 'Docker 安装 MySQL', 'chapters/chapter-02-docker-mysql.html'], ['04', 'Gin 查询 MySQL', 'chapters/chapter-03-gin-mysql.html'],
      ['05', '接口设计', 'chapters/chapter-04-api-design.html'], ['06', 'JWT 与权限', 'chapters/chapter-auth-jwt-permission.html'],
      ['07', 'OSS 文件架构', 'chapters/chapter-oss-storage.html'], ['08', '前后端联调', 'chapters/chapter-frontend-integration.html'],
      ['09', '正式上线', 'chapters/chapter-production-deployment.html'], ['10', '故障定位', 'chapters/chapter-observability-troubleshooting.html'],
      ['11', '性能优化', 'chapters/chapter-performance-cache.html'], ['12', '项目验收', 'chapters/chapter-final-review.html']
    ];
    const agentChapters = [
      ['01', '大模型、对话与 Agent', 'chapters/chapter-01-llm-agent-react.html'],
      ['02', 'Eino 与 ChatModel', 'chapters/chapter-02-eino-chatmodel.html'],
      ['03', 'Prompt 与 Chain', 'chapters/chapter-03-prompt-chain.html'],
      ['04', 'Tool、ReAct 与会话', 'chapters/chapter-04-tool-agent.html'],
      ['05', 'Skill 与外部 Agent', 'chapters/chapter-05-skill-api.html'],
      ['06', 'Retriever 与 RAG', 'chapters/chapter-06-rag.html'],
      ['07', 'Graph 与 Workflow', 'chapters/chapter-07-graph-workflow.html'],
      ['08', '工具调用的安全与故障处理', 'chapters/chapter-08-agent-product.html'],
      ['09', '测试与验收学业助手', 'chapters/chapter-09-evaluation-contract.html']
    ];
    const kafkaChapters = [
      ['01', '同步订单链路为什么失控', 'index.html'],
      ['02', 'Kafka 如何保存和分发事件', 'chapters/chapter-02-kafka-model.html'],
      ['03', '消息如何按业务分类', 'chapters/chapter-03-topic.html'],
      ['04', '多个消费者如何协作', 'chapters/chapter-04-consumer-groups.html'],
      ['05', '如何提高消费并行度', 'chapters/chapter-05-partition.html'],
      ['06', '同一个订单如何保持顺序', 'chapters/chapter-06-ordering.html'],
      ['07', 'Producer 如何避免消息丢失', 'chapters/chapter-07-producer-reliability.html'],
      ['08', '数据库和 Kafka 如何保持一致', 'chapters/chapter-08-outbox.html'],
      ['09', '重复消费如何处理', 'chapters/chapter-09-idempotency.html'],
      ['10', '重试和死信队列怎么设计', 'chapters/chapter-10-retry-dlq.html'],
      ['11', '库存失败后订单如何补偿', 'chapters/chapter-11-compensation.html'],
      ['A', '启动本地 Kafka', 'chapters/chapter-a-local-kafka.html'],
      ['B', '订单系统该选哪种中间件', 'chapters/chapter-b-middleware.html']
    ];
    const securityChapters = [
      ['01', '功能测通了，为什么系统仍不安全？', 'index.html'],
      ['02', 'SQL 注入', 'chapters/chapter-02-sql-injection.html'],
      ['03', 'XSS', 'chapters/chapter-03-xss.html'],
      ['04', '文件上传', 'chapters/chapter-04-file-upload.html'],
      ['05', '认证与 JWT', 'chapters/chapter-05-auth-jwt.html'],
      ['06', '越权', 'chapters/chapter-06-authorization.html'],
      ['07', '密码、Token 与 Secret', 'chapters/chapter-07-secrets.html'],
      ['08', 'SSRF', 'chapters/chapter-08-ssrf.html'],
      ['09', '安全配置错误', 'chapters/chapter-09-misconfiguration.html'],
      ['10', 'API 安全', 'chapters/chapter-10-api-security.html'],
      ['11', '供应链安全', 'chapters/chapter-11-supply-chain.html'],
      ['12', 'AI 编程与安全责任', 'chapters/chapter-12-ai-security.html'],
      ['13', '综合实战', 'chapters/chapter-13-lab.html']
    ];
    const nodeChapters = [
      ['01', '课程总览', 'index.html'],
      ['02', 'Node.js 是什么', 'chapters/chapter-02-runtime.html'],
      ['03', '第一个 Node 工具', 'chapters/chapter-03-first-script.html'],
      ['04', '模块化：代码多了以后怎么办？', 'chapters/chapter-04-modules.html'],
      ['05', 'package.json：一个项目如何描述自己', 'chapters/chapter-05-package-json.html'],
      ['06', 'npm：别人写好的代码怎么来到我的项目里？', 'chapters/chapter-06-registry.html'],
      ['07', '依赖管理：为什么“能安装”还不够？', 'chapters/chapter-07-dependencies.html'],
      ['08', 'pnpm：为什么还需要另一种包管理器？', 'chapters/chapter-08-pnpm-monorepo.html'],
      ['09', '开发自己的 Package：本地联调', 'chapters/chapter-09-local-registry.html'],
      ['10', '私有 Package Registry', 'chapters/chapter-10-private-registry.html'],
      ['11', 'Workspace 与 Monorepo', 'chapters/chapter-11-cli-service.html'],
      ['12', 'Node.js 工程化', 'chapters/chapter-12-ai-delivery.html'],
      ['13', 'Event Loop 与异步 I/O', 'chapters/chapter-13-event-loop.html'],
      ['14', 'Node.js 网络编程', 'chapters/chapter-14-http-server.html'],
      ['15', 'Koa Web Framework', 'chapters/chapter-15-web-framework.html']
    ];
    const dshPluginChapters = [
      ['HOME', '课程首页', 'index.html'],
      ['01', '插件是什么，什么时候该写插件', 'chapters/chapter-01.html'],
      ['02', '运行环境与七环节地图', 'chapters/chapter-02.html'],
      ['03', '第一个插件', 'chapters/chapter-03.html'],
      ['04', '事件、工具与服务', 'chapters/chapter-04.html'],
      ['05', '配置与组合', 'chapters/chapter-05.html'],
      ['06', '最小 Client 插件', 'chapters/chapter-06.html'],
      ['07', '真实插件代码解读', 'chapters/chapter-07.html'],
      ['08', '调试、打包与分发', 'chapters/chapter-08.html'],
      ['09', '综合项目：工具调用计数器', 'chapters/chapter-09.html'],
      ['A', '附录 A：JS/TS/Node 基础', 'chapters/appendix-a.html']
    ];
    const data = document.body.dataset;
    const chapters = data.course === 'product-design'
      ? window.productDesignChapters
      : data.course === 'dsh-plugin'
      ? dshPluginChapters
      : data.course === 'agent'
      ? agentChapters
      : data.course === 'kafka'
        ? kafkaChapters
        : data.course === 'security'
          ? securityChapters
          : data.course === 'node'
            ? nodeChapters
            : albumChapters;
    const inChapter = location.pathname.includes('/chapters/');
    const rawChapter = String(document.body.dataset.chapter || '01');
    const current = /^\d+$/.test(rawChapter) ? rawChapter.padStart(2, '0') : rawChapter;
    const dropdown = document.createElement('div');
    dropdown.className = 'chapter-dropdown';
    dropdown.hidden = true;
    dropdown.innerHTML = chapters.map(([number, title, href]) => {
      const target = inChapter ? (href === 'index.html' ? '../index.html' : href.replace('chapters/', '')) : href;
      const active = number === current;
      const isPart = /^P\d$/.test(number);
      const label = /^\d+$/.test(number)
        ? (data.course === 'product-design' ? `第 ${Number(number)} 课` : `Chapter ${Number(number)}`)
        : isPart ? `第${'一二三四五'[Number(number.slice(1)) - 1]}部分` : number === 'HOME' ? '课程首页' : data.course === 'product-design' && number === 'SUM' ? '课程总结' : `Appendix ${number}`;
      const classes = [active ? 'active' : '', isPart ? 'part' : ''].filter(Boolean).join(' ');
      return `<a class="${classes}" href="${target}"><span>${label}</span><strong>${title}</strong></a>`;
    }).join('');
    chapterButton.insertAdjacentElement('afterend', dropdown);
    const closeMenu = () => {
      dropdown.hidden = true;
      chapterButton.setAttribute('aria-expanded', 'false');
    };
    const openMenu = () => {
      dropdown.hidden = false;
      chapterButton.setAttribute('aria-expanded', 'true');
    };
    chapterButton.addEventListener('click', (event) => {
      event.stopPropagation();
      dropdown.hidden ? openMenu() : closeMenu();
    });
    dropdown.addEventListener('click', (event) => event.stopPropagation());
    document.addEventListener('click', closeMenu);
    document.addEventListener('keydown', (event) => { if (event.key === 'Escape') closeMenu(); });
  }

  // ---- 播放模式：全屏、按小节一页一页翻，页内可滚动 ----
  // 切片规则：内容容器的顶层 header / section / div / article 各算一页。
  //   · 章节页的容器是 article.chapter；课程首页、部分导览页没有 article，
  //     容器退到 <main>（.launch-main / .home / .part-main ...）。这些页面同样
  //     需要「播放」：上课时把首页或部分导览页投出来讲，是最常见的用法。
  //   · 平台首页是目录页，不是课件页，整页退出：<body data-no-play>。
  //   · 用顶层子元素而不是 <section>，是为了兼容历史页面——fullstack-album
  //     第 02 章整页没有 section（内容是 header + 一个 div），按 section 切会得到 0 页。
  //   · footer.chapter-footer 是前后翻章导航、nav.part-progress 是部分导航，都排除；
  //     零散的顶层 <p> 之类也排除，它们不是内容块，单独成页会很难看
  //     （播放模式下这些元素不显示）。
  // 开关与控制条都是运行时注入，不改任何页面 HTML——AI Native 的部分导览页由
  // sync_ai_native_meta.py 生成，手工加按钮会被下一次同步覆盖。
  (function () {
    // 用 hasAttribute 而不是 dataset.noPlay：写成 <body data-no-play>（无值）时
    // dataset.noPlay 是空串，空串为假，判断会失效。
    if (document.body.hasAttribute('data-no-play')) return;

    const SLIDE_TAGS = ['HEADER', 'SECTION', 'DIV', 'ARTICLE'];
    const NON_SLIDES = ['chapter-footer', 'part-progress'];
    // 从上到下第一个能切出 ≥2 页的容器。章节页优先，避免 <main> 抢走 article.chapter。
    const CONTAINERS = ['article.chapter', 'main.part-main', 'main.home', 'main.launch-main', 'main'];

    const slidesOf = (root) => Array.prototype.filter.call(root.children, (el) => {
      if (SLIDE_TAGS.indexOf(el.tagName) < 0) return false;
      return !NON_SLIDES.some((cls) => el.classList.contains(cls));
    });

    let article = null;
    let slides = [];
    for (const selector of CONTAINERS) {
      const candidate = document.querySelector(selector);
      if (!candidate) continue;
      const found = slidesOf(candidate);
      if (found.length >= 2) { article = candidate; slides = found; break; }
    }
    if (!article) return; // 切不出页就别给入口
    const actions = document.querySelector('.topbar-actions');

    // ---- 连续播放：最后一屏再往前，直接进下一章，而不是卡在原地 ----
    // 「下一站」按实际约定逐级找：多数页脚用 a.next-question（73 页），少数用
    // a.primary（27 页）；课程首页没有页脚，退回容器里的主按钮（「从第一部分开始 →」
    // 「进入 Chapter 1 →」）。最后复习章页脚里没有下一个链接，找不到就停在末屏。
    // 「上一站」取页脚里既不是 primary 也不是 next-question 的那个链接。
    // 用 querySelector 而不是 querySelectorAll：选择器集中在这里，测试的 DOM 桩好对齐。
    const anchorHref = (a) => (a && a.getAttribute('href')) || null;
    const nextHref = anchorHref(document.querySelector('.chapter-footer a.primary'))
      || anchorHref(document.querySelector('.chapter-footer a.next-question'))
      || anchorHref(document.querySelector('link[rel="next"]'))
      || anchorHref(article.querySelector('a.button.primary'));
    const prevHref = anchorHref(document.querySelector('.chapter-footer a:not(.primary):not(.next-question)'))
      || anchorHref(document.querySelector('link[rel="prev"]'));

    // 翻过去的页面要自动接着播：临时标记写在 sessionStorage（关掉标签页就没了），
    // 'last' 表示从后往前翻，落点应当是上一页的最后一屏。
    const AUTO_KEY = 'album-course-autoplay';
    const store = {
      get() { try { return typeof sessionStorage === 'undefined' ? null : sessionStorage.getItem(AUTO_KEY); } catch (e) { return null; } },
      set(v) { try { if (typeof sessionStorage !== 'undefined') sessionStorage.setItem(AUTO_KEY, v); } catch (e) { /* 隐私模式忽略 */ } },
      clear() { try { if (typeof sessionStorage !== 'undefined') sessionStorage.removeItem(AUTO_KEY); } catch (e) { /* 忽略 */ } },
    };

    const zoomKey = 'album-course-zoom';
    const zoomLabels = { 1: '标准', 2: '较大', 3: '最大' };
    const zoomMax = 3;

    const toggle = document.createElement('button');
    toggle.type = 'button';
    toggle.className = actions ? 'play-toggle' : 'play-toggle play-toggle-float';
    toggle.textContent = '播放';
    toggle.title = '播放（快捷键 F）：全屏、按小节翻页';

    const bar = document.createElement('div');
    bar.className = 'play-bar';
    bar.innerHTML =
      '<span class="play-bar-title"></span>'
      + '<span class="play-bar-section"></span>'
      + '<span class="play-bar-count"></span>'
      + '<span class="play-bar-tools">'
      + '<span class="play-bar-hint">← ↑ ↓ → / 点击 / 滚轮翻页 · F 全屏 · Esc 退出</span>'
      + '<button type="button" data-play-zoom="-1" aria-label="缩小正文">−</button>'
      + '<span data-play-zoom-label></span>'
      + '<button type="button" data-play-zoom="1" aria-label="放大正文">+</button>'
      + '<button type="button" data-play-prev aria-label="上一页">←</button>'
      + '<button type="button" data-play-next aria-label="下一页">→</button>'
      + '<button type="button" data-play-exit>退出</button>'
      + '</span>';

    // 投屏播放时顶部工具条整条隐藏（标题、计数、按钮对观众都是噪音），只留最上面
    // 一条 2px 进度线：讲的人知道讲到哪了，观众几乎看不见。控制条仍留在 DOM 里，
    // 按键（← → 回车 / +/- / F / Esc）和校验脚本照旧能找到它。
    const progress = document.createElement('div');
    progress.className = 'play-progress';
    progress.innerHTML = '<i class="play-progress-fill"></i>';

    const titleEl = bar.querySelector('.play-bar-title');
    const sectionEl = bar.querySelector('.play-bar-section');
    const countEl = bar.querySelector('.play-bar-count');
    const progressEl = progress.querySelector('.play-progress-fill');
    const zoomLabel = bar.querySelector('[data-play-zoom-label]');
    const zoomOut = bar.querySelector('[data-play-zoom="-1"]');
    const zoomIn = bar.querySelector('[data-play-zoom="1"]');
    const prevBtn = bar.querySelector('[data-play-prev]');
    const nextBtn = bar.querySelector('[data-play-next]');
    const exitBtn = bar.querySelector('[data-play-exit]');

    // 鼠标翻页：画面左右两侧常驻一对淡箭头（悬停变亮）。
    // 另外点击正文任意处也能前进，见下面的 document click 处理。
    const edgePrev = document.createElement('button');
    edgePrev.type = 'button';
    edgePrev.className = 'play-edge play-edge-prev';
    edgePrev.textContent = '‹';
    edgePrev.setAttribute('aria-label', '上一页');
    const edgeNext = document.createElement('button');
    edgeNext.type = 'button';
    edgeNext.className = 'play-edge play-edge-next';
    edgeNext.textContent = '›';
    edgeNext.setAttribute('aria-label', '下一页');

    // 章节页、首页、部分页都用同一套样式，靠这个类把「内容容器」标出来
    article.classList.add('play-root');
    if (actions) actions.append(toggle);
    document.body.append(bar, progress, edgePrev, edgeNext);
    // 没有顶栏的页面（例如整页只有 main 的落地页）把开关挂在右上角
    if (!actions) document.body.append(toggle);

    const heading = article.querySelector('.chapter-header h1') || article.querySelector('h1');
    const chapterTitle = ((heading || {}).textContent || document.title || '').trim();

    const stored = parseInt(localStorage.getItem(zoomKey), 10);
    let zoom = Math.min(zoomMax, Math.max(1, isNaN(stored) ? 1 : stored));
    let index = 0;
    let askedFullscreen = false;
    // 滚轮翻页：只在页内已经滚到边界时才生效，所以要攒够位移、翻完还要有一段冷却，
    // 否则触控板的惯性滚动会一次滑过好几页。
    let wheelAccum = 0;
    let wheelLockUntil = 0;

    const isPlaying = () => document.body.classList.contains('play-mode');

    // 每页顶部的标题：优先小节自己的 h2，没有就用 kicker 文字。
    // 首页 / 部分导览页的第一屏只有 h1；章节页的第一屏 h1 是章标题（控制条左侧已经
    // 显示了），所以只有非章节页才把 h1 当小节标题，免得同一行重复两遍。
    const isChapterPage = article.tagName === 'ARTICLE';
    function slideLabel(el) {
      const title = el.querySelector('h2') || (!isChapterPage && el.querySelector('h1'));
      if (title) return title.textContent.trim();
      const kicker = el.querySelector('.section-kicker');
      if (kicker) return kicker.textContent.trim();
      if (el !== slides[0]) return '';
      return isChapterPage ? '本章导览' : '本页导览';
    }

    function renderZoom() {
      document.body.dataset.playZoom = String(zoom);
      zoomLabel.textContent = zoomLabels[zoom];
      zoomOut.disabled = zoom <= 1;
      zoomIn.disabled = zoom >= zoomMax;
      localStorage.setItem(zoomKey, String(zoom));
    }

    function render() {
      slides.forEach((el, i) => el.classList.toggle('play-current', i === index));
      const current = slides[index];
      const atEnd = index === slides.length - 1;
      const atStart = index === 0;
      titleEl.textContent = chapterTitle;
      sectionEl.textContent = slideLabel(current);
      countEl.textContent = (index + 1) + ' / ' + slides.length;
      progressEl.style.width = ((index + 1) / slides.length * 100) + '%';
      // 有下一章时末屏不置灰：按钮变成「下一章」，按下去就翻页
      prevBtn.disabled = atStart && !prevHref;
      nextBtn.disabled = atEnd && !nextHref;
      edgePrev.disabled = atStart && !prevHref;
      edgeNext.disabled = atEnd && !nextHref;
      prevBtn.textContent = atStart && prevHref ? '← 上一章' : '←';
      nextBtn.textContent = atEnd && nextHref ? '下一章 →' : '→';
      // 同步 URL，退出播放后锚点仍然可用、可直接分享。
      // 用 try 包住：以 file:// 打开时部分浏览器会对 replaceState 抛 SecurityError。
      const id = current.id;
      if (id && history.replaceState) {
        try { history.replaceState(null, '', '#' + id); } catch (e) { /* file:// 下忽略 */ }
      }
    }

    function go(next) {
      const clamped = Math.min(slides.length - 1, Math.max(0, next));
      if (clamped === index && isPlaying()) return;
      index = clamped;
      wheelAccum = 0; // 换页后清掉滚轮余量，免得紧接着又滑一页
      render();
      if (isPlaying()) slides[index].scrollTop = 0; // 新的一页从顶部开始
    }

    // 翻到别的页面继续播：标记落点，让目标页自己接着播
    function leaveTo(href, where) {
      if (!href) return;
      store.set(where);
      location.href = href;
    }

    // 前进/后退：本页还有屏就翻屏，到头了就翻页。两个方向对称。
    function forward() {
      if (index < slides.length - 1) go(index + 1);
      else leaveTo(nextHref, 'first');
    }

    function backward() {
      if (index > 0) go(index - 1);
      else leaveTo(prevHref, 'last');
    }

    function changeZoom(delta) {
      const next = Math.min(zoomMax, Math.max(1, zoom + delta));
      if (next === zoom) return;
      zoom = next;
      renderZoom();
    }

    function start(atIndex) {
      if (typeof atIndex === 'number') {
        // 连续播放翻过来时由上一页指定落点（首页第一屏 / 末页最后一屏）
        index = Math.min(slides.length - 1, Math.max(0, atIndex));
      } else {
        // 从地址栏锚点进入对应的一页，方便直接跳到某一节。
        // 没有锚点时必须跳过匹配：空串会等于「第一个没有 id 的小节」，那样点播放
        // 会从第 2 屏开始（fullstack-album 的章节就是这种结构）。
        const hash = (location.hash || '').slice(1);
        const at = hash ? slides.findIndex((el) => el.id === hash) : -1;
        index = at >= 0 ? at : 0;
      }
      document.body.classList.add('play-mode');
      toggle.setAttribute('aria-pressed', 'true');
      renderZoom();
      render();
      slides[index].scrollTop = 0;
      // 全屏要用户手势，这里正是点击/按键触发；被拒也不影响播放，只是不全屏
      askedFullscreen = true;
      requestFullscreen();
    }

    // 全屏可能被拒（自动续播没有用户手势），失败就只在非全屏下继续播
    function requestFullscreen() {
      if (!document.documentElement.requestFullscreen) return;
      const p = document.documentElement.requestFullscreen();
      if (p && p.catch) p.catch(() => {});
    }

    function stop() {
      const current = slides[index];
      document.body.classList.remove('play-mode');
      toggle.setAttribute('aria-pressed', 'false');
      slides.forEach((el) => { el.classList.remove('play-current'); el.scrollTop = 0; });
      if (askedFullscreen && document.fullscreenElement && document.exitFullscreen) {
        const p = document.exitFullscreen();
        if (p && p.catch) p.catch(() => {});
      }
      askedFullscreen = false;
      // 回到普通视图后停在同一节，不让人丢失位置
      if (current && current.scrollIntoView) current.scrollIntoView();
    }

    const setPlaying = (on) => (on ? start() : stop());

    toggle.addEventListener('click', () => setPlaying(!isPlaying()));
    exitBtn.addEventListener('click', () => stop());
    prevBtn.addEventListener('click', backward);
    nextBtn.addEventListener('click', forward);
    zoomIn.addEventListener('click', () => changeZoom(1));
    zoomOut.addEventListener('click', () => changeZoom(-1));
    edgePrev.addEventListener('click', backward);
    edgeNext.addEventListener('click', forward);
    bar.addEventListener('click', (event) => event.stopPropagation());

    // 鼠标翻页：点击正文前进；点最左侧一条窄带后退（避免误点一下就往回跳）。
    // 链接、按钮、代码复制、展开答案这些交互元素上的点击保持原意；
    // 正在选文字时也不翻页，否则没法选中段落。
    document.addEventListener('click', (event) => {
      if (!isPlaying()) return;
      const target = event.target;
      if (target && target.closest
        && target.closest('a,button,input,textarea,select,summary,label,[contenteditable]')) return;
      const selection = typeof window !== 'undefined' && window.getSelection ? window.getSelection() : null;
      if (selection && String(selection).length) return;
      const width = (document.documentElement && document.documentElement.clientWidth) || 0;
      if (width && event.clientX < width * 0.15) backward();
      else forward();
    });

    // 滚轮翻页：页内还滚得动就交给浏览器滚，滚到边界之后再滚才翻页。
    // 这样长小节照样能用滚轮读，短小节一滑就翻，两种页面都不别扭。
    document.addEventListener('wheel', (event) => {
      if (!isPlaying()) return;
      if (event.ctrlKey || event.metaKey) return; // 双指缩放手势不当翻页
      const slide = slides[index];
      if (!slide) return;
      // deltaMode：0=像素 1=行 2=页。鼠标滚轮常按行上报，不折算永远攒不够阈值。
      const unit = event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? 100 : 1;
      const delta = event.deltaY * unit;
      if (!delta) return;
      const down = delta > 0;
      const scrollable = slide.scrollHeight > slide.clientHeight + 1;
      const atTop = slide.scrollTop <= 0;
      const atBottom = slide.scrollTop + slide.clientHeight >= slide.scrollHeight - 1;
      if (scrollable && ((down && !atBottom) || (!down && !atTop))) {
        wheelAccum = 0; // 页内还有内容可滚，让浏览器自己滚
        return;
      }
      const now = Date.now();
      if (now < wheelLockUntil) return; // 刚翻过页，惯性滚动还没停
      wheelAccum += delta;
      if (Math.abs(wheelAccum) < 100) return;
      wheelAccum = 0;
      const target = down ? index + 1 : index - 1;
      if (target < 0 || target >= slides.length) {
        // 滚到头了，和点「下一章」一样翻页继续播（翻一次后 800ms 内不再触发）
        if (now < wheelLockUntil) return;
        wheelLockUntil = now + 800;
        if (down) leaveTo(nextHref, 'first'); else leaveTo(prevHref, 'last');
        return;
      }
      wheelLockUntil = now + 700;
      go(target);
    }, { passive: true });

    // 全屏状态下按 Esc 由浏览器接管、可能收不到 keydown，所以在这里兜底同步状态
    document.addEventListener('fullscreenchange', () => {
      if (!document.fullscreenElement && isPlaying()) stop();
    });

    document.addEventListener('keydown', (event) => {
      if (event.key === 'Escape') {
        if (!isPlaying()) return;
        // 章节下拉菜单开着时，这一次 Esc 让给它（common.js 上面已有关菜单的处理）
        if (document.querySelector('.chapter-dropdown:not([hidden])')) return;
        stop();
        return;
      }
      if (event.ctrlKey || event.metaKey || event.altKey) return;
      const el = document.activeElement;
      if (el && (/^(INPUT|TEXTAREA|SELECT)$/.test(el.tagName) || el.isContentEditable)) return;

      if (!isPlaying()) {
        if (event.key === 'f' || event.key === 'F') { event.preventDefault(); start(); }
        return;
      }
      switch (event.key) {
        case 'ArrowRight': case 'ArrowDown': case 'PageDown': case 'Enter': case ' ': case 'Spacebar':
          event.preventDefault(); forward(); break;
        case 'ArrowLeft': case 'ArrowUp': case 'PageUp':
          event.preventDefault(); backward(); break;
        case 'Home': event.preventDefault(); go(0); break;
        case 'End': event.preventDefault(); go(slides.length - 1); break;
        case '+': case '=': event.preventDefault(); changeZoom(1); break;
        case '-': case '_': event.preventDefault(); changeZoom(-1); break;
        case 'f': case 'F':
          // 自动续播翻过来的页面拿不到用户手势，只能非全屏播；F 先用来补全屏，
          // 已经在全屏了才当退出用（Esc 与「退出」按钮同样是退出）。
          event.preventDefault();
          if (document.fullscreenElement) stop();
          else { askedFullscreen = true; requestFullscreen(); }
          break;
        default: break;
      }
    });

    // 上一页是翻到最后一屏继续往前走的，这里自动接着播。
    // 落点：往前翻 = 第一屏，往回翻 = 最后一屏。
    const resume = store.get();
    if (resume) {
      store.clear();
      start(resume === 'last' ? slides.length - 1 : 0);
    }
  })();
})();
