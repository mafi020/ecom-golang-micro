import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  stages: [
    { duration: "10s", target: 20 }, // Ramp up from 0 to 20 concurrent users
    { duration: "30s", target: 50 }, // Stay at 50 concurrent users (Stress phase)
    { duration: "10s", target: 0 }, // Ramp down back to 0 users
  ],
  thresholds: {
    http_req_failed: ["rate<0.01"], // Error rate must stay below 1%
    http_req_duration: ["p(95)<500"], // 95% of requests must respond under 200ms
  },
};

export default function () {
  const url = "http://localhost:8000/api/products";

  const params = {
    headers: {
      "Content-Type": "application/json",
      "X-Bypass-Rate-Limit": "true",
    },
  };

  const res = http.get(url, params);

  // ── ADD THIS DEBUGGING BLOCK TO CATCH THE ERROR ──
  if (res.status !== 200) {
    console.log(
      `❌ Fail! Status: ${res.status} | Error Message: ${res.body ? res.body.trim() : "Empty Body"}`,
    );
  }

  check(res, {
    "status is 200": (r) => r.status === 200,
  });

  sleep(1);
}
