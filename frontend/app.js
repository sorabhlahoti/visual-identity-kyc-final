const $ = (id) => document.getElementById(id);
const state = {
  apiBase: localStorage.getItem('visualKycApiBase') || 'http://localhost:8080',
  token: localStorage.getItem('visualKycToken') || '',
  pollTimer: null,
};

$('apiBase').value = state.apiBase;
$('token').value = state.token;

function normalizeBase(url) {
  return String(url || '').trim().replace(/\/$/, '');
}

function setLog(title, data) {
  const text = typeof data === 'string' ? data : JSON.stringify(data, null, 2);
  $('log').textContent = `${title}\n${text}`;
}

function saveConfig() {
  state.apiBase = normalizeBase($('apiBase').value);
  state.token = $('token').value.trim();
  localStorage.setItem('visualKycApiBase', state.apiBase);
  localStorage.setItem('visualKycToken', state.token);
}

function headers(extra = {}) {
  saveConfig();
  const h = { ...extra };
  if (state.token) h.Authorization = `Bearer ${state.token}`;
  return h;
}

async function request(path, options = {}) {
  saveConfig();
  const res = await fetch(`${state.apiBase}${path}`, options);
  const contentType = res.headers.get('content-type') || '';
  const body = contentType.includes('application/json') ? await res.json() : await res.text();
  if (!res.ok) {
    const message = typeof body === 'object' ? body.error || JSON.stringify(body) : body;
    throw new Error(message || `HTTP ${res.status}`);
  }
  return body;
}

function formToData(form) {
  const fd = new FormData(form);
  const file = form.querySelector('input[type="file"]').files[0];
  if (!file) throw new Error('Please select an image first.');
  fd.set('image', file);
  return fd;
}

function showHealth(ok, text, details) {
  const dot = $('healthDot');
  dot.classList.remove('ok', 'bad');
  dot.classList.add(ok ? 'ok' : 'bad');
  $('healthText').textContent = text;
  $('healthDetails').textContent = details;
}

async function health() {
  try {
    const data = await request('/health');
    showHealth(true, 'API is healthy', `${data.service || 'kyc-api'} running in ${data.mode || 'async'} mode`);
    setLog('Health response', data);
  } catch (err) {
    showHealth(false, 'API unreachable', err.message);
    setLog('Health error', err.message);
  }
}

async function getToken() {
  try {
    const data = await request('/auth/token', { method: 'POST' });
    $('token').value = data.token;
    state.token = data.token;
    localStorage.setItem('visualKycToken', data.token);
    setLog('Token generated', { type: data.type, token_preview: `${data.token.slice(0, 24)}...` });
  } catch (err) {
    setLog('Token error', err.message);
  }
}

async function submitJob(form, endpoint, label) {
  const button = form.querySelector('button[type="submit"]');
  button.disabled = true;
  try {
    const data = await request(endpoint, { method: 'POST', headers: headers(), body: formToData(form) });
    $('transactionId').value = data.transaction_id;
    setLog(`${label} accepted`, data);
    renderStatus({ transaction_id: data.transaction_id, type: endpoint.includes('verify') ? 'verify' : 'enroll', status: data.status });
  } catch (err) {
    setLog(`${label} error`, err.message);
  } finally {
    button.disabled = false;
  }
}

async function checkStatus() {
  const id = $('transactionId').value.trim();
  if (!id) throw new Error('Enter a transaction ID first.');
  const data = await request(`/kyc/status/${encodeURIComponent(id)}`, { headers: headers() });
  renderStatus(data);
  setLog('Status response', data);
  return data;
}

function fmtScore(v) {
  if (typeof v !== 'number') return '—';
  return `${Math.round(v * 100)}%`;
}

function renderStatus(data) {
  const result = data.result || {};
  const details = result.details || {};
  const liveness = details.liveness || {};
  const reasons = details.explainability?.decision_reasons || [];
  $('resultCards').innerHTML = `
    <div class="result-card"><b>Transaction</b>${data.transaction_id || '—'}<div class="metric"><span>Job type</span><strong>${data.type || '—'}</strong></div><div class="metric"><span>Async status</span><strong>${data.status || result.status || '—'}</strong></div></div>
    <div class="result-card"><b>Decision</b>${result.status || data.error || 'Waiting for worker...'}<div class="metric"><span>Confidence</span><strong>${fmtScore(result.confidence_score)}</strong></div><div class="metric"><span>DID</span><strong>${result.did ? result.did.slice(0, 28) + '...' : '—'}</strong></div></div>
    <div class="result-card"><b>Scores</b><div class="metric"><span>Face similarity</span><strong>${fmtScore(details.face_similarity)}</strong></div><div class="metric"><span>Name similarity</span><strong>${fmtScore(details.name_similarity)}</strong></div><div class="metric"><span>Demographic match</span><strong>${typeof details.demographic_match === 'boolean' ? details.demographic_match : '—'}</strong></div><div class="metric"><span>Liveness</span><strong>${fmtScore(liveness.score)}</strong></div></div>
    <div class="result-card"><b>Explainability</b>${reasons.length ? reasons.map(r => `<div class="metric"><span>${r}</span><strong>✓</strong></div>`).join('') : '<span class="muted">No final decision yet.</span>'}</div>
  `;
}

async function pollUntilDone() {
  clearInterval(state.pollTimer);
  $('pollBtn').disabled = true;
  try {
    await checkStatus();
    state.pollTimer = setInterval(async () => {
      try {
        const data = await checkStatus();
        if (['COMPLETED', 'FAILED'].includes(data.status)) {
          clearInterval(state.pollTimer);
          $('pollBtn').disabled = false;
        }
      } catch (err) {
        clearInterval(state.pollTimer);
        $('pollBtn').disabled = false;
        setLog('Polling error', err.message);
      }
    }, 1800);
  } catch (err) {
    $('pollBtn').disabled = false;
    setLog('Status error', err.message);
  }
}

$('apiBase').addEventListener('change', saveConfig);
$('token').addEventListener('change', saveConfig);
$('healthBtn').addEventListener('click', health);
$('tokenBtn').addEventListener('click', getToken);
$('statusBtn').addEventListener('click', () => checkStatus().catch(err => setLog('Status error', err.message)));
$('pollBtn').addEventListener('click', pollUntilDone);
$('clearLogBtn').addEventListener('click', () => setLog('Ready.', ''));
$('enrollForm').addEventListener('submit', (e) => { e.preventDefault(); submitJob(e.currentTarget, '/kyc/enroll', 'Enroll'); });
$('verifyForm').addEventListener('submit', (e) => { e.preventDefault(); submitJob(e.currentTarget, '/kyc/verify', 'Verify'); });

health();
