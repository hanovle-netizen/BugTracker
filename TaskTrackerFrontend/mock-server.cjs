const http = require('http');

const PORT = 3002;

// Fake data
const users = [
  { id: 1, email: 'admin@test.com', role: 'admin' },
  { id: 2, email: 'dev@test.com',   role: 'developer' },
  { id: 3, email: 'qa@test.com',    role: 'qa' },
];

const orgs = [
  { id: 1, name: 'Тестовая организация', role: 'owner' },
  { id: 2, name: 'Вторая org',           role: 'member' },
];

const projects = [
  { id: 1, org_id: 1, name: 'Проект Alpha', role: 'pm' },
  { id: 2, org_id: 1, name: 'Проект Beta',  role: 'dev' },
  { id: 3, org_id: 2, name: 'Проект Gamma', role: 'qa' },
];

const tasks = [
  { id: 1, title: 'Задача 1: Настроить CI/CD',        description: 'Описание задачи 1', project_id: 1, owner_id: 1, created_at: '2025-01-01T10:00:00Z', updated_at: '2025-01-05T12:00:00Z' },
  { id: 2, title: 'Задача 2: Написать тесты',          description: 'Описание задачи 2', project_id: 1, owner_id: 2, created_at: '2025-01-02T10:00:00Z', updated_at: '2025-01-06T12:00:00Z' },
  { id: 3, title: 'Задача 3: Рефакторинг авторизации', description: 'Описание задачи 3', project_id: 2, owner_id: 1, created_at: '2025-01-03T10:00:00Z', updated_at: '2025-01-07T12:00:00Z' },
];

const bugs = [
  { id: 1, task_id: 1, status: 'Open',       severity: 'Critical', priority: 'High',   title: 'Баг: кнопка не работает',     description: 'При нажатии ничего не происходит', created_by: 1, created_at: '2025-01-10T10:00:00Z' },
  { id: 2, task_id: 1, status: 'In Progress', severity: 'Major',    priority: 'Medium', title: 'Баг: неверный цвет кнопки',   description: 'Цвет не совпадает с макетом',      created_by: 2, created_at: '2025-01-11T10:00:00Z' },
  { id: 3, task_id: 1, status: 'Fixed',       severity: 'Minor',    priority: 'Low',    title: 'Баг: опечатка в заголовке',   description: 'Написано "Зоголовок"',             created_by: 1, created_at: '2025-01-12T10:00:00Z' },
  { id: 4, task_id: 2, status: 'New',         severity: 'Blocker',  priority: 'High',   title: 'Баг: приложение падает',      description: 'При загрузке страницы краш',       created_by: 3, created_at: '2025-01-13T10:00:00Z' },
];

const comments = [
  { id: 1, bug_id: 1, user_id: 1, body: 'Воспроизвожу на Chrome 120', created_at: '2025-01-10T11:00:00Z' },
  { id: 2, bug_id: 1, user_id: 2, body: 'Проверил — баг стабильный',   created_at: '2025-01-10T12:00:00Z' },
];

const audit = [
  { id: 1, bug_id: 1, user_id: 1, field: 'status',   old_value: 'New',    new_value: 'Open',        changed_at: '2025-01-10T11:30:00Z' },
  { id: 2, bug_id: 1, user_id: 2, field: 'priority', old_value: 'Medium', new_value: 'High',        changed_at: '2025-01-10T12:30:00Z' },
];

const relations = [
  { id: 1, bug_id_a: 1, bug_id_b: 2, type: 'related' },
];

const templates = [
  { id: 1, name: 'Шаблон: UI баг',      body: 'Шаги воспроизведения:\n1. \n2. \n\nОжидаемый результат:\nФактический результат:', created_by: 1, created_at: '2025-01-01T10:00:00Z' },
  { id: 2, name: 'Шаблон: Backend баг', body: 'Endpoint:\nМетод:\nТело запроса:\nОтвет сервера:', created_by: 1, created_at: '2025-01-01T10:00:00Z' },
];

const chatThreads = [
  { id: 1, scope: 'dm',      peer_email: 'dev@test.com',          last_message: 'Привет!',         last_message_at: '2025-01-10T10:00:00Z', unread_count: 2 },
  { id: 2, scope: 'org',     title: 'Тестовая организация',        last_message: 'Всем привет',     last_message_at: '2025-01-09T10:00:00Z', unread_count: 0 },
  { id: 3, scope: 'project', title: 'Проект Alpha',                last_message: 'Задача готова',   last_message_at: '2025-01-08T10:00:00Z', unread_count: 1 },
];

const chatMessages = [
  { id: 1, thread_id: 1, user_id: 2, user_email: 'dev@test.com',   body: 'Привет!',          created_at: '2025-01-10T09:00:00Z', edited_at: '', deleted_at: '' },
  { id: 2, thread_id: 1, user_id: 1, user_email: 'admin@test.com', body: 'Привет, как дела?', created_at: '2025-01-10T09:01:00Z', edited_at: '', deleted_at: '' },
  { id: 3, thread_id: 1, user_id: 2, user_email: 'dev@test.com',   body: 'Всё ок, работаю над задачей', created_at: '2025-01-10T09:02:00Z', edited_at: '', deleted_at: '' },
];

const orgMembers = [
  { user_id: 1, email: 'admin@test.com', role: 'owner' },
  { user_id: 2, email: 'dev@test.com',   role: 'admin' },
  { user_id: 3, email: 'qa@test.com',    role: 'member' },
];

const projectMembers = [
  { user_id: 1, email: 'admin@test.com', role: 'pm' },
  { user_id: 2, email: 'dev@test.com',   role: 'dev' },
  { user_id: 3, email: 'qa@test.com',    role: 'qa' },
];

const stats = [
  { status: 'New',              count: 5 },
  { status: 'Open',             count: 8 },
  { status: 'In Progress',      count: 4 },
  { status: 'Fixed',            count: 12 },
  { status: 'Ready for Retest', count: 3 },
  { status: 'Verified',         count: 7 },
  { status: 'Reopened',         count: 2 },
  { status: 'Rejected',         count: 1 },
];

// ---- helpers ----
function json(res, data, status = 200) {
  res.writeHead(status, {
    'Content-Type': 'application/json',
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET,POST,PATCH,PUT,DELETE,OPTIONS',
    'Access-Control-Allow-Headers': 'Authorization,Content-Type',
  });
  res.end(JSON.stringify(data));
}

function readBody(req) {
  return new Promise(resolve => {
    let body = '';
    req.on('data', c => body += c);
    req.on('end', () => {
      try { resolve(JSON.parse(body)); } catch { resolve({}); }
    });
  });
}

function match(path, pattern) {
  const re = new RegExp('^' + pattern.replace(/\{[^}]+\}/g, '([^/]+)') + '$');
  const m = path.match(re);
  return m ? m.slice(1) : null;
}

// ---- router ----
const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, `http://localhost:${PORT}`);
  const path = url.pathname;
  const method = req.method;

  // preflight
  if (method === 'OPTIONS') return json(res, {});

  console.log(`${method} ${path}`);

  // AUTH
  if (method === 'POST' && path === '/login') {
    return json(res, { id: 1, token: 'mock-jwt-token', role: 'admin' });
  }
  if (method === 'POST' && path === '/users') {
    return json(res, { id: 1, token: 'mock-jwt-token', role: 'qa' });
  }

  // ME
  if (method === 'GET' && path === '/me') {
    return json(res, users[0]);
  }
  if (method === 'PATCH' && path === '/me/email') {
    return json(res, { ok: true });
  }
  if (method === 'PATCH' && path === '/me/password') {
    return json(res, { ok: true });
  }
  if (method === 'POST' && path === '/me/logout-all') {
    return json(res, { ok: true });
  }

  // USERS
  let m;
  if (method === 'GET' && (m = match(path, '/users/{id}'))) {
    const u = users.find(u => u.id === Number(m[0])) || users[0];
    return json(res, u);
  }

  // ORGS
  if (method === 'GET' && path === '/orgs') {
    return json(res, orgs);
  }
  if (method === 'POST' && path === '/orgs') {
    const body = await readBody(req);
    return json(res, { id: 99, name: body.name || 'Новая org' });
  }
  if (method === 'GET' && (m = match(path, '/orgs/{id}/members'))) {
    return json(res, orgMembers);
  }
  if (method === 'POST' && (m = match(path, '/orgs/{id}/members'))) {
    return json(res, { ok: true });
  }
  if (method === 'PATCH' && (m = match(path, '/orgs/{id}/members/{userId}'))) {
    return json(res, { ok: true });
  }
  if (method === 'DELETE' && (m = match(path, '/orgs/{id}/members/{userId}'))) {
    return json(res, { ok: true });
  }

  // PROJECTS
  if (method === 'GET' && path === '/projects') {
    const orgId = Number(url.searchParams.get('org_id'));
    return json(res, orgId ? projects.filter(p => p.org_id === orgId) : projects);
  }
  if (method === 'POST' && path === '/projects') {
    const body = await readBody(req);
    return json(res, { id: 99, name: body.name || 'Новый проект' });
  }
  if (method === 'GET' && (m = match(path, '/projects/{id}/members'))) {
    return json(res, projectMembers);
  }
  if (method === 'POST' && (m = match(path, '/projects/{id}/members'))) {
    return json(res, { ok: true });
  }
  if (method === 'PATCH' && (m = match(path, '/projects/{id}/members/{userId}'))) {
    return json(res, { ok: true });
  }
  if (method === 'DELETE' && (m = match(path, '/projects/{id}/members/{userId}'))) {
    return json(res, { ok: true });
  }

  // TASKS
  if (method === 'GET' && path === '/tasks') {
    const projectId = Number(url.searchParams.get('project_id'));
    return json(res, projectId ? tasks.filter(t => t.project_id === projectId) : tasks);
  }
  if (method === 'POST' && path === '/tasks') {
    const body = await readBody(req);
    return json(res, { id: 99, title: body.title || 'Новая задача' });
  }
  if (method === 'DELETE' && (m = match(path, '/tasks/{id}'))) {
    return json(res, { ok: true });
  }

  // BUGS
  if (method === 'GET' && (m = match(path, '/bugs/{id}/photo'))) {
    res.writeHead(404, { 'Access-Control-Allow-Origin': '*' });
    return res.end('no photo');
  }
  if (method === 'POST' && (m = match(path, '/bugs/{id}/photo'))) {
    return json(res, { ok: true });
  }
  if (method === 'GET' && (m = match(path, '/bugs/{id}/comments'))) {
    return json(res, comments.filter(c => c.bug_id === Number(m[0])));
  }
  if (method === 'POST' && (m = match(path, '/bugs/{id}/comments'))) {
    return json(res, { id: 99 });
  }
  if (method === 'GET' && (m = match(path, '/bugs/{id}/audit'))) {
    return json(res, audit.filter(a => a.bug_id === Number(m[0])));
  }
  if (method === 'GET' && (m = match(path, '/bugs/{id}/relations'))) {
    return json(res, relations.filter(r => r.bug_id_a === Number(m[0]) || r.bug_id_b === Number(m[0])));
  }
  if (method === 'POST' && (m = match(path, '/bugs/{id}/relations'))) {
    return json(res, { id: 99 });
  }
  if (method === 'DELETE' && (m = match(path, '/relations/{relId}'))) {
    return json(res, { ok: true });
  }
  if (method === 'GET' && (m = match(path, '/bugs/{id}/tags'))) {
    return json(res, ['ui', 'critical']);
  }
  if (method === 'PUT' && (m = match(path, '/bugs/{id}/tags'))) {
    return json(res, { ok: true });
  }
  if (method === 'GET' && (m = match(path, '/bugs/{id}'))) {
    return json(res, bugs.filter(b => b.task_id === Number(m[0])));
  }
  if (method === 'POST' && (m = match(path, '/bugs/{id}'))) {
    return json(res, { id: 99 });
  }
  if (method === 'PATCH' && (m = match(path, '/bugs/{id}'))) {
    return json(res, { ok: true });
  }
  if (method === 'DELETE' && (m = match(path, '/bugs/{id}'))) {
    return json(res, { ok: true });
  }

  // TEMPLATES
  if (method === 'GET' && path === '/templates') {
    return json(res, templates);
  }
  if (method === 'POST' && path === '/templates') {
    return json(res, { id: 99 });
  }
  if (method === 'DELETE' && (m = match(path, '/templates/{id}'))) {
    return json(res, { ok: true });
  }

  // STATS
  if (method === 'GET' && path === '/stats') {
    return json(res, stats);
  }
  if (method === 'GET' && (m = match(path, '/stats/tasks/{id}'))) {
    return json(res, stats);
  }

  // CHAT
  if (method === 'GET' && path === '/chat/threads') {
    const scope = url.searchParams.get('scope');
    return json(res, scope ? chatThreads.filter(t => t.scope === scope) : chatThreads);
  }
  if (method === 'POST' && path === '/chat/threads') {
    return json(res, { id: chatThreads[0].id });
  }
  if (method === 'GET' && (m = match(path, '/chat/threads/{id}/messages'))) {
    const threadId = Number(m[0]);
    return json(res, chatMessages.filter(msg => msg.thread_id === threadId));
  }
  if (method === 'POST' && (m = match(path, '/chat/threads/{id}/messages'))) {
    return json(res, { id: 99 });
  }
  if (method === 'POST' && (m = match(path, '/chat/threads/{id}/read'))) {
    return json(res, { ok: true });
  }
  if (method === 'GET' && (m = match(path, '/chat/threads/{id}/typing'))) {
    return json(res, []);
  }
  if (method === 'POST' && (m = match(path, '/chat/threads/{id}/typing'))) {
    return json(res, { ok: true });
  }
  if (method === 'PATCH' && (m = match(path, '/chat/messages/{id}'))) {
    return json(res, { ok: true });
  }
  if (method === 'DELETE' && (m = match(path, '/chat/messages/{id}'))) {
    return json(res, { ok: true });
  }

  // HEALTHZ
  if (method === 'GET' && path === '/healthz') {
    return json(res, { status: 'ok' });
  }

  // 404
  console.warn(`  → 404 not handled`);
  json(res, { error: 'not found' }, 404);
});

server.listen(PORT, () => {
  console.log(`Mock API server running at http://localhost:${PORT}`);
  console.log('Login with any email/password — token is always accepted');
});
