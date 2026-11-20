const state = {
  token: localStorage.getItem("accessToken") || "",
  user: null,
  page: 1,
  q: "",
  detailId: 0,
};

const statusEl = document.querySelector("#status");
const whoEl = document.querySelector("#who");
const postsEl = document.querySelector("#posts");
const pageEl = document.querySelector("#page");
const detailEl = document.querySelector("#detail");
const answerEl = document.querySelector("#answer");
const sourcesEl = document.querySelector("#sources");
const logoutBtn = document.querySelector("#logout");

function setStatus(code, text) {
  statusEl.textContent = code ? code + " " + text : text;
}

async function api(path, options) {
  const opts = options || {};
  const headers = { "Content-Type": "application/json" };
  if (state.token) headers.Authorization = "Bearer " + state.token;
  const res = await fetch(path, {
    method: opts.method || "GET",
    headers,
    body: opts.body ? JSON.stringify(opts.body) : undefined,
  });
  if (res.status === 204) return { status: 204 };
  const payload = await res.json().catch(function () { return {}; });
  if (!res.ok) {
    const fields = payload.error && payload.error.fields ? JSON.stringify(payload.error.fields) : "";
    const message = (payload.error && payload.error.message) || "요청에 실패했습니다.";
    throw Object.assign(new Error(fields ? message + " " + fields : message), { status: res.status });
  }
  return { status: res.status, data: payload.data, meta: payload.meta };
}

function formData(form) {
  const data = {};
  new FormData(form).forEach(function (value, key) { data[key] = String(value); });
  return data;
}

async function withButton(button, work) {
  button.disabled = true;
  try {
    await work();
  } catch (err) {
    setStatus(err.status || 0, err.message);
  } finally {
    button.disabled = false;
  }
}

async function refreshMe() {
  logoutBtn.hidden = !state.token;
  if (!state.token) {
    state.user = null;
    whoEl.textContent = "로그인 전";
    return;
  }
  try {
    const res = await api("/api/auth/me");
    state.user = res.data;
    whoEl.textContent = res.data.nickname + " (" + res.data.email + ")";
  } catch (err) {
    state.token = "";
    localStorage.removeItem("accessToken");
    whoEl.textContent = "로그인이 만료되었습니다.";
  }
}

async function loadPosts() {
  const res = await api("/api/posts?page=" + state.page + "&limit=10&q=" + encodeURIComponent(state.q));
  postsEl.replaceChildren();
  (res.data || []).forEach(function (post) {
    const li = document.createElement("li");
    const button = document.createElement("button");
    button.type = "button";
    button.className = "link";
    button.textContent = post.title + " · " + post.author.nickname;
    button.addEventListener("click", function () { loadDetail(post.id); });
    li.append(button);
    postsEl.append(li);
  });
  const meta = res.meta || { page: 1, totalPages: 1, total: 0 };
  pageEl.textContent = meta.page + " / " + Math.max(meta.totalPages, 1) + " · 전체 " + meta.total;
}

async function loadDetail(id) {
  state.detailId = id;
  const res = await api("/api/posts/" + id);
  const post = res.data;
  detailEl.replaceChildren();
  const title = document.createElement("h3");
  title.textContent = post.title;
  detailEl.append(title);
  const body = document.createElement("p");
  body.textContent = post.body;
  const meta = document.createElement("p");
  meta.textContent = post.author.nickname + " · 댓글 " + post.commentCount;
  detailEl.append(body, meta);
  const comments = await api("/api/posts/" + id + "/comments");
  const list = document.createElement("ul");
  (comments.data || []).forEach(function (comment) {
    const li = document.createElement("li");
    li.textContent = comment.author.nickname + ": " + comment.body;
    list.append(li);
  });
  detailEl.append(list);
}

document.querySelector("#login").addEventListener("submit", function (event) {
  event.preventDefault();
  const button = event.target.querySelector("button");
  const data = formData(event.target);
  withButton(button, async function () {
    const res = await api("/api/auth/login", { method: "POST", body: data });
    state.token = res.data.accessToken;
    localStorage.setItem("accessToken", state.token);
    setStatus(res.status, "로그인했습니다.");
    await refreshMe();
    await loadPosts();
  });
});

document.querySelector("#signup").addEventListener("submit", function (event) {
  event.preventDefault();
  const button = event.target.querySelector("button");
  const data = formData(event.target);
  withButton(button, async function () {
    const created = await api("/api/auth/signup", { method: "POST", body: data });
    const res = await api("/api/auth/login", { method: "POST", body: { email: data.email, password: data.password } });
    state.token = res.data.accessToken;
    localStorage.setItem("accessToken", state.token);
    setStatus(created.status, "가입 후 로그인했습니다.");
    await refreshMe();
  });
});

logoutBtn.addEventListener("click", function () {
  withButton(logoutBtn, async function () {
    await api("/api/auth/logout", { method: "POST" });
    state.token = "";
    localStorage.removeItem("accessToken");
    setStatus(204, "로그아웃했습니다. 토큰은 이 브라우저에서 지웠습니다.");
    await refreshMe();
  });
});

document.querySelector("#search").addEventListener("submit", function (event) {
  event.preventDefault();
  state.q = formData(event.target).q || "";
  state.page = 1;
  withButton(event.target.querySelector("button"), loadPosts);
});

document.querySelector("#prev").addEventListener("click", function () {
  if (state.page > 1) state.page -= 1;
  loadPosts().catch(function (err) { setStatus(err.status || 0, err.message); });
});
document.querySelector("#next").addEventListener("click", function () {
  state.page += 1;
  loadPosts().catch(function (err) { setStatus(err.status || 0, err.message); });
});

document.querySelector("#writer").addEventListener("submit", function (event) {
  event.preventDefault();
  const button = event.target.querySelector("button");
  const data = formData(event.target);
  withButton(button, async function () {
    const res = await api("/api/posts", { method: "POST", body: { title: data.title, body: data.body, tags: [] } });
    setStatus(res.status, "글을 등록했습니다.");
    event.target.reset();
    state.page = 1;
    await loadPosts();
    await loadDetail(res.data.id);
  });
});

document.querySelector("#comment").addEventListener("submit", function (event) {
  event.preventDefault();
  const button = event.target.querySelector("button");
  const data = formData(event.target);
  withButton(button, async function () {
    if (!state.detailId) throw Object.assign(new Error("글을 먼저 여세요."), { status: 400 });
    const res = await api("/api/posts/" + state.detailId + "/comments", { method: "POST", body: data });
    setStatus(res.status, "댓글을 등록했습니다.");
    event.target.reset();
    await loadDetail(state.detailId);
    await loadPosts();
  });
});

document.querySelector("#doc").addEventListener("submit", function (event) {
  event.preventDefault();
  const button = event.target.querySelector("button");
  const data = formData(event.target);
  withButton(button, async function () {
    const created = await api("/api/documents", { method: "POST", body: { title: data.title, content: data.content } });
    const asked = await api("/api/documents/" + created.data.id + "/questions", {
      method: "POST",
      body: { question: data.question },
    });
    answerEl.textContent = asked.data.answer;
    sourcesEl.replaceChildren();
    (asked.data.sources || []).forEach(function (source) {
      const li = document.createElement("li");
      li.textContent = "발췌 " + source.index + " (" + source.score + ") " + source.text;
      sourcesEl.append(li);
    });
    setStatus(asked.status, "문서 질문을 저장했습니다.");
  });
});

refreshMe().then(loadPosts).catch(function (err) { setStatus(err.status || 0, err.message); });
