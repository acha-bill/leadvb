(function () {
  'use strict';

  var script = document.currentScript;
  if (!script) return;
  // Optional: sites registered by domain in the dashboard need no key at all.
  var API_KEY = script.getAttribute('data-widget-key') || script.getAttribute('data-api-key') || '';

  var BASE = script.getAttribute('data-base-url');
  if (!BASE) {
    try { BASE = new URL(script.src).origin; } catch (e) { BASE = ''; }
  }
  if (!BASE) return;

  var LS_SESSION = 'lqw_session_' + (API_KEY || location.host);
  var LS_VISITOR = 'lqw_visitor';
  var SS_PROACTIVE = 'lqw_proactive_shown';
  var SS_EXIT = 'lqw_exit_shown';
  var SS_LOADED = 'lqw_loaded_sent';

  var cfg = null;
  var wsURL = '';
  var sessionToken = localStorage.getItem(LS_SESSION) || '';
  var visitorId = localStorage.getItem(LS_VISITOR);
  if (!visitorId) {
    visitorId = 'v_' + Math.random().toString(36).slice(2) + Date.now().toString(36);
    localStorage.setItem(LS_VISITOR, visitorId);
  }

  var open = false;
  var unread = 0;
  var seenIds = {};
  var lastMsgId = 0;
  var ws = null;
  var wsFails = 0;
  var pollTimer = null;
  var typingEl = null;
  var status = 'active';
  var ended = false;

  var root, panel, msgsEl, inputEl, sendBtn, btn, badgeEl, quickWrap;

  function fetchJSON(path, opts) {
    opts = opts || {};
    opts.headers = opts.headers || {};
    if (opts.body) opts.headers['Content-Type'] = 'application/json';
    return fetch(BASE + path, opts).then(function (r) { return r.json(); });
  }

  function track(type) {
    try {
      var payload = JSON.stringify({ key: API_KEY, type: type, visitor_id: visitorId, page_url: location.href });
      if (navigator.sendBeacon) {
        navigator.sendBeacon(BASE + '/api/widget/event', new Blob([payload], { type: 'application/json' }));
      } else {
        fetchJSON('/api/widget/event', { method: 'POST', body: payload });
      }
    } catch (e) { }
  }

  function pageAllowed(rules) {
    if (!rules || rules.mode === 'all' || !rules.patterns || !rules.patterns.length) return true;
    var url = location.href;
    var matched = rules.patterns.some(function (p) {
      p = (p || '').trim();
      if (!p) return false;
      if (p.indexOf('*') >= 0) {
        var re = new RegExp('^' + p.split('*').map(escRe).join('.*') + '$');
        return re.test(url) || re.test(location.pathname);
      }
      return url.indexOf(p) >= 0;
    });
    return rules.mode === 'exclude' ? !matched : matched;
  }

  function escRe(s) { return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); }

  function textColorFor(hex) {
    try {
      var c = hex.replace('#', '');
      if (c.length === 3) c = c.split('').map(function (x) { return x + x; }).join('');
      var r = parseInt(c.substr(0, 2), 16), g = parseInt(c.substr(2, 2), 16), b = parseInt(c.substr(4, 2), 16);
      return (0.299 * r + 0.587 * g + 0.114 * b) > 150 ? '#111827' : '#ffffff';
    } catch (e) { return '#ffffff'; }
  }

  function esc(s) {
    var d = document.createElement('div');
    d.textContent = s == null ? '' : String(s);
    return d.innerHTML;
  }

  function linkify(escaped) {
    return escaped.replace(/(https?:\/\/[^\s<]+)/g, '<a href="$1" target="_blank" rel="noopener">$1</a>');
  }

  function css(primary) {
    var onPrimary = textColorFor(primary);
    var side = cfg.position === 'left' ? 'left' : 'right';
    return '\
:host{all:initial}\
*{box-sizing:border-box;font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}\
button,textarea{font:inherit}\
.lq-btn{position:fixed;bottom:22px;' + side + ':22px;width:58px;height:58px;border-radius:18px;background:' + primary + ';color:' + onPrimary + ';border:1px solid rgba(255,255,255,.22);cursor:pointer;box-shadow:0 16px 36px rgba(24,39,42,.24);display:flex;align-items:center;justify-content:center;z-index:2147483000;transition:transform .18s cubic-bezier(.22,1,.36,1),box-shadow .18s ease}\
.lq-btn:hover{transform:translateY(-2px);box-shadow:0 20px 42px rgba(24,39,42,.28)}\
.lq-btn:active{transform:translateY(0)}\
.lq-btn:focus-visible,.lq-close:focus-visible,.lq-send:focus-visible,.lq-chip:focus-visible,.lq-ended button:focus-visible,.lq-input:focus-visible{outline:3px solid rgba(36,85,214,.35);outline-offset:3px}\
.lq-btn svg{width:27px;height:27px;fill:' + onPrimary + '}\
.lq-badge{position:absolute;top:-5px;right:-5px;min-width:21px;height:21px;border-radius:11px;background:#b72f39;color:#fff;font-size:11px;font-weight:700;display:none;align-items:center;justify-content:center;padding:0 6px;box-shadow:0 2px 8px rgba(24,39,42,.18)}\
.lq-panel{position:fixed;bottom:94px;' + side + ':22px;width:390px;max-width:calc(100vw - 32px);height:600px;max-height:calc(100dvh - 126px);background:#fff;border:1px solid #d9d9d2;border-radius:24px;box-shadow:0 28px 80px rgba(24,39,42,.2);display:none;flex-direction:column;overflow:hidden;z-index:2147483001;transform-origin:bottom ' + side + '}\
.lq-panel.open{display:flex;animation:lqopen .22s cubic-bezier(.22,1,.36,1) both}\
@keyframes lqopen{from{opacity:0;transform:translateY(8px) scale(.985)}to{opacity:1;transform:none}}\
.lq-head{background:#fff;color:#18272a;padding:14px 12px 14px 16px;display:flex;align-items:center;gap:11px;flex:0 0 auto;border-bottom:1px solid #e4e3dc}\
.lq-logo,.lq-brandmark{width:36px;height:36px;border-radius:11px;object-fit:cover;background:' + primary + ';color:' + onPrimary + ';display:grid;place-items:center;flex:0 0 auto}\
.lq-brandmark svg{width:19px;height:19px;fill:none;stroke:' + onPrimary + ';stroke-width:1.9;stroke-linecap:round;stroke-linejoin:round}\
.lq-head-title{font-size:14px;font-weight:700;line-height:1.25;letter-spacing:-.01em}\
.lq-head-sub{display:flex;align-items:center;gap:6px;color:#526064;font-size:11px;margin-top:2px}\
.lq-head-sub:before{content:"";width:7px;height:7px;border-radius:50%;background:#16734a;box-shadow:0 0 0 3px #def4e8}\
.lq-close{margin-left:auto;width:44px;height:44px;border:0;border-radius:12px;background:#ecebe5;color:#526064;cursor:pointer;display:grid;place-items:center;transition:background .18s ease,color .18s ease}\
.lq-close:hover{background:#deddd6;color:#18272a}\
.lq-close svg{width:19px;height:19px;fill:none;stroke:currentColor;stroke-width:1.8;stroke-linecap:round}\
.lq-msgs{flex:1 1 auto;overflow-y:auto;padding:18px 16px;background:#f6f5f0;display:flex;flex-direction:column;gap:11px;scrollbar-color:#c9c9c2 transparent;scrollbar-width:thin}\
.lq-row{display:flex}\
.lq-row.me{justify-content:flex-end}\
.lq-bubble{max-width:82%;padding:11px 14px;border-radius:16px;font-size:14px;line-height:1.5;white-space:pre-wrap;overflow-wrap:anywhere}\
.lq-row.bot .lq-bubble{background:#fff;color:#18272a;border-bottom-left-radius:5px;box-shadow:0 5px 18px rgba(24,39,42,.055)}\
.lq-row.owner .lq-bubble{background:#fff0d8;color:#47321e;border-bottom-left-radius:5px}\
.lq-row.me .lq-bubble{background:' + primary + ';color:' + onPrimary + ';border-bottom-right-radius:5px}\
.lq-bubble a{color:inherit;text-decoration:underline}\
.lq-owner-tag{font-size:10px;color:#98540c;margin:0 0 3px 6px;font-weight:650}\
.lq-typing{display:flex;gap:4px;padding:13px 16px;background:#fff;border-radius:16px;border-bottom-left-radius:5px;width:64px;box-shadow:0 5px 18px rgba(24,39,42,.055)}\
.lq-typing i{width:7px;height:7px;border-radius:50%;background:#8b9495;animation:lqb 1.1s infinite}\
.lq-typing i:nth-child(2){animation-delay:.18s}.lq-typing i:nth-child(3){animation-delay:.36s}\
@keyframes lqb{0%,60%,100%{transform:translateY(0);opacity:.5}30%{transform:translateY(-4px);opacity:1}}\
.lq-quick{display:flex;flex-wrap:wrap;gap:8px;padding:0 16px 12px;background:#f6f5f0}\
.lq-chip{min-height:44px;border:1px solid ' + primary + ';color:' + primary + ';background:#fff;border-radius:999px;padding:8px 14px;font-size:13px;font-weight:650;cursor:pointer;transition:background .18s ease,color .18s ease}\
.lq-chip:hover{background:' + primary + ';color:' + onPrimary + '}\
.lq-foot{flex:0 0 auto;border-top:1px solid #e4e3dc;background:#fff;padding-bottom:max(0px,env(safe-area-inset-bottom))}\
.lq-inputrow{display:flex;align-items:flex-end;gap:8px;margin:12px;border:1px solid #d9d9d2;border-radius:16px;padding:5px 5px 5px 10px;transition:border-color .18s ease,box-shadow .18s ease}\
.lq-inputrow:focus-within{border-color:' + primary + ';box-shadow:0 0 0 4px rgba(36,85,214,.12)}\
.lq-input{flex:1;min-height:40px;border:0;outline:0;resize:none;font-size:14px;line-height:1.45;max-height:96px;padding:9px 4px;color:#18272a;background:transparent}\
.lq-input::placeholder{color:#7c8789}\
.lq-send{background:' + primary + ';color:' + onPrimary + ';border:none;border-radius:12px;width:44px;height:44px;cursor:pointer;display:flex;align-items:center;justify-content:center;flex:0 0 auto;transition:opacity .18s ease,transform .18s cubic-bezier(.22,1,.36,1)}\
.lq-send:hover{transform:translateY(-1px)}\
.lq-send:disabled{opacity:.45;cursor:default}\
.lq-send svg{width:18px;height:18px;fill:' + onPrimary + '}\
.lq-brand{text-align:center;font-size:10px;color:#7c8789;padding:0 0 9px}\
.lq-brand a{color:#526064;text-decoration:none;font-weight:650}\
.lq-ended{padding:11px 16px;text-align:center;font-size:13px;color:#526064;background:#fff}\
.lq-ended button{min-height:44px;margin-top:7px;border:0;background:#ecebe5;border-radius:999px;padding:0 15px;font-size:13px;font-weight:650;cursor:pointer;color:#18272a}\
@media (max-width:480px){.lq-btn{bottom:max(16px,env(safe-area-inset-bottom));' + side + ':16px}.lq-panel{bottom:0;' + side + ':0;width:100vw;max-width:100vw;height:100dvh;max-height:100dvh;border-radius:0;border:0}.lq-head{padding-top:max(14px,env(safe-area-inset-top))}.lq-bubble{max-width:88%}}\
@media (prefers-reduced-motion:reduce){*{animation-duration:.01ms!important;animation-iteration-count:1!important;transition-duration:.01ms!important}}\
';
  }

  function build() {
    var host = document.createElement('div');
    host.id = 'lq-widget-host';
    document.body.appendChild(host);
    root = host.attachShadow ? host.attachShadow({ mode: 'open' }) : host;

    var style = document.createElement('style');
    style.textContent = css(cfg.primary_color || '#2455D6');
    root.appendChild(style);

    btn = document.createElement('button');
    btn.className = 'lq-btn';
    btn.setAttribute('aria-label', 'Open chat');
    btn.setAttribute('aria-expanded', 'false');
    btn.setAttribute('aria-controls', 'lq-chat-panel');
    btn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M12 3C6.9 3 3 6.5 3 10.8c0 2.4 1.2 4.5 3.2 5.9-.1.8-.5 2.1-1.5 3.1 0 0 2.4-.3 4.2-1.5.9.2 2 .4 3.1.4 5.1 0 9-3.5 9-7.9S17.1 3 12 3z"/></svg><span class="lq-badge"></span>';
    root.appendChild(btn);
    badgeEl = btn.querySelector('.lq-badge');

    panel = document.createElement('div');
    panel.className = 'lq-panel';
    panel.id = 'lq-chat-panel';
    panel.setAttribute('role', 'dialog');
    panel.setAttribute('aria-label', 'Chat with ' + (cfg.company_name || 'our team'));
    panel.setAttribute('aria-hidden', 'true');
    var name = esc(cfg.company_name || 'Chat with us');
    var logo = cfg.logo_url
      ? '<img class="lq-logo" src="' + esc(cfg.logo_url) + '" alt="">'
      : '<span class="lq-brandmark" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M21 15a4 4 0 0 1-4 4H8l-5 3V7a4 4 0 0 1 4-4h10a4 4 0 0 1 4 4z"/></svg></span>';
    panel.innerHTML =
      '<div class="lq-head">' + logo +
      '<div><div class="lq-head-title">' + name + '</div><div class="lq-head-sub">Online now</div></div>' +
      '<button class="lq-close" aria-label="Close chat"><svg viewBox="0 0 24 24"><path d="M6 6l12 12M18 6 6 18"/></svg></button></div>' +
      '<div class="lq-msgs"></div>' +
      '<div class="lq-quick"></div>' +
      '<div class="lq-foot">' +
      '<div class="lq-inputrow"><textarea class="lq-input" rows="1" aria-label="Message" placeholder="Write a message"></textarea>' +
      '<button class="lq-send" aria-label="Send"><svg viewBox="0 0 24 24"><path d="M2.5 21.5l19-9.5-19-9.5v7.6L15 12 2.5 13.9z"/></svg></button></div>' +
      (cfg.branding ? '<div class="lq-brand">Powered by <a href="' + BASE + '" target="_blank" rel="noopener">Lead Qualifier</a></div>' : '') +
      '</div>';
    root.appendChild(panel);

    msgsEl = panel.querySelector('.lq-msgs');
    inputEl = panel.querySelector('.lq-input');
    sendBtn = panel.querySelector('.lq-send');
    quickWrap = panel.querySelector('.lq-quick');

    btn.addEventListener('click', function () { open ? closePanel() : openPanel('open'); });
    panel.querySelector('.lq-close').addEventListener('click', closePanel);
    sendBtn.addEventListener('click', submit);
    inputEl.addEventListener('keydown', function (e) {
      if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); submit(); }
    });
    inputEl.addEventListener('input', function () {
      inputEl.style.height = 'auto';
      inputEl.style.height = Math.min(inputEl.scrollHeight, 96) + 'px';
    });
    panel.addEventListener('keydown', function (e) {
      if (e.key === 'Escape') closePanel();
    });
  }

  function openPanel(trigger) {
    open = true;
    panel.classList.add('open');
    panel.setAttribute('aria-hidden', 'false');
    btn.setAttribute('aria-expanded', 'true');
    btn.setAttribute('aria-label', 'Close chat');
    unread = 0;
    renderBadge();
    track('opened');
    ensureSession(trigger).then(function () {
      inputEl.focus();
      scrollBottom();
    });
  }

  function closePanel() {
    open = false;
    panel.classList.remove('open');
    panel.setAttribute('aria-hidden', 'true');
    btn.setAttribute('aria-expanded', 'false');
    btn.setAttribute('aria-label', 'Open chat');
    btn.focus();
    track('closed');
  }

  function renderBadge() {
    if (unread > 0) {
      badgeEl.style.display = 'flex';
      badgeEl.textContent = unread > 9 ? '9+' : String(unread);
    } else {
      badgeEl.style.display = 'none';
    }
  }

  function scrollBottom() {
    if (msgsEl) msgsEl.scrollTop = msgsEl.scrollHeight;
  }

  function addMessage(m) {
    if (m.id && seenIds[m.id]) return;
    if (m.id) { seenIds[m.id] = true; if (m.id > lastMsgId) lastMsgId = m.id; }
    hideTyping();
    var row = document.createElement('div');
    var cls = m.role === 'visitor' ? 'me' : (m.role === 'owner' ? 'owner' : 'bot');
    row.className = 'lq-row ' + cls;
    var inner = '';
    if (m.role === 'owner') inner += '<div><div class="lq-owner-tag">Team member</div><div class="lq-bubble">' + linkify(esc(m.content)) + '</div></div>';
    else inner = '<div class="lq-bubble">' + linkify(esc(m.content)) + '</div>';
    row.innerHTML = inner;
    msgsEl.appendChild(row);
    renderQuickReplies(m.role !== 'visitor' ? m.quick_replies : null);
    if (!open && m.role !== 'visitor') { unread++; renderBadge(); }
    scrollBottom();
  }

  function renderQuickReplies(list) {
    quickWrap.innerHTML = '';
    if (!list || !list.length || ended) return;
    list.forEach(function (q) {
      var chip = document.createElement('button');
      chip.className = 'lq-chip';
      chip.textContent = q;
      chip.addEventListener('click', function () { sendText(q); });
      quickWrap.appendChild(chip);
    });
  }

  function showTyping() {
    if (typingEl) return;
    typingEl = document.createElement('div');
    typingEl.className = 'lq-row bot';
    typingEl.innerHTML = '<div class="lq-typing"><i></i><i></i><i></i></div>';
    msgsEl.appendChild(typingEl);
    scrollBottom();
  }

  function hideTyping() {
    if (typingEl && typingEl.parentNode) typingEl.parentNode.removeChild(typingEl);
    typingEl = null;
  }

  function setStatus(newStatus) {
    status = newStatus;
    if (status === 'qualified' || status === 'disqualified' || status === 'closed' || status === 'abandoned') {
      if (!ended) {
        ended = true;
        renderQuickReplies(null);
        var note = document.createElement('div');
        note.className = 'lq-ended';
        note.innerHTML = 'This chat is closed.<br><button type="button">Start a new chat</button>';
        note.querySelector('button').addEventListener('click', function () {
          localStorage.removeItem(LS_SESSION);
          sessionToken = '';
          seenIds = {}; lastMsgId = 0; ended = false; status = 'active';
          msgsEl.innerHTML = '';
          note.parentNode.removeChild(note);
          inputEl.disabled = false; sendBtn.disabled = false;
          teardownRealtime();
          ensureSession('open');
        });
        panel.querySelector('.lq-foot').insertBefore(note, panel.querySelector('.lq-inputrow'));
        inputEl.disabled = true;
        sendBtn.disabled = true;
      }
    }
  }

  var sessionPromise = null;
  function ensureSession(trigger) {
    if (sessionPromise) return sessionPromise;
    sessionPromise = fetchJSON('/api/widget/session', {
      method: 'POST',
      body: JSON.stringify({
        key: API_KEY,
        session_token: sessionToken,
        visitor_id: visitorId,
        page_url: location.href,
        trigger: trigger || 'open'
      })
    }).then(function (res) {
      if (res.disabled) {
        addMessage({ role: 'assistant', content: 'Chat is temporarily unavailable. Please check back soon.' });
        inputEl.disabled = true; sendBtn.disabled = true;
        return;
      }
      if (res.error) { sessionPromise = null; return; }
      sessionToken = res.session_token;
      localStorage.setItem(LS_SESSION, sessionToken);
      (res.messages || []).forEach(addMessage);
      if (res.status) setStatus(res.status);
      connectWS();
    }).catch(function () { sessionPromise = null; });
    return sessionPromise;
  }

  function connectWS() {
    if (!sessionToken || ws) return;
    if (!('WebSocket' in window) || wsFails >= 3) { startPolling(); return; }
    try {
      ws = new WebSocket(wsURL + '?session_token=' + encodeURIComponent(sessionToken));
    } catch (e) { ws = null; startPolling(); return; }
    ws.onopen = function () { wsFails = 0; stopPolling(); };
    ws.onmessage = function (ev) {
      var data;
      try { data = JSON.parse(ev.data); } catch (e) { return; }
      if (data.type === 'message') addMessage(data.message);
      else if (data.type === 'typing') showTyping();
      else if (data.type === 'status') setStatus(data.status);
    };
    ws.onclose = function () {
      ws = null;
      wsFails++;
      if (ended) return;
      if (wsFails >= 3) startPolling();
      else setTimeout(connectWS, 1000 * wsFails);
    };
    ws.onerror = function () { try { ws && ws.close(); } catch (e) { } };
  }

  function teardownRealtime() {
    try { ws && ws.close(); } catch (e) { }
    ws = null; wsFails = 0;
    stopPolling();
    sessionPromise = null;
  }

  function startPolling() {
    if (pollTimer || !sessionToken) return;
    pollTimer = setInterval(function () {
      if (!sessionToken) return;
      fetchJSON('/api/widget/poll?session_token=' + encodeURIComponent(sessionToken) + '&after=' + lastMsgId)
        .then(function (res) {
          (res.messages || []).forEach(addMessage);
          if (res.status) setStatus(res.status);
        }).catch(function () { });
      if (!ws && wsFails < 6) connectWS();
    }, 2500);
  }

  function stopPolling() {
    if (pollTimer) { clearInterval(pollTimer); pollTimer = null; }
  }

  function submit() {
    var text = (inputEl.value || '').trim();
    if (!text || ended) return;
    inputEl.value = '';
    inputEl.style.height = 'auto';
    sendText(text);
  }

  function sendText(text) {
    if (ended) return;
    renderQuickReplies(null);
    ensureSession('open').then(function () {
      if (!sessionToken) return;
      addMessage({ role: 'visitor', content: text, id: 0 });
      showTyping();
      if (ws && ws.readyState === 1) {
        ws.send(JSON.stringify({ type: 'message', content: text }));
      } else {
        fetchJSON('/api/widget/message', {
          method: 'POST',
          body: JSON.stringify({ session_token: sessionToken, content: text })
        }).then(function () { if (!ws) startPolling(); }).catch(function () { hideTyping(); });
      }
    });
  }

  function setupTriggers() {
    if (cfg.proactive && cfg.proactive.enabled && !sessionStorage.getItem(SS_PROACTIVE) && !sessionToken) {
      setTimeout(function () {
        if (open || sessionToken || sessionStorage.getItem(SS_PROACTIVE)) return;
        sessionStorage.setItem(SS_PROACTIVE, '1');
        track('proactive_shown');
        openPanel('proactive');
      }, Math.max(2, cfg.proactive.delay_seconds || 8) * 1000);
    }
    if (cfg.exit_intent && !('ontouchstart' in window)) {
      document.addEventListener('mouseout', function (e) {
        if (e.clientY > 8 || e.relatedTarget) return;
        if (open || sessionStorage.getItem(SS_EXIT)) return;
        sessionStorage.setItem(SS_EXIT, '1');
        track('exit_shown');
        openPanel('exit');
      });
    }
  }

  function init() {
    fetchJSON('/api/widget/boot?url=' + encodeURIComponent(location.href) + (API_KEY ? '&key=' + encodeURIComponent(API_KEY) : ''))
      .then(function (res) {
        if (!res || res.error) return;
        cfg = res.config || {};
        wsURL = res.ws_url || '';
        if (res.disabled && !sessionToken) return;
        if (!pageAllowed(cfg.pages)) return;
        build();
        if (!sessionStorage.getItem(SS_LOADED)) {
          sessionStorage.setItem(SS_LOADED, '1');
          track('loaded');
        }
        if (sessionToken) {
          ensureSession('open').then(function () {
            unread = 0;
            renderBadge();
            if (!ended) connectWS();
          });
        }
        setupTriggers();
      })
      .catch(function (e) { console.warn('[leadqualifier] boot failed', e); });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
