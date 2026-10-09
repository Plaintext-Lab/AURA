import axios from "axios";

const apiClient = axios.create({
  baseURL: "/api",
  timeout: 3000000,
  withCredentials: true, // send/receive the aura_session HttpOnly cookie
  headers: {
    "Content-Type": "application/json",
  },
});

// Auto redirect on 401 - session cookie missing/expired
apiClient.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err?.response?.status === 401 && typeof window !== "undefined") {
      if (!window.location.pathname.startsWith("/login")) {
        window.location.href = "/login";
        return new Promise(() => {});
      }
    }
    return Promise.reject(err);
  }
);

export default apiClient;
