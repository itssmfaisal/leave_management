async function request(method, path, body) {
  const res = await fetch(`/api${path}`, {
    method,
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (res.status === 204) return null;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || `Request failed (${res.status})`);
  return data;
}

export const api = {
  getProfile: () => request("GET", "/profile"),
  saveProfile: (profile) => request("PUT", "/profile", profile),
  getYear: (year) => request("GET", `/years/${year}`),
  setOpeningUsed: (year, used) => request("PUT", `/years/${year}/opening`, { used }),
  listLeaves: () => request("GET", "/leaves"),
  createLeave: (leave) => request("POST", "/leaves", leave),
  updateLeave: (id, leave) => request("PUT", `/leaves/${id}`, leave),
  deleteLeave: (id) => request("DELETE", `/leaves/${id}`),
};
