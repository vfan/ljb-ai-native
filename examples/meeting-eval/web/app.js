const state = {
  boot: null,
  view: 'run',
  compare: null,
  selection: { baseline: '', candidate: '' },
  filter: '全部',
  renderToken: 0,
  polling: false,
};

const $ = (selector) => document.querySelector(selector);
const VIEW_META = {
  run: { eyebrow: 'RUN', title: '运行评测', sub: '固定模型、参数与测试集，只切换提示词版本。' },
  compare: { eyebrow: 'COMPARE', title: '版本对比', sub: '同时看收益、退步和硬性错误，而不是只看总分。' },
  dataset: { eyebrow: 'DATASET', title: '测试集', sub: '每条样本都写清预期行为与不能接受的错误。' },
  gate: { eyebrow: 'GATE', title: '发布门禁', sub: '通过率达标不等于可以发布，回归需要人工确认。' },
};

const esc = (value) => String(value ?? '').replace(/[&<>"']/g, (char) => ({
  '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
}[char]));

const pct = (value, digits = 0) => `${(Number(value || 0) * 100).toFixed(digits)}%`;
const num = (value) => new Intl.NumberFormat('zh-CN').format(Number(value || 0));
const money = (value) => (value > 0 ? `¥${Number(value).toFixed(4)}` : '—');
const ms = (value) => (value > 0 ? `${num(value)} ms` : '—');
const shortTime = (iso) => {
  if (!iso) return '—';
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
};

function delta(value, suffix = 'pt', digits = 1) {
  const rounded = Number(value || 0);
  if (Math.abs(rounded) < 0.0005) return '<span class="delta flat">持平</span>';
  const cls = rounded > 0 ? 'up' : 'down';
  const sign = rounded > 0 ? '+' : '';
  return `<span class="delta ${cls}">${sign}${(rounded * 100).toFixed(digits)}${suffix}</span>`;
}

function deltaCount(value, suffix = ' 条') {
  const rounded = Number(value || 0);
  if (rounded === 0) return '<span class="delta flat">持平</span>';
  return `<span class="delta ${rounded > 0 ? 'down' : 'up'}">${rounded > 0 ? '+' : ''}${rounded}${suffix}</span>`;
}

async function api(path, options = {}) {
  const init = { method: options.method || 'GET', headers: {} };
  if (options.body !== undefined) {
    init.headers['Content-Type'] = 'application/json';
    init.body = JSON.stringify(options.body);
  }
  const response = await fetch(path, init);
  let payload = {};
  try { payload = await response.json(); } catch (error) { payload = {}; }
  if (!response.ok) throw new Error(payload.error || `请求失败（${response.status}）`);
  return payload;
}

function toast(message, isError = false) {
  const node = $('#toast');
  node.textContent = message;
  node.className = `toast${isError ? ' error' : ''}`;
  node.hidden = false;
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => { node.hidden = true; }, 3800);
}

function promptName(id) {
  const prompt = (state.boot?.prompts || []).find((item) => item.id === id);
  return prompt ? prompt.name : id;
}

function doneRuns() {
  return (state.boot?.runs || [])
    .filter((item) => item.run.status === 'done')
    .sort((a, b) => `${b.run.created_at}${b.run.id}`.localeCompare(`${a.run.created_at}${a.run.id}`));
}

// 默认拿最新的候选，再找最近一次“不同提示词”的运行作基线，
// 避免连续跑两次同一版本时把自己和自己比。
function defaultPair() {
  const completed = doneRuns();
  if (completed.length === 0) return null;
  const candidate = completed[0];
  const baseline = completed.find((item) => item.run.prompt_id !== candidate.run.prompt_id)
    || completed[1]
    || candidate;
  return { candidate, baseline };
}

function verdictPill(pass) {
  return pass ? '<span class="pill pass">通过</span>' : '<span class="pill fail">失败</span>';
}

function transitionPill(transition) {
  if (transition === 'fixed') return '<span class="pill fixed">修复</span>';
  if (transition === 'regressed') return '<span class="pill regressed">退步</span>';
  if (transition === 'both_pass') return '<span class="pill muted">保持通过</span>';
  return '<span class="pill muted">仍然失败</span>';
}

function findingChips(findings, limit = 2) {
  const seen = new Set();
  const chips = [];
  for (const finding of findings || []) {
    if (seen.has(finding.type)) continue;
    seen.add(finding.type);
    const label = state.boot?.labels?.hard?.[finding.type] || state.boot?.labels?.soft?.[finding.type] || finding.type;
    const soft = !state.boot?.labels?.hard?.[finding.type];
    chips.push(`<span class="f-chip${soft ? ' soft' : ''}" title="${esc(finding.message)}">${esc(label)}</span>`);
    if (chips.length >= limit) break;
  }
  return chips.join('');
}

function statusText(value) {
  return { confirmed: '已确认', pending: '待确认', cancelled: '已取消' }[value] || value;
}

function statusPill(value) {
  const cls = { confirmed: 'pass', pending: 'review', cancelled: 'muted' }[value] || 'muted';
  return `<span class="pill ${cls}">${esc(statusText(value))}</span>`;
}

function highlight(text, quotes) {
  const unique = [...new Set((quotes || []).filter((quote) => quote && text.includes(quote)))]
    .sort((a, b) => b.length - a.length);
  const picked = [];
  for (const quote of unique) {
    if (!picked.some((existing) => existing.includes(quote))) picked.push(quote);
  }
  let html = esc(text);
  for (const quote of picked) {
    const safe = esc(quote);
    html = html.split(safe).join(`<mark>${safe}</mark>`);
  }
  return html;
}

function skeleton(lines = 3) {
  return `<div class="panel"><div class="empty">加载中…</div></div>`;
}

/* ---------------- routing ---------------- */

function route() {
  const hash = location.hash.replace(/^#\/?/, '');
  state.view = Object.keys(VIEW_META).includes(hash) ? hash : 'run';
  document.querySelectorAll('.nav a').forEach((link) => {
    link.classList.toggle('active', link.dataset.view === state.view);
  });
  render();
}

async function render() {
  clearTimeout(state.pollTimer);
  const meta = VIEW_META[state.view];
  $('#pageEyebrow').textContent = meta.eyebrow;
  $('#pageTitle').textContent = meta.title;
  $('#pageSub').textContent = meta.sub;

  const token = ++state.renderToken;
  if (state.view === 'run') await renderRun(token);
  if (state.view === 'compare') await renderCompare(token);
  if (state.view === 'dataset') renderDataset(token);
  if (state.view === 'gate') await renderGate(token);
}

/* ---------------- run ---------------- */

async function renderRun(token) {
  const prompts = state.boot.prompts;
  const datasets = state.boot.datasets;
  const runs = state.boot.runs;

  const runRows = runs.map((item) => {
    const run = item.run;
    const statePill = run.status === 'done'
      ? '<span class="pill muted">完成</span>'
      : `<span class="pill ${run.status === 'error' ? 'fail' : 'muted'}">${run.status === 'running' ? '运行中' : run.status === 'queued' ? '排队' : '出错'}</span>`;
    return `
      <tr class="${run.status === 'done' ? 'clickable' : ''}" data-run="${esc(run.id)}">
        <td><strong>${esc(promptName(run.prompt_id))}</strong><div class="hint mono">${esc(run.mode === 'live' ? 'live' : 'replay')}</div></td>
        <td class="num">${run.status === 'done' ? pct(item.pass_rate, 0) : '—'}</td>
        <td class="num">${run.status === 'done' ? item.hard_fail_cases : '—'}</td>
        <td class="num">${run.status === 'done' ? item.p95_latency : '—'}</td>
        <td>${statePill}</td>
        <td class="hint">${esc(shortTime(run.created_at))}</td>
      </tr>`;
  }).join('');

  let guideDismissed = false;
  try { guideDismissed = localStorage.getItem('meeting-eval-guide') === 'dismissed'; } catch (error) { guideDismissed = false; }
  const showGuide = !guideDismissed && doneRuns().length < 2;

  const body = `
    ${showGuide ? `
      <section class="guide" id="runGuide">
        <div class="guide-steps">
          <div class="guide-step">
            <span class="num">1</span>
            <div><strong>选一版提示词</strong><small>v1 是初版，v2 补了判定规则；点“查看提示词”看全文</small></div>
          </div>
          <div class="guide-step">
            <span class="num">2</span>
            <div><strong>跑同一套边界样本</strong><small>模型、温度、测试集都不变，只换提示词</small></div>
          </div>
          <div class="guide-step">
            <span class="num">3</span>
            <div><strong>看对比和门禁</strong><small>修复多少、有没有退步、能不能发布</small></div>
          </div>
        </div>
        <button class="guide-close" id="dismissGuide" aria-label="收起新手引导">×</button>
      </section>` : ''}
    <div class="run-layout">
      <section class="panel run-form">
        <div class="panel-head">
          <h2>新建评测</h2>
          <span class="hint">一次只改一个变量</span>
        </div>
        <p class="lede">模型、温度、样本与评分器保持一致，只替换提示词版本，收益才能归因。</p>

        <div class="prompt-options">
          ${prompts.map((prompt, index) => `
            <div class="prompt-option">
              <label class="prompt-choice">
                <input type="radio" name="prompt" value="${esc(prompt.id)}" ${index === prompts.length - 1 ? 'checked' : ''}>
                <span class="dot"></span>
                <span class="prompt-meta">
                  <strong>${esc(prompt.name)} <span class="hint mono">v${esc(prompt.version)}</span></strong>
                  <small>${esc(prompt.notes)}</small>
                </span>
              </label>
              <button type="button" class="link-btn prompt-view" data-prompt="${esc(prompt.id)}">查看提示词</button>
            </div>`).join('')}
        </div>

        <div class="form-row">
          <label class="field">测试集
            <select id="datasetSelect">
              ${datasets.map((dataset) => `<option value="${esc(dataset.id)}">${esc(dataset.name)} · ${dataset.case_count} 条</option>`).join('')}
            </select>
          </label>
          <label class="field">重复次数
            <select id="repeatsSelect">
              <option value="1">1 次</option>
              <option value="2">2 次</option>
              <option value="3">3 次</option>
            </select>
          </label>
        </div>

        <div class="run-actions">
          <button class="btn primary" id="startRun">开始运行</button>
          <span class="hint" id="runModeHint">${state.boot.mode === 'replay' ? '回放模式：输出为预置样本' : '真实 API：将产生费用'}</span>
        </div>
        ${doneRuns().length === 0 ? '<p class="run-tip">建议顺序：先跑 v1 建立基线，再跑 v2 看改进。</p>' : ''}

        <div class="progress" id="progress" hidden>
          <div class="progress-head"><span id="progressLabel">准备中</span><span id="progressCount">0 / 0</span></div>
          <div class="progress-track"><div class="progress-fill" id="progressFill"></div></div>
          <p class="progress-note" id="progressNote"></p>
        </div>

        ${state.boot.mode === 'replay' ? `
          <p class="progress-note" style="margin-top:16px">
            想接真实模型：在项目根目录写 <span class="mono">.env</span>，设置
            <span class="mono">DEEPSEEK_API_KEY=…</span> 后重启服务。
          </p>` : ''}
      </section>

      <section class="panel">
        <div class="panel-head">
          <h2>运行记录</h2>
          <span class="hint">点一行进入对比</span>
        </div>
        ${runs.length === 0 ? `
          <div class="empty"><strong>还没有运行记录</strong>先跑一次，再跑第二次才能比较版本。</div>
        ` : `
          <div class="table-wrap">
            <table>
              <thead><tr><th>提示词</th><th class="num">通过率</th><th class="num">硬性错误</th><th class="num">P95</th><th>状态</th><th>时间</th></tr></thead>
              <tbody>${runRows}</tbody>
            </table>
          </div>`}
      </section>
    </div>`;

  if (token !== state.renderToken) return;
  $('#view').innerHTML = body;

  $('#startRun').addEventListener('click', startRun);
  document.querySelectorAll('.prompt-view').forEach((button) => {
    button.addEventListener('click', () => openPromptDrawer(button.dataset.prompt));
  });
  const dismissGuide = $('#dismissGuide');
  if (dismissGuide) {
    dismissGuide.addEventListener('click', () => {
      try { localStorage.setItem('meeting-eval-guide', 'dismissed'); } catch (error) { /* 隐私模式下忽略 */ }
      dismissGuide.closest('.guide').remove();
    });
  }
  document.querySelectorAll('tr[data-run]').forEach((row) => {
    row.addEventListener('click', () => {
      const completed = doneRuns();
      const index = completed.findIndex((item) => item.run.id === row.dataset.run);
      if (index < 0) return;
      const candidate = completed[index];
      const baseline = completed.slice(index + 1).find((item) => item.run.prompt_id !== candidate.run.prompt_id)
        || completed[index + 1]
        || candidate;
      state.selection = { baseline: baseline.run.id, candidate: candidate.run.id };
      location.hash = '#/compare';
    });
  });
}

async function startRun() {
  if (state.polling) return;
  const promptId = document.querySelector('input[name="prompt"]:checked')?.value;
  const datasetId = $('#datasetSelect').value;
  const repeats = Number($('#repeatsSelect').value || 1);
  if (!promptId) return;

  const button = $('#startRun');
  const progress = $('#progress');
  button.disabled = true;
  button.textContent = '运行中…';
  progress.hidden = false;
  $('#progressLabel').textContent = '正在运行';
  $('#progressNote').textContent = '每条样本依次调用模型、规则评分与裁判，完成后自动跳转对比。';
  state.polling = true;

  try {
    const { run } = await api('/api/runs', { method: 'POST', body: { prompt_id: promptId, dataset_id: datasetId, repeats } });
    const finished = await pollRun(run.id);
    if (finished.status === 'error') throw new Error(finished.error || '运行失败');

    const boot = await api('/api/bootstrap');
    state.boot = boot;
    const pair = defaultPair();
    state.selection = {
      candidate: pair?.candidate.run.id || '',
      baseline: pair?.baseline.run.id || '',
    };
    const summary = boot.runs.find((item) => item.run.id === run.id);
    toast(`运行完成：${promptName(promptId)} 通过率 ${pct(summary?.pass_rate || 0, 1)}`);
    if (location.hash === '#/compare') render();
    else location.hash = '#/compare';
  } catch (error) {
    toast(error.message, true);
    button.disabled = false;
    button.textContent = '开始运行';
    state.polling = false;
  }
}

function pollRun(runId) {
  return new Promise((resolve, reject) => {
    const tick = async () => {
      try {
        const data = await api(`/api/runs/${runId}`);
        const run = data.run;
        const total = run.total_cases || 1;
        const done = run.done_cases || 0;
        $('#progressCount').textContent = `${done} / ${run.total_cases}`;
        $('#progressFill').style.width = `${Math.min(100, (done / total) * 100)}%`;
        $('#progressLabel').textContent = run.status === 'done' ? '完成' : `正在运行 · ${done}/${run.total_cases}`;
        if (run.status === 'done' || run.status === 'error') {
          $('#progressFill').style.width = '100%';
          state.polling = false;
          resolve(run);
          return;
        }
        state.pollTimer = setTimeout(tick, 650);
      } catch (error) {
        state.polling = false;
        reject(error);
      }
    };
    tick();
  });
}

/* ---------------- compare ---------------- */

async function renderCompare(token) {
  const completed = doneRuns();
  if (completed.length < 2) {
    if (token !== state.renderToken) return;
    const emptyHint = completed.length === 0
      ? '还没有完成的运行。先跑 v1 作为基线，再跑 v2 作为候选。'
      : `目前只完成 1 次运行（${esc(promptName(completed[0].run.prompt_id))}），再跑另一个版本就能对比。`;
    $('#view').innerHTML = `
      <div class="panel">
        <div class="empty">
          <strong>还差一次运行才能对比</strong>
          <p style="margin:6px auto 0;max-width:460px">${emptyHint}</p>
          <div class="empty-actions">
            <a class="btn primary" href="#/run">去运行</a>
            <a class="btn ghost" href="#/dataset">先看测试集</a>
          </div>
        </div>
      </div>`;
    return;
  }

  if (!state.selection.candidate) {
    const pair = defaultPair();
    state.selection = { candidate: pair.candidate.run.id, baseline: pair.baseline.run.id };
  }

  $('#view').innerHTML = skeleton();
  let payload;
  try {
    payload = await api(`/api/compare?baseline=${encodeURIComponent(state.selection.baseline)}&candidate=${encodeURIComponent(state.selection.candidate)}`);
  } catch (error) {
    if (token !== state.renderToken) return;
    $('#view').innerHTML = `<div class="panel"><div class="empty">${esc(error.message)}</div></div>`;
    return;
  }
  if (token !== state.renderToken) return;
  state.compare = payload;

  const options = completed.map((item) => {
    const run = item.run;
    return `<option value="${esc(run.id)}">${esc(promptName(run.prompt_id))} · ${esc(shortTime(run.created_at))} · ${pct(item.pass_rate, 0)}</option>`;
  }).join('');

  const { baseline, candidate, delta: change, slices, transitions, gate } = payload;
  const fixedCount = (transitions.fixed || []).length;
  const regressedCount = (transitions.regressed || []).length;
  const keptCount = (transitions.both_pass || []).length;
  const failedCount = (transitions.both_fail || []).length;
  const compareSummary = `${promptName(baseline.run.prompt_id)} → ${promptName(candidate.run.prompt_id)}：${payload.cases.length} 条样本里，修复 ${fixedCount} 条、退步 ${regressedCount} 条、${keptCount} 条保持通过${failedCount ? `、${failedCount} 条仍然失败` : ''}。`;
  const gateMeta = {
    pass: { cls: 'pass', icon: '✓', title: '可以发布', note: '两项硬指标达标，且没有回归。' },
    review: { cls: 'review', icon: '!', title: '需要人工确认', note: '数字达标，但存在回归样本，不能自动放行。' },
    block: { cls: 'block', icon: '×', title: '阻止发布', note: '硬指标未达标，先修问题再谈发布。' },
  }[gate.decision] || { cls: 'review', icon: '!', title: gate.decision, note: '' };

  const sliceRows = slices.map((slice) => `
    <div class="slice-row">
      <span class="name">${esc(slice.name)}</span>
      <div class="slice-bars">
        <div class="bar base"><i style="width:${(slice.baseline_pass / Math.max(1, slice.total)) * 100}%"></i></div>
        <div class="bar cand"><i style="width:${(slice.candidate_pass / Math.max(1, slice.total)) * 100}%"></i></div>
      </div>
      <span class="count">${slice.baseline_pass}→${slice.candidate_pass}/${slice.total}</span>
    </div>`).join('');

  const transitionSegments = [
    ['fixed', '修复'], ['both_pass', '保持通过'], ['regressed', '退步'], ['both_fail', '仍然失败'],
  ].map(([key, label]) => {
    const count = (transitions[key] || []).length;
    return { key, label, count, width: (count / Math.max(1, payload.cases.length)) * 100 };
  });

  const caseRows = payload.cases.map((row) => `
    <tr class="clickable" data-slug="${esc(row.case.slug)}">
      <td>
        <div class="case-title">
          <strong>${esc(row.case.title)}</strong>
          <small class="mono">${esc(row.case.slug)}</small>
        </div>
      </td>
      <td><span class="tag">${esc(row.case.slice)}</span></td>
      <td>${verdictPill(row.baseline?.pass)}</td>
      <td>${verdictPill(row.candidate?.pass)}</td>
      <td>${transitionPill(row.transition)}</td>
      <td><div class="findings">${findingChips(row.candidate?.hard_fails)}</div></td>
    </tr>`).join('');

  const html = `
    <div class="compare-toolbar">
      <label class="field">基线 A
        <select id="baselineSelect">${options}</select>
      </label>
      <button class="swap-btn" id="swapRuns" title="交换基线与候选">⇄</button>
      <label class="field">候选 B
        <select id="candidateSelect">${options}</select>
      </label>
      <span class="hint">同一测试集 · 同一模型 · 仅提示词不同</span>
    </div>

    <p class="compare-summary">${esc(compareSummary)}</p>

    <div class="gate-banner ${gateMeta.cls}">
      <span class="gate-icon">${gateMeta.icon}</span>
      <div class="gate-copy">
        <h3>${gateMeta.title}<span class="hint" style="font-weight:400"> · ${esc(gateMeta.note)}</span></h3>
        <ul>${gate.reasons.map((reason) => `<li>${esc(reason.message)}</li>`).join('')}</ul>
        <p class="hint">门槛：通过率 ≥ ${pct(gate.thresholds.min_pass_rate, 0)} · 硬性错误 ≤ ${gate.thresholds.max_hard_fail_cases} 条 · 回归 ≤ ${gate.thresholds.max_regressions} 条</p>
      </div>
    </div>

    <div class="metric-grid">
      <article class="metric">
        <span class="label">通过率 <span class="tip" tabindex="0" title="一次运行里所有重复都通过，这条样本才算通过" data-tip="严格口径：一次运行里所有重复都通过，这条样本才算通过。">?</span></span>
        <span class="value">${pct(candidate.pass_rate, 1)}</span>
        <span class="foot">${candidate.passed_cases}/${candidate.total_cases} 条 ${delta(change.pass_rate)}</span>
      </article>
      <article class="metric">
        <span class="label">硬性错误用例 <span class="tip" tabindex="0" title="编造负责人、误判承诺、遗漏任务等，一条即判该样本失败" data-tip="编造负责人或时间、把建议当承诺、遗漏任务、误判取消，任意一条触发即该样本失败。">?</span></span>
        <span class="value">${candidate.hard_fail_cases}</span>
        <span class="foot">条 ${deltaCount(change.hard_fail_cases)}</span>
      </article>
      <article class="metric">
        <span class="label">平均质量分 <span class="tip" tabindex="0" title="只反映表达与依据质量，不参与发布门禁" data-tip="只反映表达与依据质量，不参与发布门禁，别拿它替代硬指标。">?</span></span>
        <span class="value">${candidate.avg_quality.toFixed(1)}</span>
        <span class="foot">分 ${deltaCount(change.avg_quality, ' 分')}</span>
      </article>
      <article class="metric">
        <span class="label">P95 延迟 <span class="tip" tabindex="0" title="95% 的模型调用快于这个值" data-tip="95% 的模型调用快于这个值；成本按 token 与单价估算。">?</span></span>
        <span class="value">${candidate.p95_latency}<small> ms</small></span>
        <span class="foot">成本 ${money(candidate.cost)} ${deltaCount(change.p95_latency, ' ms')}</span>
      </article>
    </div>

    <div class="compare-grid">
      <section class="panel">
        <div class="panel-head"><h2>分场景通过</h2><span class="hint">细条为基线 A，粗条为候选 B</span></div>
        <div class="slice-chart">${sliceRows}</div>
        <div class="legend">
          <span><i style="background:#c3c9bd"></i>基线 ${esc(promptName(baseline.run.prompt_id))}</span>
          <span><i style="background:#17795c"></i>候选 ${esc(promptName(candidate.run.prompt_id))}</span>
        </div>
      </section>

      <section class="panel">
        <div class="panel-head"><h2>用例变化</h2><span class="hint">${payload.cases.length} 条样本</span></div>
        <div class="transition-bar">
          ${transitionSegments.map((segment) => `<i class="${segment.key}" style="width:${segment.width}%"></i>`).join('')}
        </div>
        <ul class="transition-list">
          ${transitionSegments.map((segment) => `
            <li>
              <span class="k"><i class="dot" style="background:${{ fixed: '#17795c', both_pass: '#bcdccb', regressed: '#a84634', both_fail: '#c9c3b4' }[segment.key]}"></i>${segment.label}</span>
              <span class="v">${segment.count}</span>
            </li>`).join('')}
        </ul>
        <p class="hint" style="margin:12px 0 0">退步样本会被门禁单独拦下，即使整体通过率更高。</p>
      </section>
    </div>

    ${candidate.hard_fail_cases > 0 ? `
      <section class="panel" style="margin-bottom:18px">
        <div class="panel-head"><h2>硬性错误分布</h2><span class="hint">按用例计数，同一用例同类错误只算一次</span></div>
        <div class="findings">
          ${Object.entries(candidate.hard_fail_counts).map(([type, count]) => `
            <span class="f-chip">${esc(state.boot.labels.hard[type] || type)} × ${count}</span>`).join('')}
        </div>
      </section>` : ''}

    <section class="panel">
      <div class="panel-head"><h2>逐条结果</h2><span class="hint">点任意一行查看原文、输出与评分依据</span></div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr><th>样本</th><th>场景</th><th>基线 A</th><th>候选 B</th><th>变化</th><th>候选硬性错误</th></tr>
          </thead>
          <tbody>${caseRows}</tbody>
        </table>
      </div>
    </section>`;

  $('#view').innerHTML = html;

  $('#baselineSelect').value = state.selection.baseline;
  $('#candidateSelect').value = state.selection.candidate;

  $('#baselineSelect').addEventListener('change', (event) => {
    state.selection.baseline = event.target.value;
    render();
  });
  $('#candidateSelect').addEventListener('change', (event) => {
    state.selection.candidate = event.target.value;
    render();
  });
  $('#swapRuns').addEventListener('click', () => {
    state.selection = { baseline: state.selection.candidate, candidate: state.selection.baseline };
    render();
  });
  document.querySelectorAll('tr[data-slug]').forEach((row) => {
    row.addEventListener('click', () => openCase(row.dataset.slug));
  });
}

/* ---------------- case drawer ---------------- */

function judgeLabel(source) {
  return { heuristic: '本地启发式裁判', llm: 'DeepSeek 裁判', skipped: '已跳过语义评分' }[source] || source;
}

function resultCard(title, runId, promptId, results, promptNameValue) {
  const result = results?.[0];
  if (!result) {
    return `<section class="panel result-card"><h3>${esc(title)}</h3><div class="empty">这次运行没有该样本的结果</div></section>`;
  }

  const hard = result.hard_fails || [];
  const soft = result.soft_issues || [];
  const checks = result.rule?.checks || [];
  const judge = result.judge || {};

  return `
    <section class="panel result-card">
      <h3>${esc(title)} ${verdictPill(result.verdict === 'pass')}
        <span class="hint" style="font-weight:400">${esc(promptNameValue || promptName(promptId))}</span>
      </h3>
      <p class="hint mono" style="margin:6px 0 0">质量 ${result.quality} · ${ms(result.latency_ms)} · ${money(result.cost)}</p>

      <pre class="raw">${esc(result.raw_output || '（无输出）')}</pre>

      <ul class="checks">
        ${checks.map((check) => `
          <li>
            <span class="mark ${check.ok ? 'ok' : 'bad'}">${check.ok ? '✓' : '×'}</span>
            <span>${esc(check.label)}<small>${esc(check.detail)}</small></span>
          </li>`).join('')}
      </ul>

      ${hard.length ? `<ul class="finding-list">${hard.map((finding) => `<li>${esc(finding.message)}</li>`).join('')}</ul>` : ''}
      ${soft.length ? `<ul class="finding-list">${soft.map((finding) => `<li class="soft">${esc(finding.message)}</li>`).join('')}</ul>` : ''}

      <div class="judge-box">
        <strong>${esc(judgeLabel(judge.source))}</strong>
        ${judge.error ? ` · 调用失败：${esc(judge.error)}` : ''}
        ${judge.summary ? `<div>${esc(judge.summary)}</div>` : ''}
        ${(judge.items || []).filter((item) => item.reason).slice(0, 3).map((item) => `<div>· ${esc(item.reason)}</div>`).join('')}
      </div>
    </section>`;
}

async function openCase(slug) {
  const { baseline, candidate } = state.selection;
  const drawer = $('#drawer');
  const content = $('#drawerContent');
  content.innerHTML = `<div class="drawer-body"><div class="empty">加载中…</div></div>`;
  drawer.hidden = false;
  document.body.style.overflow = 'hidden';

  try {
    const data = await api(`/api/case?slug=${encodeURIComponent(slug)}&baseline=${encodeURIComponent(baseline)}&candidate=${encodeURIComponent(candidate)}`);
    const evalCase = data.case;
    const quotes = [...(data.baseline?.results || []), ...(data.candidate?.results || [])]
      .flatMap((result) => (result.parsed?.action_items || []).map((item) => item.evidence));

    content.innerHTML = `
      <div class="drawer-top">
        <div>
          <h2>${esc(evalCase.title)}</h2>
          <div class="meta">
            <span class="tag">${esc(evalCase.slice)}</span>
            <span class="tag mono">${esc(evalCase.slug)}</span>
            <span class="tag">会议日期 ${esc(evalCase.meeting_date)}</span>
          </div>
        </div>
        <button class="close-btn" data-close aria-label="关闭">×</button>
      </div>
      <div class="drawer-body">
        <p class="note">${esc(evalCase.note)}</p>
        <div>
          <div class="panel-head"><h2 style="font-size:13px">会议原文</h2><span class="hint">高亮为模型给出的依据</span></div>
          <div class="transcript">${highlight(evalCase.transcript, quotes)}</div>
        </div>
        <div class="result-grid">
          ${resultCard('基线 A', baseline, data.baseline?.run_id, data.baseline?.results)}
          ${resultCard('候选 B', candidate, data.candidate?.run_id, data.candidate?.results)}
        </div>
        <div class="panel">
          <div class="panel-head"><h2 style="font-size:13px">期望行为</h2><span class="hint">约束而非唯一答案</span></div>
          ${evalCase.expect.no_action ? '<p class="hint">会议没有行动项，正确输出是空数组；任何一条任务都算编造。</p>' : `
            <table class="expect">
              <thead><tr><th>任务需包含</th><th>负责人</th><th>截止时间</th><th>状态</th></tr></thead>
              <tbody>
                ${evalCase.expect.items.map((item) => `
                  <tr>
                    <td>${esc(item.contains.join(' / '))}</td>
                    <td>${item.owner ? esc(item.owner) : '<span class="hint">必须留空</span>'}</td>
                    <td class="mono">${item.due ? esc(item.due) : '<span class="hint">必须留空</span>'}</td>
                    <td>${statusPill(item.status)}</td>
                  </tr>`).join('')}
              </tbody>
            </table>`}
        </div>
      </div>`;
  } catch (error) {
    content.innerHTML = `<div class="drawer-body"><div class="empty">${esc(error.message)}</div></div>`;
  }
}

function closeDrawer() {
  $('#drawer').hidden = true;
  document.body.style.overflow = '';
}

function promptBodyHTML(body) {
  return esc(body)
    .split('{meeting_date}').join('<mark class="placeholder">{meeting_date}</mark>')
    .split('{transcript}').join('<mark class="placeholder">{transcript}</mark>');
}

function openPromptDrawer(promptId) {
  const prompt = (state.boot?.prompts || []).find((item) => item.id === promptId);
  if (!prompt) return;
  $('#drawerContent').innerHTML = `
    <div class="drawer-top">
      <div>
        <p class="eyebrow">PROMPT · v${esc(prompt.version)}</p>
        <h2>${esc(prompt.name)}</h2>
        <div class="meta">
          <span class="tag mono">${esc(prompt.id)}</span>
          <span class="tag">只改提示词，模型与样本不变</span>
        </div>
      </div>
      <button class="close-btn" data-close aria-label="关闭">×</button>
    </div>
    <div class="drawer-body">
      ${prompt.notes ? `<p class="note">${esc(prompt.notes)}</p>` : ''}
      <div class="panel prompt-panel">
        <div class="panel-head"><h2 style="font-size:13px">完整提示词</h2><span class="hint">运行时替换两个占位符</span></div>
        <pre class="prompt-body">${promptBodyHTML(prompt.body)}</pre>
      </div>
      <p class="hint">这是运行时真正发给模型的指令。任务、模型、温度和测试集都不变，只替换这段文本，收益才能归因。</p>
    </div>`;
  $('#drawer').hidden = false;
  document.body.style.overflow = 'hidden';
}

/* ---------------- dataset ---------------- */

function renderDataset(token) {
  if (token !== state.renderToken) return;
  const dataset = state.boot.datasets[0];
  const cases = state.boot.cases;
  const slices = ['全部', ...state.boot.slices];
  const filtered = state.filter === '全部' ? cases : cases.filter((item) => item.slice === state.filter);

  const card = (evalCase) => `
    <section class="panel case-card">
      <div class="top">
        <h3>${esc(evalCase.title)}</h3>
        <span class="tag">${esc(evalCase.slice)}</span>
      </div>
      <div class="transcript">${esc(evalCase.transcript)}</div>
      ${evalCase.expect.no_action ? `
        <p class="hint">期望：<strong>空数组</strong>。这条样本专门检查“强行生成任务”。</p>
      ` : `
        <table class="expect">
          <thead><tr><th>任务需包含</th><th>负责人</th><th>截止</th><th>状态</th></tr></thead>
          <tbody>
            ${evalCase.expect.items.map((item) => `
              <tr>
                <td>${esc(item.contains.join(' / '))}</td>
                <td>${item.owner ? esc(item.owner) : '<span class="hint">必须留空</span>'}</td>
                <td class="mono">${item.due ? esc(item.due) : '<span class="hint">必须留空</span>'}</td>
                <td>${statusPill(item.status)}</td>
              </tr>`).join('')}
          </tbody>
        </table>`}
      <p class="note">${esc(evalCase.note)}</p>
    </section>`;

  $('#view').innerHTML = `
    <section class="panel" style="margin-bottom:18px">
      <div class="dataset-head">
        <div>
          <h2>${esc(dataset.name)}</h2>
          <div class="dataset-meta">
            <span class="tag">${dataset.kind === 'holdout' ? '验收集' : '调试集'}</span>
            <span class="tag mono">${esc(dataset.id)}</span>
            <span class="tag">${cases.length} 条样本</span>
            <span class="tag">${state.boot.slices.length} 类场景</span>
          </div>
        </div>
      </div>
      <p class="hint" style="margin:0">${esc(dataset.notes)}</p>
    </section>

    <p class="dataset-intro">
      这 ${cases.length} 条不是随机抽样，而是专门挑的边界：正常表达、信息缺失、模糊承诺、决策变更、取消任务、无行动项。
      通过率衡量的是“已知的坑有没有踩”，不是线上准确率。
    </p>

    <div class="filters">
      ${slices.map((slice) => {
        const count = slice === '全部' ? cases.length : cases.filter((item) => item.slice === slice).length;
        return `<button class="filter-btn ${state.filter === slice ? 'active' : ''}" data-slice="${esc(slice)}">${esc(slice)}<span class="count">${count}</span></button>`;
      }).join('')}
    </div>

    <div class="case-grid">${filtered.map(card).join('')}</div>`;

  document.querySelectorAll('.filter-btn').forEach((button) => {
    button.addEventListener('click', () => {
      state.filter = button.dataset.slice;
      render();
    });
  });
}

/* ---------------- gate ---------------- */

async function renderGate(token) {
  const completed = doneRuns();
  const thresholds = state.boot.thresholds;
  const gates = state.boot.gates || [];

  if (completed.length === 0) {
    if (token !== state.renderToken) return;
    $('#view').innerHTML = `<div class="panel"><div class="empty"><strong>还没有可判定的运行</strong>先跑一次评测，门禁才有输入。</div></div>`;
    return;
  }

  const pair = defaultPair();
  const candidate = pair.candidate;
  const baseline = pair.baseline;
  let gate;
  let regressions = 0;

  try {
    const payload = await api(`/api/compare?baseline=${encodeURIComponent(baseline.run.id)}&candidate=${encodeURIComponent(candidate.run.id)}`);
    gate = payload.gate;
    regressions = (payload.transitions?.regressed || []).length;
    state.compare = payload;
  } catch (error) {
    gate = { decision: 'review', reasons: [{ level: 'review', message: error.message }], thresholds };
  }
  if (token !== state.renderToken) return;

  const gateMeta = {
    pass: { cls: 'pass', icon: '✓', title: '可以发布', note: '达标且无回归' },
    review: { cls: 'review', icon: '!', title: '需要人工确认', note: '数字达标，但有回归' },
    block: { cls: 'block', icon: '×', title: '阻止发布', note: '硬指标未达标' },
  }[gate.decision] || { cls: 'review', icon: '!', title: gate.decision, note: '' };

  const history = gates.length ? gates.map((item) => `
    <div class="history-item">
      <div>
        <strong>${esc(promptName(completed.find((run) => run.run.id === item.candidate_run_id)?.run.prompt_id) || '候选')}</strong>
        <small> · ${esc(shortTime(item.created_at))}</small>
        <div class="hint">${esc(item.reasons?.[0]?.message || '')}</div>
      </div>
      <span class="pill ${item.decision === 'pass' ? 'pass' : item.decision === 'block' ? 'fail' : 'review'}">
        ${item.decision === 'pass' ? '通过' : item.decision === 'block' ? '阻断' : '复核'}
      </span>
    </div>`).join('') : '<div class="empty">还没有发布判定记录</div>';

  $('#view').innerHTML = `
    <div class="gate-layout">
      <section class="panel">
        <div class="panel-head"><h2>发布门槛</h2><span class="hint">按错误类型设限</span></div>
        <p class="hint" style="margin:0">门禁只看硬性错误与回归，不看平均分——格式再漂亮，编造负责人也不能发。</p>

        <div class="threshold-grid">
          <label class="field">最低通过率（0–1）
            <input type="number" id="minPassRate" min="0" max="1" step="0.01" value="${thresholds.min_pass_rate}">
            <small class="field-hint">低于此值直接阻断</small>
          </label>
          <label class="field">允许的硬性错误用例数
            <input type="number" id="maxHardFail" min="0" step="1" value="${thresholds.max_hard_fail_cases}">
            <small class="field-hint">编造、误判承诺等，建议 0</small>
          </label>
          <label class="field">允许的回归用例数
            <input type="number" id="maxRegressions" min="0" step="1" value="${thresholds.max_regressions}">
            <small class="field-hint">通过变失败的样本，建议 0</small>
          </label>
        </div>

        <div class="preset-row">
          <span class="hint">快速预设</span>
          <button type="button" class="filter-btn" data-preset="loose">宽松 80% / 2 / 2</button>
          <button type="button" class="filter-btn" data-preset="default">默认 85% / 1 / 0</button>
          <button type="button" class="filter-btn" data-preset="strict">严格 95% / 0 / 0</button>
        </div>

        <button class="btn primary" id="saveGate">用最新两次运行判定</button>
        <p class="hint" style="margin:12px 0 0">
          判定对象：${esc(promptName(baseline.run.prompt_id))} → ${esc(promptName(candidate.run.prompt_id))}
          （${esc(shortTime(baseline.run.created_at))} → ${esc(shortTime(candidate.run.created_at))}）
        </p>
      </section>

      <div class="gate-result">
        <div class="panel">
          <div class="gate-banner ${gateMeta.cls}" style="margin-bottom:0">
            <span class="gate-icon">${gateMeta.icon}</span>
            <div class="gate-copy">
              <h3>${gateMeta.title}<span class="hint" style="font-weight:400"> · ${esc(gateMeta.note)}</span></h3>
              <ul>${gate.reasons.map((reason) => `<li>${esc(reason.message)}</li>`).join('')}</ul>
              <p class="hint">通过率 ${pct(candidate.pass_rate, 1)} · 硬性错误 ${candidate.hard_fail_cases} 条 · 回归 ${regressions} 条</p>
            </div>
          </div>
        </div>

        <section class="panel" style="margin-top:16px">
          <div class="panel-head"><h2>发布记录</h2><span class="hint">最近 ${gates.length} 次</span></div>
          <div class="history-list">${history}</div>
        </section>
      </div>
    </div>`;

  const presets = { loose: [0.8, 2, 2], default: [0.85, 1, 0], strict: [0.95, 0, 0] };
  document.querySelectorAll('[data-preset]').forEach((button) => {
    button.addEventListener('click', () => {
      const [rate, hard, regression] = presets[button.dataset.preset];
      $('#minPassRate').value = rate;
      $('#maxHardFail').value = hard;
      $('#maxRegressions').value = regression;
      document.querySelectorAll('[data-preset]').forEach((item) => item.classList.toggle('active', item === button));
      toast('已填入预设，点下面的判定按钮生效');
    });
  });

  $('#saveGate').addEventListener('click', async () => {
    const payload = {
      baseline: baseline.run.id,
      candidate: candidate.run.id,
      thresholds: {
        min_pass_rate: Number($('#minPassRate').value),
        max_hard_fail_cases: Number($('#maxHardFail').value),
        max_regressions: Number($('#maxRegressions').value),
      },
    };
    try {
      await api('/api/gate', { method: 'POST', body: payload });
      state.boot = await api('/api/bootstrap');
      toast('已保存门槛并记录判定');
      render();
    } catch (error) {
      toast(error.message, true);
    }
  });
}

/* ---------------- boot ---------------- */

function renderMode() {
  const chip = $('#modeChip');
  const live = state.boot?.mode === 'live';
  chip.textContent = live ? '真实 API' : '回放模式';
  chip.className = `chip ${live ? 'live' : 'replay'}`;
  chip.title = live ? '调用 DeepSeek 生成输出与裁判' : '无 API key，使用预置输出；延迟与成本为模拟值';
  $('#modelName').textContent = state.boot?.model || 'deepseek-flash';
}

async function init() {
  try {
    state.boot = await api('/api/bootstrap');
  } catch (error) {
    $('#view').innerHTML = `<div class="panel"><div class="empty"><strong>无法连接服务</strong>${esc(error.message)}</div></div>`;
    return;
  }
  renderMode();
  window.addEventListener('hashchange', route);
  $('#drawer').addEventListener('click', (event) => {
    if (event.target.closest('[data-close]')) closeDrawer();
  });
  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') closeDrawer();
  });
  if (!location.hash) location.hash = '#/run';
  route();
}

init();
