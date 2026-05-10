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

function formPreview(form) {
  return {
    name: form.elements.name?.value || '',
    dob: form.elements.dob?.value || '',
    gender: form.elements.gender?.value || '',
    image_name: form.querySelector('input[type="file"]')?.files?.[0]?.name || '',
  };
}

async function submitJob(form, endpoint, label) {
  const button = form.querySelector('button[type="submit"]');
  button.disabled = true;
  clearInterval(state.pollTimer);
  $('transactionId').value = '';
  const expectedType = endpoint.includes('verify') ? 'verify' : 'enroll';
  const expectedTopic = expectedType === 'verify' ? 'kyc_verify' : 'kyc_enroll';
  try {
    saveConfig();
    setLog(`${label} submitting`, {
      api_base_url: state.apiBase,
      endpoint,
      expected_type: expectedType,
      expected_kafka_topic: expectedTopic,
      form: formPreview(form),
      token_present: Boolean(state.token),
      token_preview: state.token ? `${state.token.slice(0, 18)}...` : 'missing',
    });
    const data = await request(endpoint, { method: 'POST', headers: headers(), body: formToData(form) });
    if (!data.transaction_id) throw new Error('API did not return transaction_id. Check backend response.');
    $('transactionId').value = data.transaction_id;
    const warnings = [];
    if (data.type && data.type !== expectedType) warnings.push(`Expected type ${expectedType}, API returned ${data.type}`);
    if (data.kafka_topic && data.kafka_topic !== expectedTopic) warnings.push(`Expected Kafka topic ${expectedTopic}, API returned ${data.kafka_topic}`);
    setLog(`${label} accepted`, { ...data, frontend_expected: { endpoint, type: expectedType, kafka_topic: expectedTopic }, warnings });
    renderStatus({ transaction_id: data.transaction_id, type: data.type || expectedType, status: data.status, kafka_topic: data.kafka_topic });
    await pollUntilDone(45);
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

function prettyStatus(v) {
  if (!v) return '—';
  return String(v).replaceAll('_', ' ').toLowerCase().replace(/\b\w/g, c => c.toUpperCase());
}

function prettyReason(reason) {
  const map = {
    liveness_passed: ['Liveness passed', '✓'],
    liveness_failed_or_not_available: ['Liveness failed / not available', '✗'],
    strong_face_match: ['Strong face match', '✓'],
    moderate_face_match: ['Moderate face match', '•'],
    weak_face_match: ['Weak face match', '✗'],
    name_match: ['Name match', '✓'],
    name_mismatch_or_not_in_shortlist: ['Name mismatch / not shortlisted', '✗'],
    demographic_hash_exact_match: ['Demographic hash exact match', '✓'],
    demographic_hash_mismatch: ['Demographic hash mismatch', '✗'],
    final_status_MATCHED: ['Final status', 'Matched'],
    final_status_PARTIAL_MATCH: ['Final status', 'Partial Match'],
    final_status_NO_MATCH: ['Final status', 'No Match'],
    final_status_NEW_USER: ['Final status', 'New User'],
    final_status_ALREADY_EXISTS: ['Final status', 'Already Exists'],
    final_status_POTENTIAL_FRAUD: ['Final status', 'Potential Fraud'],
  };
  return map[reason] || [String(reason).replaceAll('_', ' '), '•'];
}

function renderStatus(data) {
  const result = data.result || {};
  const details = result.details || {};
  const liveness = details.liveness || {};
  const reasons = details.explainability?.decision_reasons || [];
  const decision = result.status || data.error || (data.status === 'ACCEPTED' ? 'Waiting for worker...' : data.status);
  $('resultCards').innerHTML = `
    <div class="result-card"><b>Transaction</b>${data.transaction_id || '—'}<div class="metric"><span>Job type</span><strong>${prettyStatus(data.type)}</strong></div><div class="metric"><span>Async status</span><strong>${prettyStatus(data.status || result.status)}</strong></div></div>
    <div class="result-card"><b>Decision</b>${prettyStatus(decision)}<div class="metric"><span>Confidence</span><strong>${fmtScore(result.confidence_score)}</strong></div><div class="metric"><span>DID</span><strong>${result.did ? result.did.slice(0, 28) + '...' : '—'}</strong></div></div>
    <div class="result-card"><b>Scores</b><div class="metric"><span>Face similarity</span><strong>${fmtScore(details.face_similarity)}</strong></div><div class="metric"><span>Name similarity</span><strong>${fmtScore(details.name_similarity)}</strong></div><div class="metric"><span>Demographic match</span><strong>${typeof details.demographic_match === 'boolean' ? (details.demographic_match ? 'Yes' : 'No') : '—'}</strong></div><div class="metric"><span>Liveness score</span><strong>${fmtScore(liveness.score)}</strong></div><div class="metric"><span>Liveness passed</span><strong>${typeof liveness.passed === 'boolean' ? (liveness.passed ? 'Yes' : 'No') : '—'}</strong></div></div>
    <div class="result-card"><b>Explainable scoring</b>${reasons.length ? reasons.map(r => { const [label, value] = prettyReason(r); return `<div class="metric"><span>${label}</span><strong>${value}</strong></div>`; }).join('') : '<span class="muted">No final decision yet. If async status is ACCEPTED, click Poll until done.</span>'}</div>
  `;
}

async function pollUntilDone(maxAttempts = 45) {
  clearInterval(state.pollTimer);
  $('pollBtn').disabled = true;
  let attempts = 0;
  try {
    const first = await checkStatus();
    if (['COMPLETED', 'FAILED'].includes(first.status)) {
      $('pollBtn').disabled = false;
      return first;
    }
    return await new Promise((resolve, reject) => {
      state.pollTimer = setInterval(async () => {
        attempts += 1;
        try {
          const data = await checkStatus();
          if (['COMPLETED', 'FAILED'].includes(data.status)) {
            clearInterval(state.pollTimer);
            $('pollBtn').disabled = false;
            resolve(data);
          } else if (attempts >= maxAttempts) {
            clearInterval(state.pollTimer);
            $('pollBtn').disabled = false;
            setLog('Polling timed out', {
              message: 'The API accepted the job, but it did not finish within the frontend polling window. Check worker logs for this transaction ID.',
              transaction_id: $('transactionId').value.trim(),
              api_base_url: state.apiBase,
              next_command: `kubectl logs deployment/worker --since=20m | Select-String "${$('transactionId').value.trim()}"`,
              last_status: data,
            });
            resolve(data);
          }
        } catch (err) {
          clearInterval(state.pollTimer);
          $('pollBtn').disabled = false;
          setLog('Polling error', err.message);
          reject(err);
        }
      }, 1800);
    });
  } catch (err) {
    $('pollBtn').disabled = false;
    setLog('Status error', err.message);
    throw err;
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

setLog('Frontend version', { version: 'verify-debug-2026-05-10', note: 'Verify submissions show expected endpoint and Kafka topic. Use this to confirm the browser is calling /kyc/verify on the same backend you watch in kubectl logs.' });
health();
